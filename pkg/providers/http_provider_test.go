package providers

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestShouldUseMaxCompletionTokens(t *testing.T) {
	cases := []struct {
		model string
		want  bool
	}{
		{model: "glm-4.6", want: true},
		{model: "zai/glm-4.6", want: true},
		{model: "gpt-5", want: true},
		{model: "openai/gpt-5.2", want: true},
		{model: "o1", want: true},
		{model: "openai/o3-mini", want: true},
		{model: "openai/o4-mini", want: true},
		{model: "gpt-4o", want: false},
		{model: "openai/gpt-4.1", want: false},
		{model: "claude-sonnet-4-5", want: false},
		{model: "", want: false},
	}

	for _, tc := range cases {
		got := shouldUseMaxCompletionTokens(tc.model)
		if got != tc.want {
			t.Fatalf("shouldUseMaxCompletionTokens(%q)=%v want=%v", tc.model, got, tc.want)
		}
	}
}

func TestShouldUseStreamingResponse(t *testing.T) {
	tests := []struct {
		name    string
		apiBase string
		model   string
		tools   []ToolDefinition
		options map[string]any
		want    bool
	}{
		{
			name:    "auto z.ai glm without tools",
			apiBase: "https://api.z.ai/api/coding/paas/v4",
			model:   "glm-4.7",
			want:    true,
		},
		{
			name:    "disabled when tools present",
			apiBase: "https://api.z.ai/api/coding/paas/v4",
			model:   "glm-4.7",
			tools:   []ToolDefinition{{Type: "function"}},
			want:    false,
		},
		{
			name:    "disabled for non-zai endpoint",
			apiBase: "https://api.openai.com/v1",
			model:   "glm-4.7",
			want:    false,
		},
		{
			name:    "explicit option overrides auto",
			apiBase: "https://api.openai.com/v1",
			model:   "gpt-4o",
			options: map[string]any{"stream": true},
			want:    true,
		},
	}

	for _, tt := range tests {
		got := shouldUseStreamingResponse(tt.apiBase, tt.model, tt.tools, tt.options)
		if got != tt.want {
			t.Fatalf("%s: got %v want %v", tt.name, got, tt.want)
		}
	}
}

func TestParseStreamingResponseSSE(t *testing.T) {
	body := strings.Join([]string{
		`data: {"choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}`,
		``,
		`data: {"choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}`,
		``,
		`data: {"choices":[{"index":0,"finish_reason":"stop","delta":{"content":""}}],"usage":{"prompt_tokens":8,"completion_tokens":2,"total_tokens":10}}`,
		``,
		`data: [DONE]`,
		``,
	}, "\n")

	resp := &http.Response{
		Header: http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:   io.NopCloser(strings.NewReader(body)),
	}

	p := &HTTPProvider{}
	got, err := p.parseStreamingResponse(resp)
	if err != nil {
		t.Fatalf("parseStreamingResponse error: %v", err)
	}
	if got.Content != "Hello world" {
		t.Fatalf("content=%q want %q", got.Content, "Hello world")
	}
	if got.FinishReason != "stop" {
		t.Fatalf("finish_reason=%q want stop", got.FinishReason)
	}
	if got.Usage == nil || got.Usage.TotalTokens != 10 {
		t.Fatalf("usage not parsed: %+v", got.Usage)
	}
}

func TestParseStreamingResponseFallsBackToJSON(t *testing.T) {
	body := `{"choices":[{"message":{"content":"plain"},"finish_reason":"stop"}]}`
	resp := &http.Response{
		Header: http.Header{"Content-Type": []string{"application/json"}},
		Body:   io.NopCloser(strings.NewReader(body)),
	}
	p := &HTTPProvider{}

	got, err := p.parseStreamingResponse(resp)
	if err != nil {
		t.Fatalf("parseStreamingResponse fallback error: %v", err)
	}
	if got.Content != "plain" {
		t.Fatalf("content=%q want plain", got.Content)
	}
}
