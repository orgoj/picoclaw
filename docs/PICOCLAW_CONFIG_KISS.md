# PicoClaw Configuration - KISS Plan

## Overview

Add subagent-specific parameters and LLM timeout to `agents.defaults`. Main loop keeps existing `max_tool_iterations` and `max_tokens` unchanged.

## Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `max_iterations_subagent` | int | 20 | Max tool iterations for subagents (main uses `max_tool_iterations`) |
| `max_tokens_subagent` | int | 4096 | Max LLM response tokens for subagents (main uses `max_tokens`) |
| `llm_timeout` | int | 120 | LLM API timeout in seconds (shared for all calls) |

**Note:** Main loop continues using existing `max_tool_iterations` and `max_tokens` fields. Only subagents get new dedicated parameters.

## Config Example

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "restrict_to_workspace": true,
      "model": "glm-4.7",
      "temperature": 0.7,
      "max_tool_iterations": 20,
      "max_tokens": 8192,
      "max_iterations_subagent": 20,
      "max_tokens_subagent": 4096,
      "llm_timeout": 120
    }
  }
}
```

## Implementation

### 1. Add Config Fields (`pkg/config/config.go`)

```go
// Add to AgentDefaults struct (no env tags per requirements):
MaxIterationsSubagent int `json:"max_iterations_subagent"`
MaxTokensSubagent     int `json:"max_tokens_subagent"`
LLMTimeout            int `json:"llm_timeout"`

// Add defaults:
MaxIterationsSubagent: 20,
MaxTokensSubagent:     4096,
LLMTimeout:            120,
```

**Important:** Do NOT change existing `MaxToolIterations` or `MaxTokens` - main loop uses these as-is.

### 2. Update Subagent (`pkg/tools/subagent.go`)

Pass config to SubagentManager and use new fields:

```go
// Change NewSubagentManager signature:
func NewSubagentManager(provider providers.LLMProvider, cfg *config.Config, workspace string, bus *bus.MessageBus) *SubagentManager {
    return &SubagentManager{
        tasks:         make(map[string]*SubagentTask),
        provider:      provider,
        defaultModel:  cfg.Agents.Defaults.Model,
        bus:           bus,
        workspace:     workspace,
        tools:         NewToolRegistry(),
        maxIterations: cfg.Agents.Defaults.MaxIterationsSubagent,
        nextID:        1,
    }
}
```

Update RunToolLoop calls to use config:

```go
loopResult, err := RunToolLoop(ctx, ToolLoopConfig{
    Provider:      sm.provider,
    Model:         sm.defaultModel,
    Tools:         tools,
    MaxIterations: maxIter,
    LLMOptions: map[string]any{
        "max_tokens":  cfg.Agents.Defaults.MaxTokensSubagent,
        "temperature": 0.7,
    },
}, messages, task.OriginChannel, task.OriginChatID)
```

### 3. Update SubagentTool (`pkg/tools/subagent.go`)

Update synchronous subagent to use config:

```go
loopResult, err := RunToolLoop(ctx, ToolLoopConfig{
    Provider:      sm.provider,
    Model:         sm.defaultModel,
    Tools:         tools,
    MaxIterations: maxIter,
    LLMOptions: map[string]any{
        "max_tokens":  sm.manager.maxTokensSubagent,  // or read from config
        "temperature": 0.7,
    },
}, messages, t.originChannel, t.originChatID)
```

### 4. Update Agent Loop Initialization (`pkg/agent/loop.go`)

Change SubagentManager creation:

```go
// Create subagent manager with config
subagentManager := tools.NewSubagentManager(provider, cfg, workspace, msgBus)
```

### 5. Update HTTP Provider (`pkg/providers/http_provider.go`)

Add context with timeout:

```go
func (p *HTTPProvider) Chat(ctx context.Context, messages []Message, toolDefs []ToolDef, model string, options map[string]interface{}) (*ChatResponse, error) {
    // Get timeout from config (passed via provider constructor)
    timeout := time.Duration(p.llmTimeout) * time.Second
    if timeout == 0 {
        timeout = 120 * time.Second  // default
    }

    // Create context with timeout
    if ctx, cancel := context.WithTimeout(ctx, timeout); cancel != nil {
        ctx = ctx
    }
    // ... rest of implementation
}
```

Add llmTimeout field to HTTPProvider struct.

## Files to Modify

1. `pkg/config/config.go` - Add 3 fields to AgentDefaults
2. `pkg/tools/subagent.go` - Pass config, use new fields, add maxTokensSubagent field
3. `pkg/agent/loop.go` - Pass cfg to NewSubagentManager
4. `pkg/providers/http_provider.go` - Add timeout support
5. `config/config.example.json` - Add 3 params to agents.defaults

## Done
