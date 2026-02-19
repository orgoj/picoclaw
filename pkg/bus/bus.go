package bus

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type MessageBus struct {
	inboundMu    sync.Mutex
	inboundQueue []*InboundQueueItem
	inboundCap   int
	inboundSeq   atomic.Uint64
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

func (mb *MessageBus) PublishInboundWithID(msg InboundMessage) (string, bool) {
	return mb.publishInboundWithIDTimeout(msg, 2*time.Second)
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
		return true
	case <-timer.C:
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

	id := fmt.Sprintf("in-%06d", mb.inboundSeq.Add(1))
	mb.inboundQueue = append(mb.inboundQueue, &InboundQueueItem{
		ID:         id,
		Message:    msg,
		EnqueuedAt: time.Now().UnixMilli(),
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
		return item.Message, true, false
	}
	if mb.closed {
		return InboundMessage{}, false, true
	}
	return InboundMessage{}, false, false
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
