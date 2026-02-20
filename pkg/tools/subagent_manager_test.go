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
	if msg.Metadata["subagent_id"] != "subagent-1" {
		t.Fatalf("expected metadata subagent_id=subagent-1, got %q", msg.Metadata["subagent_id"])
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

func TestRunTask_NoDeadlockOnTerminalPublish(t *testing.T) {
	msgBus := bus.NewMessageBus()
	manager := NewSubagentManager(&MockLLMProvider{}, testConfig(), t.TempDir(), msgBus)

	task := &SubagentTask{
		ID:            "subagent-deadlock-check",
		Task:          "quick task",
		Status:        "running",
		OriginChannel: "telegram",
		OriginChatID:  "chat-deadlock",
	}

	manager.mu.Lock()
	manager.tasks[task.ID] = task
	manager.cancels[task.ID] = func() {}
	manager.mu.Unlock()

	done := make(chan struct{})
	go manager.runTask(context.Background(), task, func(ctx context.Context, result *ToolResult) {
		close(done)
	})

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("runTask callback did not complete (possible deadlock)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	msg, ok := msgBus.ConsumeInbound(ctx)
	if !ok {
		t.Fatal("expected terminal update message")
	}
	if msg.SenderID != "subagent:subagent-deadlock-check" {
		t.Fatalf("unexpected sender id: %s", msg.SenderID)
	}
	if msg.Metadata["subagent_id"] != "subagent-deadlock-check" {
		t.Fatalf("expected metadata subagent_id=subagent-deadlock-check, got %q", msg.Metadata["subagent_id"])
	}
}
