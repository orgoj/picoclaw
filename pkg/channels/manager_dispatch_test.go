package channels

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
)

type failingDispatchChannel struct {
	err error
}

func (f *failingDispatchChannel) Name() string { return "failing" }
func (f *failingDispatchChannel) Start(context.Context) error {
	return nil
}
func (f *failingDispatchChannel) Stop(context.Context) error {
	return nil
}
func (f *failingDispatchChannel) Send(context.Context, bus.OutboundMessage) error {
	return f.err
}
func (f *failingDispatchChannel) IsRunning() bool { return true }
func (f *failingDispatchChannel) IsAllowed(string) bool {
	return true
}

func TestDispatchOutbound_SendFailurePublishesInboundNotice(t *testing.T) {
	mb := bus.NewMessageBus()
	m := &Manager{
		bus: mb,
		channels: map[string]Channel{
			"telegram": &failingDispatchChannel{err: errors.New("boom")},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go m.dispatchOutbound(ctx)

	if ok := mb.PublishOutbound(bus.OutboundMessage{
		Channel: "telegram",
		ChatID:  "42",
		Content: "hello world",
	}); !ok {
		t.Fatal("failed to enqueue outbound message")
	}

	inboundCtx, inboundCancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer inboundCancel()
	in, ok := mb.ConsumeInbound(inboundCtx)
	if !ok {
		t.Fatal("expected delivery failure inbound notice")
	}
	if in.Channel != "telegram" || in.ChatID != "42" || in.SessionKey != "telegram:42" {
		t.Fatalf("unexpected routing: %+v", in)
	}
	if in.Metadata["source"] != "system:delivery_failure" || in.Metadata["urgent"] != "true" {
		t.Fatalf("unexpected metadata: %+v", in.Metadata)
	}
	if !strings.Contains(in.Content, "<delivery_failure") || !strings.Contains(in.Content, "boom") {
		t.Fatalf("unexpected failure content: %q", in.Content)
	}
}

func TestNotifyDeliveryFailure_SkipsRecursiveMarker(t *testing.T) {
	mb := bus.NewMessageBus()
	m := &Manager{bus: mb}

	m.notifyDeliveryFailure(bus.OutboundMessage{
		Channel: "telegram",
		ChatID:  "42",
		Content: "<delivery_failure channel=\"telegram\">",
	}, errors.New("still failing"))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, ok := mb.ConsumeInbound(ctx); ok {
		t.Fatal("did not expect recursive delivery failure notice")
	}
}
