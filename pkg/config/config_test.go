package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDefaultConfig_MaxConcurrentSubagents verifies max_concurrent_subagents default is 2
func TestDefaultConfig_MaxConcurrentSubagents(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Agents.Defaults.MaxConcurrentSubagents != 2 {
		t.Errorf("Expected MaxConcurrentSubagents default 2, got %d", cfg.Agents.Defaults.MaxConcurrentSubagents)
	}
}

// TestDefaultConfig_MaxConcurrentSubagents_Parsing verifies config parsing
func TestDefaultConfig_MaxConcurrentSubagents_Parsing(t *testing.T) {
	jsonData := `{
		"agents": {
			"defaults": {
				"max_concurrent_subagents": 5
			}
		}
	}`

	var cfg Config
	if err := json.Unmarshal([]byte(jsonData), &cfg); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if cfg.Agents.Defaults.MaxConcurrentSubagents != 5 {
		t.Errorf("Expected MaxConcurrentSubagents 5, got %d", cfg.Agents.Defaults.MaxConcurrentSubagents)
	}
}

func TestDefaultConfig_LLMRetryDefaults(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Agents.Defaults.LLMMaxRetries != 2 {
		t.Errorf("Expected LLMMaxRetries default 2, got %d", cfg.Agents.Defaults.LLMMaxRetries)
	}
	if cfg.Agents.Defaults.LLMRetryBackoffSeconds != 2 {
		t.Errorf("Expected LLMRetryBackoffSeconds default 2, got %d", cfg.Agents.Defaults.LLMRetryBackoffSeconds)
	}
	if cfg.Agents.Defaults.LLMRetryMaxBackoff != 8 {
		t.Errorf("Expected LLMRetryMaxBackoff default 8, got %d", cfg.Agents.Defaults.LLMRetryMaxBackoff)
	}
	if cfg.Agents.Defaults.LLMRateLimitBackoff != 10 {
		t.Errorf("Expected LLMRateLimitBackoff default 10, got %d", cfg.Agents.Defaults.LLMRateLimitBackoff)
	}
	if cfg.Agents.Defaults.LLMRateLimitMaxBackoff != 30 {
		t.Errorf("Expected LLMRateLimitMaxBackoff default 30, got %d", cfg.Agents.Defaults.LLMRateLimitMaxBackoff)
	}
	if cfg.Agents.Defaults.LLMRetryMaxElapsed != 60 {
		t.Errorf("Expected LLMRetryMaxElapsed default 60, got %d", cfg.Agents.Defaults.LLMRetryMaxElapsed)
	}
}

func TestDefaultConfig_LLMRetryParsing(t *testing.T) {
	jsonData := `{
		"agents": {
			"defaults": {
				"llm_max_retries": 4,
				"llm_retry_backoff_seconds": 3,
				"llm_retry_max_backoff_seconds": 12,
				"llm_rate_limit_backoff_seconds": 15,
				"llm_rate_limit_max_backoff_seconds": 45,
				"llm_retry_max_elapsed_seconds": 180
			}
		}
	}`

	var cfg Config
	if err := json.Unmarshal([]byte(jsonData), &cfg); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if cfg.Agents.Defaults.LLMMaxRetries != 4 {
		t.Errorf("Expected LLMMaxRetries 4, got %d", cfg.Agents.Defaults.LLMMaxRetries)
	}
	if cfg.Agents.Defaults.LLMRetryBackoffSeconds != 3 {
		t.Errorf("Expected LLMRetryBackoffSeconds 3, got %d", cfg.Agents.Defaults.LLMRetryBackoffSeconds)
	}
	if cfg.Agents.Defaults.LLMRetryMaxBackoff != 12 {
		t.Errorf("Expected LLMRetryMaxBackoff 12, got %d", cfg.Agents.Defaults.LLMRetryMaxBackoff)
	}
	if cfg.Agents.Defaults.LLMRateLimitBackoff != 15 {
		t.Errorf("Expected LLMRateLimitBackoff 15, got %d", cfg.Agents.Defaults.LLMRateLimitBackoff)
	}
	if cfg.Agents.Defaults.LLMRateLimitMaxBackoff != 45 {
		t.Errorf("Expected LLMRateLimitMaxBackoff 45, got %d", cfg.Agents.Defaults.LLMRateLimitMaxBackoff)
	}
	if cfg.Agents.Defaults.LLMRetryMaxElapsed != 180 {
		t.Errorf("Expected LLMRetryMaxElapsed 180, got %d", cfg.Agents.Defaults.LLMRetryMaxElapsed)
	}
}

