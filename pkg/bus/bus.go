package bus

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sipeed/picoclaw/pkg/audit"
)

type MessageBus struct {
	inboundMu    sync.Mutex
	inboundQueue []*InboundQueueItem
	inboundCap   int
	inboundSeq   atomic.Uint64
	mergeWindow  time.Duration
	gapNotice    time.Duration
	concatPrefix string
	lastInbound  map[string]int64
	closed       bool
	inboundReady chan struct{}
	inboundSpace chan struct{}

	outbound chan OutboundMessage
	handlers map[string]MessageHandler
	mu       sync.RWMutex
}

func NewMessageBus() *MessageBus {
	mb := &MessageBus{
		inboundQueue: make([]*InboundQueueItem, 0, 100),
		inboundCap:   100,
		mergeWindow:  3 * time.Second,
		gapNotice:    10 * time.Minute,
		concatPrefix: "+",
		lastInbound:  make(map[string]int64),
		inboundReady: make(chan struct{}, 1),
		inboundSpace: make(chan struct{}, 1),
		outbound:     make(chan OutboundMessage, 100),
		handlers:     make(map[string]MessageHandler),
	}
	return mb
}

func (mb *MessageBus) PublishInbound(msg InboundMessage) bool {
	return mb.publishInboundWithTimeout(msg, 2*time.Second)
}

// PublishInboundCritical ensures critical inbound messages are enqueued even when
// the queue is full by dropping the oldest queued item.
func (mb *MessageBus) PublishInboundCritical(msg InboundMessage) bool {
	mb.inboundMu.Lock()
	defer mb.inboundMu.Unlock()

	if mb.closed {
		return false
	}
	if len(mb.inboundQueue) >= mb.inboundCap && len(mb.inboundQueue) > 0 {
		mb.inboundQueue = mb.inboundQueue[1:]
	}

	id := fmt.Sprintf("in-%06d", mb.inboundSeq.Add(1))
	if msg.Metadata == nil {
		msg.Metadata = map[string]string{}
	}
	msg.Metadata["enqueued_at"] = strconv.FormatInt(time.Now().UnixMilli(), 10)
	mb.inboundQueue = append(mb.inboundQueue, &InboundQueueItem{
		ID:         id,
		Message:    msg,
		EnqueuedAt: time.Now().UnixMilli(),
	})
	audit.Record("inbound_enqueue_critical", map[string]interface{}{
		"id":          id,
		"session_key": msg.SessionKey,
		"channel":     msg.Channel,
		"chat_id":     msg.ChatID,
		"sender_id":   msg.SenderID,
	})
	mb.notifyInboundReady()
	return true
}

func (mb *MessageBus) PublishInboundWithID(msg InboundMessage) (string, bool) {
	return mb.publishInboundWithIDTimeout(msg, 2*time.Second)
}

func (mb *MessageBus) InsertInboundFirst(msg InboundMessage) (string, bool) {
	return mb.insertInboundFirstWithIDTimeout(msg, 2*time.Second)
}

func (mb *MessageBus) ConfigureIngress(mergeWindow, gapNotice time.Duration, concatPrefix string) {
	mb.inboundMu.Lock()
	defer mb.inboundMu.Unlock()
	if mergeWindow < 0 {
		mergeWindow = 0
	}
	if gapNotice < 0 {
		gapNotice = 0
	}
	mb.mergeWindow = mergeWindow
	mb.gapNotice = gapNotice
	mb.concatPrefix = strings.TrimSpace(concatPrefix)
}

func (mb *MessageBus) publishInboundWithTimeout(msg InboundMessage, timeout time.Duration) bool {
	_, ok := mb.publishInboundWithIDTimeout(msg, timeout)
	return ok
}

func (mb *MessageBus) publishInboundWithIDTimeout(msg InboundMessage, timeout time.Duration) (string, bool) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		id, ok, needWait := mb.tryEnqueueInbound(msg)
		if ok {
			return id, true
		}
		if !needWait {
			return "", false
		}
		select {
		case <-mb.inboundSpace:
			continue
		case <-timer.C:
			return "", false
		}
	}
}

