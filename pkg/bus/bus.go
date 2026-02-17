package bus

import (
	"context"
	"sync"
	"time"
)

type MessageBus struct {
	inbound  chan InboundMessage
	outbound chan OutboundMessage
	handlers map[string]MessageHandler
	mu       sync.RWMutex
}

func NewMessageBus() *MessageBus {
	return &MessageBus{
		inbound:  make(chan InboundMessage, 100),
		outbound: make(chan OutboundMessage, 100),
		handlers: make(map[string]MessageHandler),
	}
}

func (mb *MessageBus) PublishInbound(msg InboundMessage) {
	mb.inbound <- msg
}

func (mb *MessageBus) ConsumeInbound(ctx context.Context) (InboundMessage, bool) {
	select {
	case msg := <-mb.inbound:
		return msg, true
	case <-ctx.Done():
		return InboundMessage{}, false
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
	select {
	case msg := <-mb.inbound:
		return msg, true, false
	case <-timer.C:
		return InboundMessage{}, false, true
	case <-ctx.Done():
		return InboundMessage{}, false, false
	}
}

func (mb *MessageBus) PublishOutbound(msg OutboundMessage) {
	mb.outbound <- msg
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
	close(mb.inbound)
	close(mb.outbound)
}
