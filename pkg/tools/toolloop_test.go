package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

type maxIterProvider struct{}

func (m *maxIterProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string, options map[string]interface{}) (*providers.LLMResponse, error) {
	return &providers.LLMResponse{
		Content: "",
		ToolCalls: []providers.ToolCall{
			{
				ID:        "tc-1",
				Name:      "noop",
				Arguments: map[string]interface{}{},
			},
		},
	}, nil
}

func (m *maxIterProvider) GetDefaultModel() string {
	return "test-model"
}

type directAnswerProvider struct{}

func (d *directAnswerProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string, options map[string]interface{}) (*providers.LLMResponse, error) {
	return &providers.LLMResponse{
		Content: "done",
	}, nil
}

func (d *directAnswerProvider) GetDefaultModel() string {
	return "test-model"
}

func TestRunToolLoop_ReturnsErrorWhenMaxIterationsReachedWithoutFinalResponse(t *testing.T) {
	cfg := ToolLoopConfig{
		Provider:      &maxIterProvider{},
		Model:         "test-model",
		MaxIterations: 3,
		RunID:         "test-max-iter",
		LLMOptions:    map[string]any{"max_tokens": 128},
	}

	_, err := RunToolLoop(context.Background(), cfg, []providers.Message{
		{Role: "system", Content: "system"},
		{Role: "user", Content: "user"},
	}, "cli", "direct")
	if err == nil {
		t.Fatal("expected error when max iterations are reached without final response")
	}
	if !strings.Contains(err.Error(), "max iterations") {
		t.Fatalf("expected max iterations error, got: %v", err)
	}
}

func TestRunToolLoop_SucceedsWithDirectAnswer(t *testing.T) {
	cfg := ToolLoopConfig{
		Provider:      &directAnswerProvider{},
		Model:         "test-model",
		MaxIterations: 3,
		RunID:         "test-direct-answer",
		LLMOptions:    map[string]any{"max_tokens": 128},
	}

	res, err := RunToolLoop(context.Background(), cfg, []providers.Message{
		{Role: "system", Content: "system"},
		{Role: "user", Content: "user"},
	}, "cli", "direct")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil || res.Content != "done" {
		t.Fatalf("unexpected result: %+v", res)
	}
}
