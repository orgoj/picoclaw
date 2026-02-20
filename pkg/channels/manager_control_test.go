package channels

import (
	"context"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestSplitControlCommand_ConfigurablePrefix(t *testing.T) {
	cmd, body, ok := splitControlCommand("A", "Ainject now")
	if !ok {
		t.Fatal("expected control command to parse")
	}
	if cmd != "inject" || body != "now" {
		t.Fatalf("got (%q,%q), want (inject,now)", cmd, body)
	}
}

func TestHandleInboundControl_EscapePrefix_AppendsLiteralPlus(t *testing.T) {
	mb := bus.NewMessageBus()
	mb.ConfigureIngress(3*time.Second, 10*time.Minute, "A")
	cfg := config.DefaultConfig()
	cfg.Ingress.ConcatPrefix = "A"

	m := &Manager{
		bus:    mb,
		config: cfg,
	}

	if ok := mb.PublishInbound(bus.InboundMessage{
		Channel:    "discord",
		SenderID:   "u1",
		ChatID:     "c1",
		SessionKey: "discord:c1",
		Content:    "older",
	}); !ok {
		t.Fatal("failed to enqueue baseline message")
	}

	handled := m.handleInboundControl(bus.InboundMessage{
		Channel:    "discord",
		SenderID:   "u1",
		ChatID:     "c1",
		SessionKey: "discord:c1",
		Content:    "A+",
	})
	if !handled {
		t.Fatal("expected escaped control prefix to be handled")
	}

	items := mb.ListInbound()
	if len(items) != 1 {
		t.Fatalf("expected 1 inbound item, got %d", len(items))
	}
	if items[0].Message.Content != "older\n+" {
		t.Fatalf("got content %q, want merged older\\n+", items[0].Message.Content)
	}
}

func TestHandleInboundControl_FirstEnqueuesHead(t *testing.T) {
	mb := bus.NewMessageBus()
	cfg := config.DefaultConfig()

	m := &Manager{
		bus:    mb,
		config: cfg,
	}

	if ok := mb.PublishInbound(bus.InboundMessage{
		Channel:    "line",
		SenderID:   "u1",
		ChatID:     "c1",
		SessionKey: "line:c1",
		Content:    "older",
	}); !ok {
		t.Fatal("failed to enqueue baseline message")
	}

	handled := m.handleInboundControl(bus.InboundMessage{
		Channel:    "line",
		SenderID:   "u1",
		ChatID:     "c1",
		SessionKey: "line:c1",
		Content:    "+first newer",
	})
	if !handled {
		t.Fatal("expected +first to be handled")
	}

	items := mb.ListInbound()
	if len(items) < 2 {
		t.Fatalf("expected at least 2 inbound items, got %d", len(items))
	}
	if items[0].Message.Content != "newer" {
		t.Fatalf("head content = %q, want newer", items[0].Message.Content)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, ok := mb.SubscribeOutbound(ctx); !ok {
		t.Fatal("expected +first acknowledgement in outbound queue")
	}
}
