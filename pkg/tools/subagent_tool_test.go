package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// MockLLMProvider is a test implementation of LLMProvider
type MockLLMProvider struct{}

func (m *MockLLMProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string, options map[string]interface{}) (*providers.LLMResponse, error) {
	// Find the last user message to generate a response
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			return &providers.LLMResponse{
				Content: "Task completed: " + messages[i].Content,
			}, nil
		}
	}
	return &providers.LLMResponse{Content: "No task provided"}, nil
}

func (m *MockLLMProvider) GetDefaultModel() string {
	return "test-model"
}

func (m *MockLLMProvider) SupportsTools() bool {
	return false
}

func (m *MockLLMProvider) GetContextWindow() int {
	return 4096
}

type EmptyResponseProvider struct{}

func (m *EmptyResponseProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string, options map[string]interface{}) (*providers.LLMResponse, error) {
	return &providers.LLMResponse{
		Content: "",
	}, nil
}

func (m *EmptyResponseProvider) GetDefaultModel() string {
	return "test-model"
}

func (m *EmptyResponseProvider) SupportsTools() bool {
	return false
}

func (m *EmptyResponseProvider) GetContextWindow() int {
	return 4096
}

type EmptyFinalWithAssistantProvider struct {
	calls int
}

func (m *EmptyFinalWithAssistantProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string, options map[string]interface{}) (*providers.LLMResponse, error) {
	m.calls++
	if m.calls == 1 {
		return &providers.LLMResponse{
			Content: "Collected outputs from tools.",
			ToolCalls: []providers.ToolCall{
				{
					ID:   "call-1",
					Name: "read_file",
					Arguments: map[string]any{
						"path": "missing.txt",
					},
				},
			},
		}, nil
	}
	return &providers.LLMResponse{
		Content: "",
	}, nil
}

func (m *EmptyFinalWithAssistantProvider) GetDefaultModel() string {
	return "test-model"
}

func (m *EmptyFinalWithAssistantProvider) SupportsTools() bool {
	return true
}

func (m *EmptyFinalWithAssistantProvider) GetContextWindow() int {
	return 4096
}

// testConfig creates a minimal config for testing
func testConfig() *config.Config {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Model = "test-model"
	cfg.Agents.Subagents.Model = "test-model"
	cfg.Agents.Subagents.MaxIterations = 5
	cfg.Agents.Subagents.MaxTokens = 1024
	cfg.Agents.Defaults.LLMTimeout = 30
	return cfg
}

// TestSubagentTool_Name verifies tool name
func TestSubagentTool_Name(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, testConfig(), "/tmp/test", nil)
	tool := NewSubagentTool(manager)

	if tool.Name() != "subagent" {
		t.Errorf("Expected name 'subagent', got '%s'", tool.Name())
	}
}

// TestSubagentTool_Description verifies tool description
func TestSubagentTool_Description(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, testConfig(), "/tmp/test", nil)
	tool := NewSubagentTool(manager)

	desc := tool.Description()
	if desc == "" {
		t.Error("Description should not be empty")
	}
	if !strings.Contains(desc, "subagent") {
		t.Errorf("Description should mention 'subagent', got: %s", desc)
	}
}

// TestSubagentTool_Parameters verifies tool parameters schema
func TestSubagentTool_Parameters(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, testConfig(), "/tmp/test", nil)
	tool := NewSubagentTool(manager)

	params := tool.Parameters()
	if params == nil {
		t.Error("Parameters should not be nil")
	}

	// Check type
	if params["type"] != "object" {
		t.Errorf("Expected type 'object', got: %v", params["type"])
	}

	// Check properties
	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Properties should be a map")
	}

	// Verify task parameter
	task, ok := props["task"].(map[string]interface{})
	if !ok {
		t.Fatal("Task parameter should exist")
	}
	if task["type"] != "string" {
		t.Errorf("Task type should be 'string', got: %v", task["type"])
	}

	// Verify label parameter
	label, ok := props["label"].(map[string]interface{})
	if !ok {
		t.Fatal("Label parameter should exist")
	}
	if label["type"] != "string" {
		t.Errorf("Label type should be 'string', got: %v", label["type"])
	}

	// Verify name parameter
	nameParam, ok := props["name"].(map[string]interface{})
	if !ok {
		t.Fatal("Name parameter should exist")
	}
	if nameParam["type"] != "string" {
		t.Errorf("Name type should be 'string', got: %v", nameParam["type"])
	}

	// Check required fields
	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("Required should be a string array")
	}
	if len(required) != 1 || required[0] != "task" {
		t.Errorf("Required should be ['task'], got: %v", required)
	}
}

// TestSubagentTool_SetContext verifies context setting
func TestSubagentTool_SetContext(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, testConfig(), "/tmp/test", nil)
	tool := NewSubagentTool(manager)

	tool.SetContext("test-channel", "test-chat")

	// Verify context is set (we can't directly access private fields,
	// but we can verify it doesn't crash)
	// The actual context usage is tested in Execute tests
}

