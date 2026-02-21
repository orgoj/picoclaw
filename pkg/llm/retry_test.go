package llm

import (
	"context"
	"errors"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

type flakyProvider struct {
	failures int
	calls    int
	errMsg   string
}

func (f *flakyProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string, options map[string]interface{}) (*providers.LLMResponse, error) {
	f.calls++
	if f.calls <= f.failures {
		if f.errMsg != "" {
			return nil, errors.New(f.errMsg)
		}
		return nil, errors.New("failed to send request: unexpected EOF")
	}
	return &providers.LLMResponse{Content: "ok"}, nil
}

func (f *flakyProvider) GetDefaultModel() string {
	return "test"
}

func TestCallWithRetry_RetriesTransientError(t *testing.T) {
	p := &flakyProvider{failures: 1}
	resp, err := CallWithRetry(context.Background(), CallConfig{
		Provider: p,
		Messages: []providers.Message{{Role: "user", Content: "hello"}},
		Model:    "test",
		Retry: RetryConfig{
			MaxRetries:              2,
			RetryBackoffSeconds:     1,
			RetryMaxBackoffSeconds:  1,
			RateLimitBackoffSeconds: 1,
			RateLimitMaxBackoffSecs: 1,
		},
		LogComponent: "test",
	})
	if err != nil {
		t.Fatalf("expected success after retry, got error: %v", err)
	}
	if resp == nil || resp.Content != "ok" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if p.calls != 2 {
		t.Fatalf("expected 2 calls, got %d", p.calls)
	}
}

func TestCallWithRetry_RetriesContextDeadlineExceeded(t *testing.T) {
	p := &flakyProvider{
		failures: 1,
		errMsg:   "failed to send request: Post \"https://api.z.ai/api/coding/paas/v4/chat/completions\": context deadline exceeded (Client.Timeout exceeded while awaiting headers)",
	}
	resp, err := CallWithRetry(context.Background(), CallConfig{
		Provider: p,
		Messages: []providers.Message{{Role: "user", Content: "hello"}},
		Model:    "test",
		Retry: RetryConfig{
			MaxRetries:              2,
			RetryBackoffSeconds:     1,
			RetryMaxBackoffSeconds:  1,
			RateLimitBackoffSeconds: 1,
			RateLimitMaxBackoffSecs: 1,
		},
		LogComponent: "test",
	})
	if err != nil {
		t.Fatalf("expected success after timeout retry, got error: %v", err)
	}
	if resp == nil || resp.Content != "ok" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if p.calls != 2 {
		t.Fatalf("expected 2 calls, got %d", p.calls)
	}
}
