package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAvailableAgents_Empty(t *testing.T) {
	agents := LoadAvailableAgents("/tmp/nonexistent-workspace-xyz")
	if len(agents) != 0 {
		t.Errorf("Expected 0 agents for missing workspace, got %d", len(agents))
	}
}

func TestLoadAvailableAgents_WithAgents(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, "agents")

	// Create valid agent
	validDir := filepath.Join(agentsDir, "coder")
	os.MkdirAll(validDir, 0755)
	os.WriteFile(filepath.Join(validDir, "AGENTS.md"), []byte(`---
name: coder
description: Senior Go developer
---

## Identity
I am a coder.
`), 0644)

	// Create agent without frontmatter (should be skipped)
	noFMDir := filepath.Join(agentsDir, "plain")
	os.MkdirAll(noFMDir, 0755)
	os.WriteFile(filepath.Join(noFMDir, "AGENTS.md"), []byte("No frontmatter here.\n"), 0644)

	// Create agent with invalid name (path traversal) — won't exist as dir with dots, but test name validation
	badDir := filepath.Join(agentsDir, "reviewer")
	os.MkdirAll(badDir, 0755)
	os.WriteFile(filepath.Join(badDir, "AGENTS.md"), []byte(`---
name: ../../etc
description: Bad agent
---
`), 0644)

	agents := LoadAvailableAgents(dir)

	if len(agents) != 1 {
		t.Errorf("Expected 1 valid agent, got %d: %v", len(agents), agents)
	}
	if agents[0].Name != "coder" {
		t.Errorf("Expected name 'coder', got '%s'", agents[0].Name)
	}
	if agents[0].Description != "Senior Go developer" {
		t.Errorf("Expected description 'Senior Go developer', got '%s'", agents[0].Description)
	}
}

func TestLoadAvailableAgents_MissingDescription(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, "agents", "nodesc")
	os.MkdirAll(agentsDir, 0755)
	os.WriteFile(filepath.Join(agentsDir, "AGENTS.md"), []byte(`---
name: nodesc
---
`), 0644)

	agents := LoadAvailableAgents(dir)
	if len(agents) != 0 {
		t.Errorf("Agent without description should be skipped, got %d", len(agents))
	}
}

func TestParseAgentFrontmatter_QuotedValues(t *testing.T) {
	content := `---
name: my-agent
description: "Does something useful"
---
`
	meta := parseAgentFrontmatter(content)
	if meta == nil {
		t.Fatal("Expected non-nil metadata")
	}
	if meta.Name != "my-agent" {
		t.Errorf("Expected 'my-agent', got '%s'", meta.Name)
	}
	if meta.Description != "Does something useful" {
		t.Errorf("Expected 'Does something useful', got '%s'", meta.Description)
	}
}
