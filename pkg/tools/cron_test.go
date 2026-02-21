package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/cron"
)

type noopJobExecutor struct{}

func (n noopJobExecutor) ProcessDirectWithChannel(ctx context.Context, content, sessionKey, channel, chatID string) (string, error) {
	return "", nil
}

func TestCronTool_ExecuteJob_CommandRespectsDenyPathPatterns(t *testing.T) {
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, ".beads"), 0755); err != nil {
		t.Fatalf("Failed to create workspace beads dir: %v", err)
	}

	msgBus := bus.NewMessageBus()
	cronService := cron.NewCronService(filepath.Join(workspace, "cron", "jobs.json"), nil)
	tool := NewCronTool(
		cronService,
		noopJobExecutor{},
		msgBus,
		workspace,
		false,
		"**/.beads/*.jsonl",
		"**/.beads/*.db",
	)

	job := &cron.CronJob{
		ID:   "job-1",
		Name: "blocked command test",
		Payload: cron.CronPayload{
			Message: "run blocked command",
			Command: "cat .beads/issues.jsonl",
			Channel: "telegram",
			To:      "1234",
		},
	}

	status := tool.ExecuteJob(context.Background(), job)
	if status != "ok" {
		t.Fatalf("Expected ExecuteJob to return ok, got %q", status)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, ok := msgBus.SubscribeOutbound(ctx)
	if !ok {
		t.Fatal("Expected outbound message from scheduled command execution")
	}

	if !strings.Contains(out.Content, "Error executing scheduled command:") {
		t.Fatalf("Expected scheduled command error message, got: %s", out.Content)
	}
	if !strings.Contains(out.Content, "path matches denied pattern") {
		t.Fatalf("Expected deny_path_patterns block reason, got: %s", out.Content)
	}
}