// TestDefaultConfig_HeartbeatEnabled verifies heartbeat is enabled by default
func TestDefaultConfig_HeartbeatEnabled(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.Heartbeat.Enabled {
		t.Error("Heartbeat should be enabled by default")
	}
}

// TestDefaultConfig_WorkspacePath verifies workspace path is correctly set
func TestDefaultConfig_WorkspacePath(t *testing.T) {
	cfg := DefaultConfig()

	// Just verify the workspace is set, don't compare exact paths
	// since expandHome behavior may differ based on environment
	if cfg.Agents.Defaults.Workspace == "" {
		t.Error("Workspace should not be empty")
	}
}

// TestDefaultConfig_Model verifies model is set
func TestDefaultConfig_Model(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Agents.Defaults.Model == "" {
		t.Error("Model should not be empty")
	}
}

// TestDefaultConfig_MaxTokens verifies max tokens has default value
func TestDefaultConfig_MaxTokens(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Agents.Defaults.MaxTokens == 0 {
		t.Error("MaxTokens should not be zero")
	}
}

// TestDefaultConfig_MaxToolIterations verifies max tool iterations has default value
func TestDefaultConfig_MaxToolIterations(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Agents.Defaults.MaxToolIterations == 0 {
		t.Error("MaxToolIterations should not be zero")
	}
}

// TestDefaultConfig_Temperature verifies temperature has default value
func TestDefaultConfig_Temperature(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Agents.Defaults.Temperature == 0 {
		t.Error("Temperature should not be zero")
	}
}

// TestDefaultConfig_Gateway verifies gateway defaults
func TestDefaultConfig_Gateway(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Gateway.Host != "0.0.0.0" {
		t.Error("Gateway host should have default value")
	}
	if cfg.Gateway.Port == 0 {
		t.Error("Gateway port should have default value")
	}
}

// TestDefaultConfig_Providers verifies provider structure
func TestDefaultConfig_Providers(t *testing.T) {
	cfg := DefaultConfig()

	// Verify all providers are empty by default
	if cfg.Providers.Anthropic.APIKey != "" {
		t.Error("Anthropic API key should be empty by default")
	}
	if cfg.Providers.OpenAI.APIKey != "" {
		t.Error("OpenAI API key should be empty by default")
	}
	if cfg.Providers.OpenRouter.APIKey != "" {
		t.Error("OpenRouter API key should be empty by default")
	}
	if cfg.Providers.Groq.APIKey != "" {
		t.Error("Groq API key should be empty by default")
	}
	if cfg.Providers.Zhipu.APIKey != "" {
		t.Error("Zhipu API key should be empty by default")
	}
	if cfg.Providers.VLLM.APIKey != "" {
		t.Error("VLLM API key should be empty by default")
	}
	if cfg.Providers.Gemini.APIKey != "" {
		t.Error("Gemini API key should be empty by default")
	}
}

// TestDefaultConfig_Channels verifies channels are disabled by default
func TestDefaultConfig_Channels(t *testing.T) {
	cfg := DefaultConfig()

	// Verify all channels are disabled by default
	if cfg.Channels.WhatsApp.Enabled {
		t.Error("WhatsApp should be disabled by default")
	}
	if cfg.Channels.Telegram.Enabled {
		t.Error("Telegram should be disabled by default")
	}
	if cfg.Channels.Feishu.Enabled {
		t.Error("Feishu should be disabled by default")
	}
	if cfg.Channels.Discord.Enabled {
		t.Error("Discord should be disabled by default")
	}
	if cfg.Channels.MaixCam.Enabled {
		t.Error("MaixCam should be disabled by default")
	}
	if cfg.Channels.QQ.Enabled {
		t.Error("QQ should be disabled by default")
	}
	if cfg.Channels.DingTalk.Enabled {
		t.Error("DingTalk should be disabled by default")
	}
	if cfg.Channels.Slack.Enabled {
		t.Error("Slack should be disabled by default")
	}
}

// TestDefaultConfig_WebTools verifies web tools config
func TestDefaultConfig_WebTools(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.Tools.Spawn.Enabled {
		t.Error("Spawn tool should be enabled by default")
	}
	if !cfg.Tools.Subagent.Enabled {
		t.Error("Subagent tool should be enabled by default")
	}

	// Verify web tools defaults
	if cfg.Tools.Web.Brave.MaxResults != 5 {
		t.Error("Expected Brave MaxResults 5, got ", cfg.Tools.Web.Brave.MaxResults)
	}
	if cfg.Tools.Web.Brave.APIKey != "" {
		t.Error("Brave API key should be empty by default")
	}
	if cfg.Tools.Web.DuckDuckGo.MaxResults != 5 {
		t.Error("Expected DuckDuckGo MaxResults 5, got ", cfg.Tools.Web.DuckDuckGo.MaxResults)
	}
}

