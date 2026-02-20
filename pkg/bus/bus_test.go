package bus

import (
	"context"
	"testing"
	"time"
)

func TestInboundQueuePublishConsumeOrder(t *testing.T) {
	mb := NewMessageBus()

	if !mb.PublishInbound(InboundMessage{Content: "a"}) {
		t.Fatal("publish a failed")
	}
	if !mb.PublishInbound(InboundMessage{Content: "b"}) {
		t.Fatal("publish b failed")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	msg, ok := mb.ConsumeInbound(ctx)
	if !ok || msg.Content != "a" {
		t.Fatalf("first consume = (%v,%v), want (a,true)", msg.Content, ok)
	}
	msg, ok = mb.ConsumeInbound(ctx)
	if !ok || msg.Content != "b" {
		t.Fatalf("second consume = (%v,%v), want (b,true)", msg.Content, ok)
	}
}

func TestInboundQueueEditDeleteMove(t *testing.T) {
	mb := NewMessageBus()

	id1, ok := mb.PublishInboundWithID(InboundMessage{Content: "one"})
	if !ok {
		t.Fatal("publish one failed")
	}
	id2, ok := mb.PublishInboundWithID(InboundMessage{Content: "two"})
	if !ok {
		t.Fatal("publish two failed")
	}
	id3, ok := mb.PublishInboundWithID(InboundMessage{Content: "three"})
	if !ok {
		t.Fatal("publish three failed")
	}

	if !mb.UpdateInboundContent(id2, "two-edited") {
		t.Fatal("update inbound content failed")
	}
	if !mb.MoveInbound(id3, 0) {
		t.Fatal("move inbound failed")
	}
	if !mb.DeleteInbound(id1) {
		t.Fatal("delete inbound failed")
	}

	items := mb.ListInbound()
	if len(items) != 2 {
		t.Fatalf("list len=%d, want 2", len(items))
	}
	if items[0].ID != id3 || items[0].Message.Content != "three" {
		t.Fatalf("items[0]=(%s,%s), want (%s,three)", items[0].ID, items[0].Message.Content, id3)
	}
	if items[1].ID != id2 || items[1].Message.Content != "two-edited" {
		t.Fatalf("items[1]=(%s,%s), want (%s,two-edited)", items[1].ID, items[1].Message.Content, id2)
	}
}

func TestConsumeInboundWithTimeout(t *testing.T) {
	mb := NewMessageBus()
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, got, timedOut := mb.ConsumeInboundWithTimeout(ctx, 20*time.Millisecond)
	if got {
		t.Fatal("expected no message")
	}
	if !timedOut {
		t.Fatal("expected timeout")
	}
}

func TestPublishInboundCritical_DropsOldestWhenFull(t *testing.T) {
	mb := NewMessageBus()
	mb.inboundCap = 2

	if !mb.PublishInbound(InboundMessage{Content: "a"}) {
		t.Fatal("publish a failed")
	}
	if !mb.PublishInbound(InboundMessage{Content: "b"}) {
		t.Fatal("publish b failed")
	}
	if !mb.PublishInboundCritical(InboundMessage{Content: "critical"}) {
		t.Fatal("critical publish failed")
	}

	items := mb.ListInbound()
	if len(items) != 2 {
		t.Fatalf("expected queue len 2, got %d", len(items))
	}
	if items[0].Message.Content != "b" {
		t.Fatalf("expected oldest message to be dropped, first item=%q", items[0].Message.Content)
	}
	if items[1].Message.Content != "critical" {
		t.Fatalf("expected critical message at tail, got %q", items[1].Message.Content)
	}
}

func TestInsertInboundFirst(t *testing.T) {
	mb := NewMessageBus()
	mb.ConfigureIngress(0, 10*time.Minute, "+")

	if _, ok := mb.PublishInboundWithID(InboundMessage{Content: "one", SessionKey: "s"}); !ok {
		t.Fatal("publish one failed")
	}
	if _, ok := mb.PublishInboundWithID(InboundMessage{Content: "two", SessionKey: "s"}); !ok {
		t.Fatal("publish two failed")
	}
	firstID, ok := mb.InsertInboundFirst(InboundMessage{Content: "first", SessionKey: "s"})
	if !ok {
		t.Fatal("insert first failed")
	}

	items := mb.ListInbound()
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if items[0].ID != firstID || items[0].Message.Content != "first" {
		t.Fatalf("head item mismatch: got id=%s content=%q", items[0].ID, items[0].Message.Content)
	}
	if got := items[0].Message.Metadata["queue_insert_mode"]; got != "head" {
		t.Fatalf("expected queue_insert_mode=head, got %q", got)
	}
}

func TestMergeInboundWithPrefix(t *testing.T) {
	mb := NewMessageBus()
	mb.ConfigureIngress(3*time.Second, 10*time.Minute, "+")

	id1, ok := mb.PublishInboundWithID(InboundMessage{
		Channel:    "telegram",
		SenderID:   "u1",
		ChatID:     "c1",
		SessionKey: "telegram:c1",
		Content:    "hello",
	})
	if !ok {
		t.Fatal("publish first failed")
	}
	id2, ok := mb.PublishInboundWithID(InboundMessage{
		Channel:    "telegram",
		SenderID:   "u1",
		ChatID:     "c1",
		SessionKey: "telegram:c1",
		Content:    "+ and one more",
	})
	if !ok {
		t.Fatal("publish second failed")
	}
	if id1 != id2 {
		t.Fatalf("expected merged id to stay %s, got %s", id1, id2)
	}

	items := mb.ListInbound()
	if len(items) != 1 {
		t.Fatalf("expected 1 merged item, got %d", len(items))
	}
	if items[0].Message.Content != "hello\nand one more" {
		t.Fatalf("unexpected merged content: %q", items[0].Message.Content)
	}
	if items[0].Message.Metadata["merged"] != "true" {
		t.Fatalf("expected merged metadata")
	}
}