// TestSubagentTool_Execute_Success tests successful execution
func TestSubagentTool_Execute_Success(t *testing.T) {
	provider := &MockLLMProvider{}
	msgBus := bus.NewMessageBus()
	manager := NewSubagentManager(provider, testConfig(), "/tmp/test", msgBus)
	tool := NewSubagentTool(manager)
	tool.SetContext("telegram", "chat-123")

	ctx := context.Background()
	args := map[string]interface{}{
		"task":  "Write a haiku about coding",
		"label": "haiku-task",
	}

	result := tool.Execute(ctx, args)

	// Verify basic ToolResult structure
	if result == nil {
		t.Fatal("Result should not be nil")
	}

	// Verify no error
	if result.IsError {
		t.Errorf("Expected success, got error: %s", result.ForLLM)
	}

	// Verify not async
	if result.Async {
		t.Error("SubagentTool should be synchronous, not async")
	}

	// Verify not silent
	if result.Silent {
		t.Error("SubagentTool should not be silent")
	}

	// Verify ForUser contains brief summary (not empty)
	if result.ForUser == "" {
		t.Error("ForUser should contain result summary")
	}
	if !strings.Contains(result.ForUser, "Task completed") {
		t.Errorf("ForUser should contain task completion, got: %s", result.ForUser)
	}

	// Verify ForLLM contains full details
	if result.ForLLM == "" {
		t.Error("ForLLM should contain full details")
	}
	if !strings.Contains(result.ForLLM, "haiku-task") {
		t.Errorf("ForLLM should contain label 'haiku-task', got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "Task completed:") {
		t.Errorf("ForLLM should contain task result, got: %s", result.ForLLM)
	}
}

// TestSubagentTool_Execute_NoLabel tests execution without label
func TestSubagentTool_Execute_NoLabel(t *testing.T) {
	provider := &MockLLMProvider{}
	msgBus := bus.NewMessageBus()
	manager := NewSubagentManager(provider, testConfig(), "/tmp/test", msgBus)
	tool := NewSubagentTool(manager)

	ctx := context.Background()
	args := map[string]interface{}{
		"task": "Test task without label",
	}

	result := tool.Execute(ctx, args)

	if result.IsError {
		t.Errorf("Expected success without label, got error: %s", result.ForLLM)
	}

	// ForLLM should show no-label for missing label
	if !strings.Contains(result.ForLLM, "no-label") {
		t.Errorf("ForLLM should show 'no-label' for missing label, got: %s", result.ForLLM)
	}
}

func TestSubagentTool_Execute_EmptyFinalContentUsesFallback(t *testing.T) {
	provider := &EmptyResponseProvider{}
	msgBus := bus.NewMessageBus()
	manager := NewSubagentManager(provider, testConfig(), "/tmp/test", msgBus)
	tool := NewSubagentTool(manager)

	ctx := context.Background()
	args := map[string]interface{}{
		"task": "task with empty response",
	}

	result := tool.Execute(ctx, args)
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForUser, "without textual summary") {
		t.Fatalf("expected fallback user content, got: %q", result.ForUser)
	}
	if !strings.Contains(result.ForLLM, "without textual summary") {
		t.Fatalf("expected fallback llm content, got: %q", result.ForLLM)
	}
}

func TestSubagentTool_Execute_EmptyFinalUsesLastAssistantFallback(t *testing.T) {
	provider := &EmptyFinalWithAssistantProvider{}
	msgBus := bus.NewMessageBus()
	manager := NewSubagentManager(provider, testConfig(), "/tmp/test", msgBus)
	tool := NewSubagentTool(manager)

	ctx := context.Background()
	args := map[string]interface{}{
		"task": "task with empty final response but prior assistant text",
	}

	result := tool.Execute(ctx, args)
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForUser, "without textual summary") {
		t.Fatalf("expected fallback marker, got: %q", result.ForUser)
	}
	if !strings.Contains(result.ForUser, "Last assistant message:") {
		t.Fatalf("expected last assistant fallback marker, got: %q", result.ForUser)
	}
	if !strings.Contains(result.ForUser, "Collected outputs from tools.") {
		t.Fatalf("expected assistant fallback text, got: %q", result.ForUser)
	}
}

// TestSubagentTool_Execute_MissingTask tests error handling for missing task
func TestSubagentTool_Execute_MissingTask(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, testConfig(), "/tmp/test", nil)
	tool := NewSubagentTool(manager)

	ctx := context.Background()
	args := map[string]interface{}{
		"label": "test",
	}

	result := tool.Execute(ctx, args)

	// Should return error
	if !result.IsError {
		t.Error("Expected error for missing task parameter")
	}

	// ForLLM should contain error message
	if !strings.Contains(result.ForLLM, "task is required") {
		t.Errorf("Error message should mention 'task is required', got: %s", result.ForLLM)
	}

	// Err should be set
	if result.Err == nil {
		t.Error("Err should be set for validation failure")
	}
}