func TestDefaultConfig_SubagentToolTogglesParsing(t *testing.T) {
	jsonData := `{
		"tools": {
			"spawn": {"enabled": false},
			"subagent": {"enabled": true}
		}
	}`

	var cfg Config
	if err := json.Unmarshal([]byte(jsonData), &cfg); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if cfg.Tools.Spawn.Enabled {
		t.Error("Expected spawn tool disabled")
	}
	if !cfg.Tools.Subagent.Enabled {
		t.Error("Expected subagent tool enabled")
	}
}

// TestConfig_Complete verifies all config fields are set
func TestConfig_Complete(t *testing.T) {
	cfg := DefaultConfig()

	// Verify complete config structure
	if cfg.Agents.Defaults.Workspace == "" {
		t.Error("Workspace should not be empty")
	}
	if cfg.Agents.Defaults.Model == "" {
		t.Error("Model should not be empty")
	}
	if cfg.Agents.Defaults.Temperature == 0 {
		t.Error("Temperature should have default value")
	}
	if cfg.Agents.Defaults.MaxTokens == 0 {
		t.Error("MaxTokens should not be zero")
	}
	if cfg.Agents.Defaults.MaxToolIterations == 0 {
		t.Error("MaxToolIterations should not be zero")
	}
	if cfg.Gateway.Host != "0.0.0.0" {
		t.Error("Gateway host should have default value")
	}
	if cfg.Gateway.Port == 0 {
		t.Error("Gateway port should have default value")
	}
	if !cfg.Heartbeat.Enabled {
		t.Error("Heartbeat should be enabled by default")
	}
}

// TestAgentsConfig_UnmarshalJSON_NamedAgents verifies custom unmarshal for flat named agents
func TestAgentsConfig_UnmarshalJSON_NamedAgents(t *testing.T) {
	jsonData := `{
		"defaults": {
			"model": "test-model",
			"max_tokens": 4096,
			"temperature": 0.7,
			"history_message_threshold": 100
		},
		"subagents": {
			"max_tokens": 2048,
			"max_iterations": 10,
			"history_message_threshold": 50
		},
		"analyst": {
			"max_tokens": 8192,
			"history_message_threshold": 200
		},
		"researcher": {
			"max_tokens": 16384,
			"temperature": 0.5
		}
	}`

	var agents AgentsConfig
	if err := json.Unmarshal([]byte(jsonData), &agents); err != nil {
		t.Fatalf("Failed to unmarshal AgentsConfig: %v", err)
	}

	// Verify defaults
	if agents.Defaults.Model != "test-model" {
		t.Errorf("Expected defaults.model 'test-model', got '%s'", agents.Defaults.Model)
	}

	// Verify subagents
	if agents.Subagents.MaxTokens != 2048 {
		t.Errorf("Expected subagents.max_tokens 2048, got %d", agents.Subagents.MaxTokens)
	}

	// Verify named agents exist
	if len(agents.NamedAgents) != 2 {
		t.Errorf("Expected 2 named agents, got %d", len(agents.NamedAgents))
	}

	// Verify analyst config
	analyst, ok := agents.NamedAgents["analyst"]
	if !ok {
		t.Fatal("Expected 'analyst' named agent")
	}
	if analyst.MaxTokens != 8192 {
		t.Errorf("Expected analyst.max_tokens 8192, got %d", analyst.MaxTokens)
	}
	if analyst.HistoryMessageThreshold != 200 {
		t.Errorf("Expected analyst.history_message_threshold 200, got %d", analyst.HistoryMessageThreshold)
	}

	// Verify researcher config
	researcher, ok := agents.NamedAgents["researcher"]
	if !ok {
		t.Fatal("Expected 'researcher' named agent")
	}
	if researcher.MaxTokens != 16384 {
		t.Errorf("Expected researcher.max_tokens 16384, got %d", researcher.MaxTokens)
	}
	if researcher.Temperature != 0.5 {
		t.Errorf("Expected researcher.temperature 0.5, got %f", researcher.Temperature)
	}
}

