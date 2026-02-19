package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// MockProvider that returns errors
type MockErrorProvider struct {
	err error
}

func (m *MockErrorProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string, options map[string]interface{}) (*providers.LLMResponse, error) {
	return nil, m.err
}

func (m *MockErrorProvider) GetDefaultModel() string {
	return "test-model"
}

func TestFormatErrorMessage(t *testing.T) {
	cfg := config.DefaultConfig()
	msgBus := bus.NewMessageBus()
	provider := &MockErrorProvider{err: errors.New("test error")}

	agentLoop := NewAgentLoop(cfg, msgBus, provider)

	tests := []struct {
		name     string
		err      error
		contains string
	}{
		{
			name:     "timeout error",
			err:      errors.New("context deadline exceeded: timeout"),
			contains: "API timeout",
		},
		{
			name:     "unexpected EOF",
			err:      errors.New("Post \"https://api.z.ai/api/coding/paas/v4/chat/completions\": unexpected EOF"),
			contains: "connection interrupted",
		},
		{
			name:     "connection reset",
			err:      errors.New("connection reset by peer"),
			contains: "connection interrupted",
		},
		{
			name:     "rate limit 429",
			err:      errors.New("API request failed: status=429 - rate limited"),
			contains: "rate limited",
		},
		{
			name:     "server error 500",
			err:      errors.New("API request failed: status=500"),
			contains: "status=500",
		},
		{
			name:     "server error 503",
			err:      errors.New("API request failed: status=503"),
			contains: "status=503",
		},
		{
			name:     "connection refused",
			err:      errors.New("connection refused"),
			contains: "connection refused",
		},
		{
			name:     "network unreachable",
			err:      errors.New("network is unreachable"),
			contains: "network unreachable",
		},
		{
			name:     "LLM call failed",
			err:      errors.New("LLM call failed: provider error"),
			contains: "Failed to communicate with AI service",
		},
		{
			name:     "generic error",
			err:      errors.New("some random error"),
			contains: "An error occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := agentLoop.formatErrorMessage(tt.err, "test-session")
			if !containsString(result, tt.contains) {
				t.Errorf("formatErrorMessage() = %q, want to contain %q", result, tt.contains)
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