func (mb *MessageBus) insertInboundFirstWithIDTimeout(msg InboundMessage, timeout time.Duration) (string, bool) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		id, ok, needWait := mb.tryEnqueueInboundAtHead(msg)
		if ok {
			return id, true
		}
		if !needWait {
			return "", false
		}
		select {
		case <-mb.inboundSpace:
			continue
		case <-timer.C:
			return "", false
		}
	}
}

func (mb *MessageBus) ConsumeInbound(ctx context.Context) (InboundMessage, bool) {
	for {
		msg, ok, closed := mb.tryDequeueInbound()
		if ok {
			return msg, true
		}
		if closed {
			return InboundMessage{}, false
		}
		select {
		case <-mb.inboundReady:
			continue
		case <-ctx.Done():
			return InboundMessage{}, false
		}
	}
}

// ConsumeInboundWithTimeout attempts to consume a message with a timeout.
// Returns: (message, gotMessage, timedOut)
// If context cancelled: gotMessage=false, timedOut=false
// If timeout expires:   gotMessage=false, timedOut=true
// If message received:  gotMessage=true,  timedOut=false
func (mb *MessageBus) ConsumeInboundWithTimeout(ctx context.Context, timeout time.Duration) (InboundMessage, bool, bool) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		msg, ok, closed := mb.tryDequeueInbound()
		if ok {
			return msg, true, false
		}
		if closed {
			return InboundMessage{}, false, false
		}
		select {
		case <-mb.inboundReady:
			continue
		case <-timer.C:
			return InboundMessage{}, false, true
		case <-ctx.Done():
			return InboundMessage{}, false, false
		}
	}
}

func (mb *MessageBus) PublishOutbound(msg OutboundMessage) bool {
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case mb.outbound <- msg:
		audit.Record("outbound_enqueue", map[string]interface{}{
			"channel": msg.Channel,
			"chat_id": msg.ChatID,
		})
		return true
	case <-timer.C:
		audit.Record("outbound_drop_timeout", map[string]interface{}{
			"channel": msg.Channel,
			"chat_id": msg.ChatID,
		})
		return false
	}
}

func (mb *MessageBus) SubscribeOutbound(ctx context.Context) (OutboundMessage, bool) {
	select {
	case msg := <-mb.outbound:
		return msg, true
	case <-ctx.Done():
		return OutboundMessage{}, false
	}
}

func (mb *MessageBus) RegisterHandler(channel string, handler MessageHandler) {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	mb.handlers[channel] = handler
}

func (mb *MessageBus) GetHandler(channel string) (MessageHandler, bool) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()
	handler, ok := mb.handlers[channel]
	return handler, ok
}

func (mb *MessageBus) Close() {
	mb.inboundMu.Lock()
	mb.closed = true
	mb.inboundQueue = nil
	mb.inboundMu.Unlock()
	mb.notifyInboundReady()
	mb.notifyInboundSpace()
	close(mb.outbound)
}

func (mb *MessageBus) ListInbound() []InboundQueueItem {
	mb.inboundMu.Lock()
	defer mb.inboundMu.Unlock()
	out := make([]InboundQueueItem, 0, len(mb.inboundQueue))
	for _, item := range mb.inboundQueue {
		out = append(out, InboundQueueItem{
			ID:         item.ID,
			Message:    item.Message,
			EnqueuedAt: item.EnqueuedAt,
		})
	}
	return out
}

func (mb *MessageBus) UpdateInboundContent(id, content string) bool {
	mb.inboundMu.Lock()
	defer mb.inboundMu.Unlock()
	for _, item := range mb.inboundQueue {
		if item.ID == id {
			item.Message.Content = content
			audit.Record("inbound_update", map[string]interface{}{
				"id": id,
			})
			return true
		}
	}
	return false
}

func (mb *MessageBus) DeleteInbound(id string) bool {
	mb.inboundMu.Lock()
	defer mb.inboundMu.Unlock()
	for i, item := range mb.inboundQueue {
		if item.ID == id {
			mb.inboundQueue = append(mb.inboundQueue[:i], mb.inboundQueue[i+1:]...)
			mb.notifyInboundSpace()
			audit.Record("inbound_delete", map[string]interface{}{
				"id": id,
			})
			return true
		}
	}
	return false
}

