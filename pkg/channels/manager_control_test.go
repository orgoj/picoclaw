package channels

import (
	"context"
	"strings"
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

func TestHandleInboundControl_HelpHandled(t *testing.T) {
	mb := bus.NewMessageBus()
	cfg := config.DefaultConfig()

	m := &Manager{
		bus:    mb,
		config: cfg,
	}

	handled := m.handleInboundControl(bus.InboundMessage{
		Channel:    "discord",
		SenderID:   "u1",
		ChatID:     "c1",
		SessionKey: "discord:c1",
		Content:    "+help",
	})
	if !handled {
		t.Fatal("expected +help to be handled")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	out, ok := mb.SubscribeOutbound(ctx)
	if !ok {
		t.Fatal("expected +help response")
	}
	if !strings.Contains(out.Content, "+status") {
		t.Fatalf("expected help response to include +status, got %q", out.Content)
	}
}

func TestHandleInboundControl_DeleteRemovesLastSameSession(t *testing.T) {
	mb := bus.NewMessageBus()
	mb.ConfigureIngress(0, 10*time.Minute, "+")
	cfg := config.DefaultConfig()

	m := &Manager{
		bus:    mb,
		config: cfg,
	}

	if _, ok := mb.PublishInboundWithID(bus.InboundMessage{
		Channel:    "line",
		SenderID:   "u1",
		ChatID:     "c1",
		SessionKey: "line:c1",
		Content:    "older",
	}); !ok {
		t.Fatal("failed to enqueue older")
	}
	if _, ok := mb.PublishInboundWithID(bus.InboundMessage{
		Channel:    "line",
		SenderID:   "u1",
		ChatID:     "c1",
		SessionKey: "line:c1",
		Content:    "newer",
	}); !ok {
		t.Fatal("failed to enqueue newer")
	}

	handled := m.handleInboundControl(bus.InboundMessage{
		Channel:    "line",
		SenderID:   "u1",
		ChatID:     "c1",
		SessionKey: "line:c1",
		Content:    "+delete",
	})
	if !handled {
		t.Fatal("expected +delete to be handled")
	}

	items := mb.ListInbound()
	if len(items) != 1 {
		t.Fatalf("expected 1 queued item after delete, got %d", len(items))
	}
	if items[0].Message.Content != "older" {
		t.Fatalf("expected remaining content older, got %q", items[0].Message.Content)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	out, ok := mb.SubscribeOutbound(ctx)
	if !ok {
		t.Fatal("expected +delete acknowledgement")
	}
	if !strings.Contains(out.Content, "Deleted last queued message") {
		t.Fatalf("unexpected +delete reply: %q", out.Content)
	}
}
