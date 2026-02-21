package channels

import (
	"strings"
	"testing"
)

func TestMarkdownToTelegramHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple strikethrough",
			input:    "~~deleted~~",
			expected: "<s>deleted</s>",
		},
		{
			name:     "strikethrough in text",
			input:    "This is ~~old~~ and new",
			expected: "This is <s>old</s> and new",
		},
		{
			name:     "multiline strikethrough",
			input:    "~~line1\nline2~~",
			expected: "<s>line1\nline2</s>",
		},
		{
			name:     "bold text",
			input:    "**bold**",
			expected: "<b>bold</b>",
		},
		{
			name:     "italic text",
			input:    "_italic_",
			expected: "<i>italic</i>",
		},
		{
			name:     "link",
			input:    "[text](https://example.com)",
			expected: `<a href="https://example.com">text</a>`,
		},
		{
			name:     "escape HTML",
			input:    "<script>alert('xss')</script>",
			expected: "&lt;script&gt;alert('xss')&lt;/script&gt;",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "mixed formatting",
			input:    "**bold** and ~~strikethrough~~ and _italic_",
			expected: "<b>bold</b> and <s>strikethrough</s> and <i>italic</i>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := markdownToTelegramHTML(tt.input)
			if result != tt.expected {
				t.Errorf("markdownToTelegramHTML(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSplitTelegramMessage_SplitsLongContent(t *testing.T) {
	input := strings.Repeat("a", telegramMessageLimit+120)
	chunks := splitTelegramMessage(input)
	if len(chunks) < 2 {
		t.Fatalf("expected split into multiple chunks, got %d", len(chunks))
	}
	for i, chunk := range chunks {
		if len(chunk) > telegramMessageLimit {
			t.Fatalf("chunk %d too long: %d", i, len(chunk))
		}
	}
}

func TestSplitTelegramMessage_EmptyReturnsNil(t *testing.T) {
	chunks := splitTelegramMessage("   ")
	if len(chunks) != 0 {
		t.Fatalf("expected no chunks for empty content, got %d", len(chunks))
	}
}