func (mb *MessageBus) MoveInbound(id string, newIndex int) bool {
	mb.inboundMu.Lock()
	defer mb.inboundMu.Unlock()
	if newIndex < 0 || newIndex >= len(mb.inboundQueue) {
		return false
	}
	cur := -1
	for i, item := range mb.inboundQueue {
		if item.ID == id {
			cur = i
			break
		}
	}
	if cur == -1 || cur == newIndex {
		return cur == newIndex
	}

	item := mb.inboundQueue[cur]
	mb.inboundQueue = append(mb.inboundQueue[:cur], mb.inboundQueue[cur+1:]...)
	if newIndex > cur {
		newIndex--
	}
	prefix := append([]*InboundQueueItem{}, mb.inboundQueue[:newIndex]...)
	suffix := append([]*InboundQueueItem{}, mb.inboundQueue[newIndex:]...)
	mb.inboundQueue = append(prefix, item)
	mb.inboundQueue = append(mb.inboundQueue, suffix...)
	mb.notifyInboundReady()
	audit.Record("inbound_move", map[string]interface{}{
		"id":        id,
		"new_index": newIndex,
	})
	return true
}

func (mb *MessageBus) tryEnqueueInbound(msg InboundMessage) (string, bool, bool) {
	mb.inboundMu.Lock()
	defer mb.inboundMu.Unlock()

	if mb.closed {
		return "", false, false
	}
	if len(mb.inboundQueue) >= mb.inboundCap {
		return "", false, true
	}

	mb.enrichInboundMetadata(&msg, len(mb.inboundQueue))
	if mergedID, merged := mb.tryMergeIntoTail(msg); merged {
		mb.notifyInboundReady()
		return mergedID, true, false
	}

	id := fmt.Sprintf("in-%06d", mb.inboundSeq.Add(1))
	if msg.Metadata == nil {
		msg.Metadata = map[string]string{}
	}
	msg.Metadata["enqueued_at"] = strconv.FormatInt(time.Now().UnixMilli(), 10)
	mb.inboundQueue = append(mb.inboundQueue, &InboundQueueItem{
		ID:         id,
		Message:    msg,
		EnqueuedAt: time.Now().UnixMilli(),
	})
	audit.Record("inbound_enqueue_tail", map[string]interface{}{
		"id":          id,
		"session_key": msg.SessionKey,
		"channel":     msg.Channel,
		"chat_id":     msg.ChatID,
		"sender_id":   msg.SenderID,
	})
	mb.notifyInboundReady()
	return id, true, false
}

func (mb *MessageBus) tryEnqueueInboundAtHead(msg InboundMessage) (string, bool, bool) {
	mb.inboundMu.Lock()
	defer mb.inboundMu.Unlock()

	if mb.closed {
		return "", false, false
	}
	if len(mb.inboundQueue) >= mb.inboundCap {
		return "", false, true
	}

	mb.enrichInboundMetadata(&msg, 0)
	if msg.Metadata == nil {
		msg.Metadata = map[string]string{}
	}
	msg.Metadata["queue_insert_mode"] = "head"

	id := fmt.Sprintf("in-%06d", mb.inboundSeq.Add(1))
	msg.Metadata["enqueued_at"] = strconv.FormatInt(time.Now().UnixMilli(), 10)
	item := &InboundQueueItem{
		ID:         id,
		Message:    msg,
		EnqueuedAt: time.Now().UnixMilli(),
	}
	mb.inboundQueue = append([]*InboundQueueItem{item}, mb.inboundQueue...)
	audit.Record("inbound_enqueue_head", map[string]interface{}{
		"id":          id,
		"session_key": msg.SessionKey,
		"channel":     msg.Channel,
		"chat_id":     msg.ChatID,
		"sender_id":   msg.SenderID,
	})
	mb.notifyInboundReady()
	return id, true, false
}

