package preflight

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
}

func makeBootstrapWorkspace(t *testing.T) string {
	t.Helper()
	ws := t.TempDir()
	writeFile(t, filepath.Join(ws, "AGENT.md"), "# agent")
	writeFile(t, filepath.Join(ws, "IDENTITY.md"), "# identity")
	writeFile(t, filepath.Join(ws, "SOUL.md"), "# soul")
	writeFile(t, filepath.Join(ws, "USER.md"), "# user")
	writeFile(t, filepath.Join(ws, "memory", "MEMORY.md"), "# memory")
	return ws
}

func TestRun_PassesForCleanWorkspace(t *testing.T) {
	ws := makeBootstrapWorkspace(t)
	skill := `---
name: ok-skill
description: desc
---

body`
	writeFile(t, filepath.Join(ws, "skills", "ok", "SKILL.md"), skill)

	if err := Run(ws); err != nil {
		t.Fatalf("expected no preflight error, got: %v", err)
	}
}

func TestRun_FailsForBadProjectAgentsDir(t *testing.T) {
	ws := makeBootstrapWorkspace(t)
	if err := os.MkdirAll(filepath.Join(ws, "projects", "demo", "agents"), 0755); err != nil {
		t.Fatal(err)
	}

	err := Run(ws)
	if err == nil {
		t.Fatal("expected preflight failure")
	}
	if !strings.Contains(err.Error(), "project-agents-dir") {
		t.Fatalf("expected project-agents-dir issue, got: %v", err)
	}
}

func TestRun_FailsForSkillLinterViolations(t *testing.T) {
	ws := makeBootstrapWorkspace(t)
	skill := `---
name: bad-skill
description: desc
---

Use grep -r and find . -name.`
	writeFile(t, filepath.Join(ws, "skills", "bad", "SKILL.md"), skill)

	err := Run(ws)
	if err == nil {
		t.Fatal("expected preflight failure")
	}
	msg := err.Error()
	if !strings.Contains(msg, "grep -r") {
		t.Fatalf("expected grep -r violation, got: %v", err)
	}
	if !strings.Contains(msg, "find . -name") {
		t.Fatalf("expected find . -name violation, got: %v", err)
	}
}