// TestSubagentTool_Execute_NilManager tests error handling for nil manager
func TestSubagentTool_Execute_NilManager(t *testing.T) {
	tool := NewSubagentTool(nil)

	ctx := context.Background()
	args := map[string]interface{}{
		"task": "test task",
	}

	result := tool.Execute(ctx, args)

	// Should return error
	if !result.IsError {
		t.Error("Expected error for nil manager")
	}

	if !strings.Contains(result.ForLLM, "Subagent manager not configured") {
		t.Errorf("Error message should mention manager not configured, got: %s", result.ForLLM)
	}
}

// TestSubagentTool_Execute_ContextPassing verifies context is properly used
func TestSubagentTool_Execute_ContextPassing(t *testing.T) {
	provider := &MockLLMProvider{}
	msgBus := bus.NewMessageBus()
	manager := NewSubagentManager(provider, testConfig(), "/tmp/test", msgBus)
	tool := NewSubagentTool(manager)

	// Set context
	channel := "test-channel"
	chatID := "test-chat"
	tool.SetContext(channel, chatID)

	ctx := context.Background()
	args := map[string]interface{}{
		"task": "Test context passing",
	}

	result := tool.Execute(ctx, args)

	// Should succeed
	if result.IsError {
		t.Errorf("Expected success with context, got error: %s", result.ForLLM)
	}

	// The context is used internally; we can't directly test it
	// but execution success indicates context was handled properly
}

// TestSubagentTool_Execute_WithName tests that named agent executes successfully
// even when the agent directory doesn't exist (silent fallback).
func TestSubagentTool_Execute_WithName(t *testing.T) {
	provider := &MockLLMProvider{}
	msgBus := bus.NewMessageBus()
	manager := NewSubagentManager(provider, testConfig(), "/tmp/test", msgBus)
	tool := NewSubagentTool(manager)

	ctx := context.Background()
	args := map[string]interface{}{
		"task":  "Test with name",
		"name":  "test-agent", // workspace/agents/test-agent/ doesn't exist → ignored silently
		"label": "named-test",
	}

	result := tool.Execute(ctx, args)
	if result.IsError {
		t.Errorf("Should succeed even without agent dir: %s", result.ForLLM)
	}
}

func TestSubagentTool_Execute_WithNameProviderOverride(t *testing.T) {
	provider := &MockLLMProvider{}
	msgBus := bus.NewMessageBus()
	cfg := testConfig()
	cfg.Agents.Defaults.Provider = "zhipu"
	cfg.Agents.NamedAgents = map[string]config.NamedAgentConfig{
		"silicon-worker": {
			Provider: "openai",
			Model:    "deepseek-ai/DeepSeek-V3.2",
		},
	}
	manager := NewSubagentManager(provider, cfg, "/tmp/test", msgBus)
	tool := NewSubagentTool(manager)

	ctx := context.Background()
	args := map[string]interface{}{
		"task": "Test per-agent provider override",
		"name": "silicon-worker",
	}

	result := tool.Execute(ctx, args)
	if !result.IsError {
		t.Fatalf("Expected provider resolution error due to missing OpenAI credentials")
	}
	if !strings.Contains(strings.ToLower(result.ForLLM), "provider") {
		t.Fatalf("Expected provider-related error, got: %s", result.ForLLM)
	}
}

// TestSubagentTool_Execute_InvalidName tests that path traversal names are ignored safely.
func TestSubagentTool_Execute_InvalidName(t *testing.T) {
	provider := &MockLLMProvider{}
	msgBus := bus.NewMessageBus()
	manager := NewSubagentManager(provider, testConfig(), "/tmp/test", msgBus)
	tool := NewSubagentTool(manager)

	ctx := context.Background()
	args := map[string]interface{}{
		"task": "Test",
		"name": "../../etc",
	}

	result := tool.Execute(ctx, args)
	// Should succeed (name ignored), not panic/error
	if result.IsError {
		t.Errorf("Invalid name should be ignored, not cause error: %s", result.ForLLM)
	}
}

// TestSubagentTool_ForUserTruncation verifies long content is truncated for user
func TestSubagentTool_ForUserTruncation(t *testing.T) {
	// Create a mock provider that returns very long content
	provider := &MockLLMProvider{}
	msgBus := bus.NewMessageBus()
	manager := NewSubagentManager(provider, testConfig(), "/tmp/test", msgBus)
	tool := NewSubagentTool(manager)

	ctx := context.Background()

	// Create a task that will generate long response
	longTask := strings.Repeat("This is a very long task description. ", 100)
	args := map[string]interface{}{
		"task":  longTask,
		"label": "long-test",
	}

	result := tool.Execute(ctx, args)

	// ForUser should be truncated to 500 chars + "..."
	maxUserLen := 500
	if len(result.ForUser) > maxUserLen+3 { // +3 for "..."
		t.Errorf("ForUser should be truncated to ~%d chars, got: %d", maxUserLen, len(result.ForUser))
	}

	// ForLLM should have full content
	if !strings.Contains(result.ForLLM, longTask[:50]) {
		t.Error("ForLLM should contain reference to original task")
	}
}
