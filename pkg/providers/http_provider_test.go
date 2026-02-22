package providers

import "testing"

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
