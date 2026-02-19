package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

type writeScopeProvider struct {
	toolPath string
	calls    int
}

func (p *writeScopeProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string, options map[string]interface{}) (*providers.LLMResponse, error) {
	p.calls++
	if p.calls == 1 {
		return &providers.LLMResponse{
			ToolCalls: []providers.ToolCall{{
				ID:   "tool-1",
				Name: "write_file",
				Arguments: map[string]interface{}{
					"path":    p.toolPath,
					"content": "from subagent",
				},
			}},
		}, nil
	}

	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "tool" {
			return &providers.LLMResponse{Content: messages[i].Content}, nil
		}
	}
	return &providers.LLMResponse{Content: "done"}, nil
}

func (p *writeScopeProvider) GetDefaultModel() string {
	return "test-model"
}

func writeScopeTestConfig(workspace string) *config.Config {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = workspace
	cfg.Agents.Defaults.Model = "test-model"
	cfg.Agents.Defaults.MaxIterationsSubagent = 3
	cfg.Agents.Defaults.MaxTokensSubagent = 1024
	cfg.Agents.Defaults.LLMTimeout = 30
	cfg.Agents.Defaults.RestrictToWorkspace = true
	return cfg
}

func TestSubagentWriteScope_AllowsNamedAgentMemory(t *testing.T) {
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, "agents", "code-reviewer", "memory"), 0755); err != nil {
		t.Fatal(err)
	}

	provider := &writeScopeProvider{toolPath: "agents/code-reviewer/memory/notes.md"}
	manager := NewSubagentManager(provider, writeScopeTestConfig(workspace), workspace, nil)
	registry := NewToolRegistry()
	registry.Register(NewWriteFileTool(workspace, true))
	manager.SetTools(registry)

	tool := NewSubagentTool(manager)
	result := tool.Execute(context.Background(), map[string]interface{}{
		"task": "write memory",
		"name": "code-reviewer",
	})
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}

	memoryFile := filepath.Join(workspace, "agents", "code-reviewer", "memory", "notes.md")
	if _, err := os.Stat(memoryFile); err != nil {
		t.Fatalf("expected file in memory scope, stat failed: %v", err)
	}
}

func TestSubagentWriteScope_BlocksOutsideNamedAgentMemory(t *testing.T) {
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, "agents", "code-reviewer", "memory"), 0755); err != nil {
		t.Fatal(err)
	}

	provider := &writeScopeProvider{toolPath: "memory/outside.md"}
	manager := NewSubagentManager(provider, writeScopeTestConfig(workspace), workspace, nil)
	registry := NewToolRegistry()
	registry.Register(NewWriteFileTool(workspace, true))
	manager.SetTools(registry)

	tool := NewSubagentTool(manager)
	result := tool.Execute(context.Background(), map[string]interface{}{
		"task": "write outside",
		"name": "code-reviewer",
	})
	if result.IsError {
		t.Fatalf("expected non-fatal completion, got error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "write scope denied") {
		t.Fatalf("expected write scope denial in result, got: %s", result.ForLLM)
	}

	outsideFile := filepath.Join(workspace, "memory", "outside.md")
	if _, err := os.Stat(outsideFile); !os.IsNotExist(err) {
		t.Fatalf("expected outside file to be blocked, got stat err: %v", err)
	}
}

func TestSubagentWriteScope_AllowsSelfUpdateProjectWrite(t *testing.T) {
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, "agents", "picoclaw-self-update", "memory"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(workspace, "projects", "picoclaw"), 0755); err != nil {
		t.Fatal(err)
	}

	provider := &writeScopeProvider{toolPath: "projects/picoclaw/CHANGELOG.md"}
	manager := NewSubagentManager(provider, writeScopeTestConfig(workspace), workspace, nil)
	registry := NewToolRegistry()
	registry.Register(NewWriteFileTool(workspace, true))
	manager.SetTools(registry)

	tool := NewSubagentTool(manager)
	result := tool.Execute(context.Background(), map[string]interface{}{
		"task": "self update write",
		"name": "picoclaw-self-update",
	})
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}

	projectFile := filepath.Join(workspace, "projects", "picoclaw", "CHANGELOG.md")
	if _, err := os.Stat(projectFile); err != nil {
		t.Fatalf("expected file in self-update scope, stat failed: %v", err)
	}
}