func (mb *MessageBus) tryDequeueInbound() (InboundMessage, bool, bool) {
	mb.inboundMu.Lock()
	defer mb.inboundMu.Unlock()

	if len(mb.inboundQueue) > 0 {
		item := mb.inboundQueue[0]
		mb.inboundQueue = mb.inboundQueue[1:]
		mb.notifyInboundSpace()
		audit.Record("inbound_dequeue", map[string]interface{}{
			"id":          item.ID,
			"session_key": item.Message.SessionKey,
			"channel":     item.Message.Channel,
			"chat_id":     item.Message.ChatID,
			"sender_id":   item.Message.SenderID,
		})
		return item.Message, true, false
	}
	if mb.closed {
		return InboundMessage{}, false, true
	}
	return InboundMessage{}, false, false
}

func (mb *MessageBus) enrichInboundMetadata(msg *InboundMessage, queueLen int) {
	now := time.Now().UnixMilli()
	if msg.Metadata == nil {
		msg.Metadata = map[string]string{}
	}
	if _, ok := msg.Metadata["received_at"]; !ok {
		msg.Metadata["received_at"] = strconv.FormatInt(now, 10)
	}
	msg.Metadata["queue_len_at_enqueue"] = strconv.Itoa(queueLen)

	sessionKey := strings.TrimSpace(msg.SessionKey)
	if sessionKey == "" {
		return
	}
	if prev, ok := mb.lastInbound[sessionKey]; ok {
		delta := now - prev
		msg.Metadata["delta_since_prev_ms"] = strconv.FormatInt(delta, 10)
		if mb.gapNotice > 0 && delta >= mb.gapNotice.Milliseconds() {
			msg.Metadata["gap_notice"] = "true"
		}
	}
	mb.lastInbound[sessionKey] = now
}

func (mb *MessageBus) tryMergeIntoTail(msg InboundMessage) (string, bool) {
	if mb.mergeWindow <= 0 || len(mb.inboundQueue) == 0 {
		return "", false
	}
	tail := mb.inboundQueue[len(mb.inboundQueue)-1]
	if tail == nil {
		return "", false
	}
	if !mb.canMerge(tail, msg) {
		return "", false
	}

	mergedContent := strings.TrimSpace(msg.Content)
	if mb.concatPrefix != "" && strings.HasPrefix(strings.TrimSpace(mergedContent), mb.concatPrefix) {
		mergedContent = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(mergedContent), mb.concatPrefix))
	}
	if mergedContent == "" {
		return "", false
	}

	tail.Message.Content = strings.TrimSpace(tail.Message.Content + "\n" + mergedContent)
	if tail.Message.Metadata == nil {
		tail.Message.Metadata = map[string]string{}
	}
	tail.Message.Metadata["merged"] = "true"
	mergedCount := 1
	if raw, ok := tail.Message.Metadata["merged_count"]; ok {
		if n, err := strconv.Atoi(raw); err == nil {
			mergedCount = n + 1
		}
	}
	tail.Message.Metadata["merged_count"] = strconv.Itoa(mergedCount)
	audit.Record("inbound_merged", map[string]interface{}{
		"id":           tail.ID,
		"session_key":  tail.Message.SessionKey,
		"merged_count": mergedCount,
	})
	return tail.ID, true
}

func (mb *MessageBus) canMerge(tail *InboundQueueItem, msg InboundMessage) bool {
	if tail == nil {
		return false
	}
	if tail.Message.SessionKey == "" || msg.SessionKey == "" || tail.Message.SessionKey != msg.SessionKey {
		return false
	}
	if tail.Message.Channel != msg.Channel || tail.Message.ChatID != msg.ChatID || tail.Message.SenderID != msg.SenderID {
		return false
	}
	if isUrgentMetadata(tail.Message.Metadata) || isUrgentMetadata(msg.Metadata) {
		return false
	}

	content := strings.TrimSpace(msg.Content)
	if mb.concatPrefix != "" && strings.HasPrefix(content, mb.concatPrefix) {
		return true
	}
	return time.Now().UnixMilli()-tail.EnqueuedAt <= mb.mergeWindow.Milliseconds()
}

func isUrgentMetadata(meta map[string]string) bool {
	if len(meta) == 0 {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(meta["urgent"]), "true")
}

func (mb *MessageBus) notifyInboundReady() {
	select {
	case mb.inboundReady <- struct{}{}:
	default:
	}
}

func (mb *MessageBus) notifyInboundSpace() {
	select {
	case mb.inboundSpace <- struct{}{}:
	default:
	}
}
