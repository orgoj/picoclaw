package tools

import (
	"context"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
)

func TestCancel_PublishesTerminalUpdateImmediately(t *testing.T) {
	msgBus := bus.NewMessageBus()
	manager := NewSubagentManager(&MockLLMProvider{}, testConfig(), "/tmp/test", msgBus)

	manager.mu.Lock()
	manager.tasks["subagent-1"] = &SubagentTask{
		ID:            "subagent-1",
		Status:        "running",
		OriginChannel: "telegram",
		OriginChatID:  "chat-1",
	}
	manager.cancels["subagent-1"] = func() {}
	manager.mu.Unlock()

	if err := manager.Cancel("subagent-1"); err != nil {
		t.Fatalf("cancel failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	msg, ok := msgBus.ConsumeInbound(ctx)
	if !ok {
		t.Fatal("expected terminal update message after cancel")
	}
	if msg.SenderID != "subagent:subagent-1" {
		t.Fatalf("unexpected sender id: %s", msg.SenderID)
	}
}

func TestPublishTaskUpdate_OnlyOncePerTask(t *testing.T) {
	msgBus := bus.NewMessageBus()
	manager := NewSubagentManager(&MockLLMProvider{}, testConfig(), "/tmp/test", msgBus)
	task := &SubagentTask{
		ID:            "subagent-2",
		Status:        "completed",
		Result:        "done",
		OriginChannel: "telegram",
		OriginChatID:  "chat-2",
	}

	manager.publishTaskUpdate(task)
	manager.publishTaskUpdate(task)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if _, ok := msgBus.ConsumeInbound(ctx); !ok {
		t.Fatal("expected first terminal update")
	}
	if _, got, timedOut := msgBus.ConsumeInboundWithTimeout(ctx, 50*time.Millisecond); got || !timedOut {
		t.Fatal("expected no duplicate terminal update")
	}
}
