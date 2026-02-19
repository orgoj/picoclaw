package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildSystemPrompt_IncludesNamedAgentsSummary(t *testing.T) {
	workspace := t.TempDir()
	agentDir := filepath.Join(workspace, "agents", "dev-agent")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	agentsFile := `---
name: dev-agent
description: Writes tests
---

# Dev Agent
`
	if err := os.WriteFile(filepath.Join(agentDir, "AGENTS.md"), []byte(agentsFile), 0644); err != nil {
		t.Fatalf("write AGENTS.md failed: %v", err)
	}

	cb := NewContextBuilder(workspace)
	prompt := cb.BuildSystemPrompt()

	if !strings.Contains(prompt, "# Named Agents") {
		t.Fatalf("expected named agents section in prompt, got: %s", prompt)
	}
	if !strings.Contains(prompt, "**dev-agent**") {
		t.Fatalf("expected named agent name in prompt, got: %s", prompt)
	}
	if !strings.Contains(prompt, "Writes tests") {
		t.Fatalf("expected named agent description in prompt, got: %s", prompt)
	}
}