// TestAgentsConfig_ResolveAgentConfig verifies config resolution priority
func TestAgentsConfig_ResolveAgentConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Add a named agent
	cfg.Agents.NamedAgents = map[string]NamedAgentConfig{
		"analyst": {
			MaxTokens:               8192,
			HistoryMessageThreshold: 50,
		},
	}
	cfg.Agents.Subagents = SubagentsConfig{
		MaxTokens:               2048,
		MaxIterations:           10,
		HistoryMessageThreshold: 30,
	}

	// Test 1: Named agent exists - should use named config merged with defaults
	resolved := cfg.Agents.ResolveAgentConfig("analyst")
	if resolved.MaxTokens != 8192 {
		t.Errorf("Expected analyst max_tokens 8192, got %d", resolved.MaxTokens)
	}
	if resolved.HistoryMessageThreshold != 50 {
		t.Errorf("Expected analyst history_threshold 50, got %d", resolved.HistoryMessageThreshold)
	}
	// Should inherit from defaults when not specified
	if resolved.MaxIterations != cfg.Agents.Defaults.MaxIterationsSubagent {
		t.Errorf("Expected analyst max_iterations from defaults, got %d", resolved.MaxIterations)
	}

	// Test 2: Empty name - should use subagents config
	resolved = cfg.Agents.ResolveAgentConfig("")
	if resolved.MaxTokens != 2048 {
		t.Errorf("Expected subagent max_tokens 2048, got %d", resolved.MaxTokens)
	}
	if resolved.MaxIterations != 10 {
		t.Errorf("Expected subagent max_iterations 10, got %d", resolved.MaxIterations)
	}
	if resolved.HistoryMessageThreshold != 30 {
		t.Errorf("Expected subagent history_threshold 30, got %d", resolved.HistoryMessageThreshold)
	}

	// Test 3: Unknown name - should use defaults (for backward compatibility)
	resolved = cfg.Agents.ResolveAgentConfig("unknown")
	if resolved.MaxTokens != cfg.Agents.Defaults.MaxTokensSubagent {
		t.Errorf("Expected unknown agent to use default max_tokens, got %d", resolved.MaxTokens)
	}
}

// TestAgentsConfig_ResolveAgentConfig_Priority verifies exact priority chain
func TestAgentsConfig_ResolveAgentConfig_Priority(t *testing.T) {
	cfg := &AgentsConfig{
		Defaults: AgentDefaults{
			MaxTokensSubagent:       1000,
			MaxIterationsSubagent:   5,
			Temperature:             0.9,
			HistoryMessageThreshold: 10,
		},
		Subagents: SubagentsConfig{
			MaxTokens:               2000,
			MaxIterations:           15,
			Temperature:             0.5,
			HistoryMessageThreshold: 20,
		},
		NamedAgents: map[string]NamedAgentConfig{
			"custom": {
				MaxTokens:               3000,
				HistoryMessageThreshold: 30,
				// MaxIterations and Temperature not set - should inherit from defaults
			},
		},
	}

	// Anonymous subagent: subagents > defaults
	anon := cfg.ResolveAgentConfig("")
	if anon.MaxTokens != 2000 {
		t.Errorf("Anonymous: expected max_tokens 2000 (from subagents), got %d", anon.MaxTokens)
	}
	if anon.Temperature != 0.5 {
		t.Errorf("Anonymous: expected temperature 0.5 (from subagents), got %f", anon.Temperature)
	}

	// Named agent: named > defaults (NOT subagents!)
	named := cfg.ResolveAgentConfig("custom")
	if named.MaxTokens != 3000 {
		t.Errorf("Named: expected max_tokens 3000 (from named), got %d", named.MaxTokens)
	}
	if named.HistoryMessageThreshold != 30 {
		t.Errorf("Named: expected history_threshold 30 (from named), got %d", named.HistoryMessageThreshold)
	}
	// These should come from defaults, NOT subagents
	if named.MaxIterations != 5 {
		t.Errorf("Named: expected max_iterations 5 (from defaults), got %d", named.MaxIterations)
	}
	if named.Temperature != 0.9 {
		t.Errorf("Named: expected temperature 0.9 (from defaults), got %f", named.Temperature)
	}
}

func TestLoadConfig_SyntaxErrorIncludesLineContext(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	bad := "{\n  \"gateway\": {\n    \"host\": \"0.0.0.0\" \"port\": 18790\n  }\n}\n"
	if err := os.WriteFile(path, []byte(bad), 0644); err != nil {
		t.Fatalf("write bad config: %v", err)
	}

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected syntax error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "line") || !strings.Contains(msg, "column") {
		t.Fatalf("expected line/column in error, got: %s", msg)
	}
	if !strings.Contains(msg, "\"host\": \"0.0.0.0\" \"port\": 18790") {
		t.Fatalf("expected source line in error, got: %s", msg)
	}
	if !strings.Contains(msg, "^") {
		t.Fatalf("expected caret marker in error, got: %s", msg)
	}
}
