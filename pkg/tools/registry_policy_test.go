package tools

import (
	"context"
	"testing"
)

type registryPolicyMockTool struct {
	name string
}

func (m *registryPolicyMockTool) Name() string { return m.name }

func (m *registryPolicyMockTool) Description() string { return "mock tool" }

func (m *registryPolicyMockTool) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object"}
}

func (m *registryPolicyMockTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	return SilentResult("ok")
}

func TestToolRegistryPolicy_BlocksUnlistedTools(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(&registryPolicyMockTool{name: "read_file"})
	reg.Register(&registryPolicyMockTool{name: "exec"})
	reg.SetPolicy(ToolPolicy{
		DenyByDefault: true,
		AllowList:     []string{"read_file"},
		NotifyOnBlock: true,
	})

	notified := 0
	reg.SetBlockedToolNotify(func(channel, chatID, toolName, reason string) {
		notified++
		if channel != "telegram" || chatID != "123" {
			t.Fatalf("unexpected notify routing: %s/%s", channel, chatID)
		}
		if toolName != "exec" {
			t.Fatalf("unexpected tool name: %s", toolName)
		}
	})

	allowed := reg.ExecuteWithContext(context.Background(), "read_file", map[string]interface{}{}, "telegram", "123", nil)
	if allowed == nil || allowed.IsError {
		t.Fatalf("expected read_file to be allowed, got %+v", allowed)
	}

	blocked := reg.ExecuteWithContext(context.Background(), "exec", map[string]interface{}{}, "telegram", "123", nil)
	if blocked == nil || !blocked.IsError {
		t.Fatalf("expected exec to be blocked, got %+v", blocked)
	}
	if notified != 1 {
		t.Fatalf("expected one blocked notification, got %d", notified)
	}

	if defs := reg.ToProviderDefs(); len(defs) != 1 || defs[0].Function.Name != "read_file" {
		t.Fatalf("expected only read_file in provider defs, got %+v", defs)
	}
}

func TestToolRegistryPolicy_NotifyCanBeDisabled(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(&registryPolicyMockTool{name: "write_file"})
	reg.SetPolicy(ToolPolicy{
		DenyByDefault: true,
		AllowList:     []string{},
		NotifyOnBlock: false,
	})

	notified := 0
	reg.SetBlockedToolNotify(func(channel, chatID, toolName, reason string) {
		notified++
	})

	result := reg.ExecuteWithContext(context.Background(), "write_file", map[string]interface{}{}, "telegram", "123", nil)
	if result == nil || !result.IsError {
		t.Fatalf("expected write_file to be blocked, got %+v", result)
	}
	if notified != 0 {
		t.Fatalf("expected blocked notify disabled, got %d calls", notified)
	}
}

func TestToolRegistryPolicy_DenyListBlocksSpecificToolWithoutDenyByDefault(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(&registryPolicyMockTool{name: "read_file"})
	reg.Register(&registryPolicyMockTool{name: "subagent_cancel"})
	reg.SetPolicy(ToolPolicy{
		DenyByDefault: false,
		DenyList:      []string{"subagent_cancel"},
		AllowList:     []string{},
		NotifyOnBlock: true,
	})

	allowed := reg.ExecuteWithContext(context.Background(), "read_file", map[string]interface{}{}, "telegram", "123", nil)
	if allowed == nil || allowed.IsError {
		t.Fatalf("expected read_file to be allowed, got %+v", allowed)
	}

	blocked := reg.ExecuteWithContext(context.Background(), "subagent_cancel", map[string]interface{}{}, "telegram", "123", nil)
	if blocked == nil || !blocked.IsError {
		t.Fatalf("expected subagent_cancel to be blocked by deny_list, got %+v", blocked)
	}
}
