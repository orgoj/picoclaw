# Agent Config Hierarchy Analysis Report

**Date:** 2026-02-16
**Task:** Design 3-level config structure for agents
**Status:** Analysis Phase (No Implementation)

---

## Executive Summary

This report analyzes the current config structure and proposes a 3-level hierarchy for agent configuration:
- `defaults` → applies to ALL agents
- `subagents` → overrides for subagents
- `named` → per-named-agent overrides

**Priority Rule:** `named > subagents > defaults`

---

## 1. Current Config Structure

### 1.1 Config File Location
- **Primary:** `pkg/config/config.go`
- **Loaded by:** `LoadConfig(path string)` from JSON file

### 1.2 Current Structure (Single Level)

```go
// pkg/config/config.go
type AgentsConfig struct {
    Defaults AgentDefaults `json:"defaults"`
}

type AgentDefaults struct {
    Workspace               string  `json:"workspace"`
    RestrictToWorkspace     bool    `json:"restrict_to_workspace"`
    Provider                string  `json:"provider"`
    Model                   string  `json:"model"`
    MaxTokens               int     `json:"max_tokens"`            // Main agent max output
    ContextWindow           int     `json:"context_window"`
    Temperature             float64 `json:"temperature"`
    MaxToolIterations       int     `json:"max_tool_iterations"`   // Main agent max iterations
    MaxIterationsSubagent   int     `json:"max_iterations_subagent"` // Subagent max iterations
    MaxTokensSubagent       int     `json:"max_tokens_subagent"`     // Subagent max output
    LLMTimeout              int     `json:"llm_timeout"`
    MemoryThreshold         float64 `json:"memory_threshold"`
    HistoryMessageThreshold int     `json:"history_message_threshold"`
}
```

### 1.3 Current JSON Example

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "provider": "openai",
      "model": "gpt-4.1-mini",
      "max_tokens": 8192,
      "context_window": 131072,
      "temperature": 0.7,
      "max_tool_iterations": 50,
      "max_iterations_subagent": 20,
      "max_tokens_subagent": 4096,
      "llm_timeout": 120,
      "memory_threshold": 0.8,
      "history_message_threshold": 100
    }
  }
}
```

---

## 2. Where Config is Used

### 2.1 Main Agent (pkg/agent/loop.go)

```go
func NewAgentLoop(cfg *config.Config, msgBus *bus.MessageBus, provider providers.LLMProvider) *AgentLoop {
    // Main agent reads from cfg.Agents.Defaults
    return &AgentLoop{
        model:                   cfg.Agents.Defaults.Model,
        contextWindow:           cfg.Agents.Defaults.ContextWindow,
        maxIterations:           cfg.Agents.Defaults.MaxToolIterations,  // Main agent iterations
        maxTokens:               cfg.Agents.Defaults.MaxTokens,           // Main agent tokens
        temperature:             cfg.Agents.Defaults.Temperature,
        llmTimeout:              cfg.Agents.Defaults.LLMTimeout,
        memoryThreshold:         cfg.Agents.Defaults.MemoryThreshold,
        historyMessageThreshold: cfg.Agents.Defaults.HistoryMessageThreshold,
        // ...
    }
}
```

### 2.2 Subagent Manager (pkg/tools/subagent.go)

```go
func NewSubagentManager(provider providers.LLMProvider, cfg *config.Config, workspace string, bus *bus.MessageBus) *SubagentManager {
    // Subagent reads from cfg.Agents.Defaults (flat structure)
    contextLimit := cfg.Agents.Defaults.MaxTokensSubagent * 20  // Derived

    return &SubagentManager{
        defaultModel:  cfg.Agents.Defaults.Model,
        cfg:           cfg,
        maxIterations: cfg.Agents.Defaults.MaxIterationsSubagent,  // Subagent iterations
        maxTokens:     cfg.Agents.Defaults.MaxTokensSubagent,       // Subagent tokens
        contextLimit:  contextLimit,
        // ...
    }
}
```

### 2.3 Named Agents (Current State)

**NOT IMPLEMENTED** - Named agents use the same `SubagentManager` with a `name` parameter, but there's no per-name config override mechanism.

Current named agent behavior:
```go
// In buildSubagentSystemPrompt()
if name != "" {
    agentDir := filepath.Join(sm.workspace, "agents", name)
    // Loads identity from agents/<name>/AGENTS.md
    // Loads memory from agents/<name>/memory/MEMORY.md
}
```

---

## 3. Proposed New Structure

### 3.1 New Config Types

```go
// pkg/config/config.go

// AgentConfig holds config for a specific agent type
type AgentConfig struct {
    HistoryMessageThreshold *int     `json:"history_message_threshold,omitempty"`
    MaxTokens               *int     `json:"max_tokens,omitempty"`
    MaxIterations           *int     `json:"max_iterations,omitempty"`
    Temperature             *float64 `json:"temperature,omitempty"`
    ContextWindow           *int     `json:"context_window,omitempty"`
}

type AgentsConfig struct {
    Defaults  AgentDefaults          `json:"defaults"`   // Base defaults for ALL
    Subagents *AgentConfig           `json:"subagents,omitempty"`  // Subagent overrides
    Named     map[string]AgentConfig `json:"named,omitempty"`      // Per-name overrides
}
```

### 3.2 Proposed YAML/JSON Example

```yaml
agents:
  defaults:           # applies to ALL agents (main + subagents + named)
    history_message_threshold: 100
    max_tokens: 4096
    max_iterations: 20
    temperature: 0.7
    context_window: 131072
    
  subagents:          # overrides for subagents (applies to all subagents)
    history_message_threshold: 100
    max_tokens: 8192
    max_iterations: 20
    
  named:              # per-named-agent overrides
    "analyst":
      history_message_threshold: 50   # Shorter history for focused tasks
      max_tokens: 2048
    "researcher":
      history_message_threshold: 200  # Longer history for research
      max_tokens: 8192
    "coder":
      max_iterations: 30              # More iterations for code tasks
```

JSON equivalent:
```json
{
  "agents": {
    "defaults": {
      "history_message_threshold": 100,
      "max_tokens": 4096,
      "max_iterations": 20,
      "temperature": 0.7,
      "context_window": 131072
    },
    "subagents": {
      "history_message_threshold": 100,
      "max_tokens": 8192,
      "max_iterations": 20
    },
    "named": {
      "analyst": {
        "history_message_threshold": 50,
        "max_tokens": 2048
      },
      "researcher": {
        "history_message_threshold": 200,
        "max_tokens": 8192
      },
      "coder": {
        "max_iterations": 30
      }
    }
  }
}
```

---

## 4. Resolution Logic (Priority: named > subagents > defaults)

### 4.1 Config Resolver Function

```go
// pkg/config/config.go

// ResolveAgentConfig returns the effective config for an agent
// Priority: named > subagents > defaults
func (c *AgentsConfig) ResolveAgentConfig(isSubagent bool, name string) ResolvedAgentConfig {
    result := ResolvedAgentConfig{
        HistoryMessageThreshold: c.Defaults.HistoryMessageThreshold,
        MaxTokens:               c.Defaults.MaxTokens,
        MaxIterations:           c.Defaults.MaxToolIterations,
        Temperature:             c.Defaults.Temperature,
        ContextWindow:           c.Defaults.ContextWindow,
    }
    
    // Apply subagent overrides (if this is a subagent)
    if isSubagent && c.Subagents != nil {
        if c.Subagents.HistoryMessageThreshold != nil {
            result.HistoryMessageThreshold = *c.Subagents.HistoryMessageThreshold
        }
        if c.Subagents.MaxTokens != nil {
            result.MaxTokens = *c.Subagents.MaxTokens
        }
        if c.Subagents.MaxIterations != nil {
            result.MaxIterations = *c.Subagents.MaxIterations
        }
        if c.Subagents.Temperature != nil {
            result.Temperature = *c.Subagents.Temperature
        }
        if c.Subagents.ContextWindow != nil {
            result.ContextWindow = *c.Subagents.ContextWindow
        }
    }
    
    // Apply named overrides (highest priority)
    if name != "" && c.Named != nil {
        if named, ok := c.Named[name]; ok {
            if named.HistoryMessageThreshold != nil {
                result.HistoryMessageThreshold = *named.HistoryMessageThreshold
            }
            if named.MaxTokens != nil {
                result.MaxTokens = *named.MaxTokens
            }
            if named.MaxIterations != nil {
                result.MaxIterations = *named.MaxIterations
            }
            if named.Temperature != nil {
                result.Temperature = *named.Temperature
            }
            if named.ContextWindow != nil {
                result.ContextWindow = *named.ContextWindow
            }
        }
    }
    
    return result
}

type ResolvedAgentConfig struct {
    HistoryMessageThreshold int
    MaxTokens               int
    MaxIterations           int
    Temperature             float64
    ContextWindow           int
}
```

---

## 5. Files to Modify

| File | Changes Required |
|------|------------------|
| `pkg/config/config.go` | Add `AgentConfig` struct, update `AgentsConfig`, add `ResolveAgentConfig()` method |
| `pkg/agent/loop.go` | Use `cfg.Agents.ResolveAgentConfig(false, "")` for main agent |
| `pkg/tools/subagent.go` | Use `cfg.Agents.ResolveAgentConfig(true, name)` for subagents, pass name parameter |
| `pkg/tools/subagent_tools.go` | Update SubagentTool to pass name to manager for config resolution |

---

## 6. Backward Compatibility

### 6.1 Old Config Compatibility

**WILL OLD CONFIGS STILL WORK?** ✅ YES

Old config format (flat structure):
```json
{
  "agents": {
    "defaults": {
      "max_tokens": 8192,
      "max_iterations_subagent": 20,
      "max_tokens_subagent": 4096
    }
  }
}
```

Migration strategy:
1. Keep `MaxIterationsSubagent` and `MaxTokensSubagent` in `AgentDefaults` for backward compat
2. In `ResolveAgentConfig()`, if `Subagents` is nil, fall back to old fields:
   ```go
   if isSubagent && c.Subagents == nil {
       // Fallback to old fields for backward compatibility
       result.MaxIterations = c.Defaults.MaxIterationsSubagent
       result.MaxTokens = c.Defaults.MaxTokensSubagent
   }
   ```

### 6.2 Migration Path

1. **Phase 1:** Add new fields, keep old fields, add resolver with fallback
2. **Phase 2:** Update internal code to use resolver
3. **Phase 3:** Deprecate old fields (warn in logs)
4. **Phase 4:** Remove old fields (major version bump)

---

## 7. Implementation Complexity Estimate

| Component | Lines of Code | Risk |
|-----------|---------------|------|
| `AgentConfig` struct + resolver | ~60 lines | Low |
| Update `AgentsConfig` | ~10 lines | Low |
| Update main agent (`loop.go`) | ~15 lines | Low |
| Update subagent manager | ~30 lines | Medium |
| Backward compat logic | ~20 lines | Medium |
| **Total** | **~135 lines** | **Medium** |

**Estimated file size impact:** +2KB

---

## 8. Open Questions

1. **Should `model` be configurable per named agent?**
   - Current: All agents use same model from `defaults.model`
   - Proposed: Allow per-named-agent model override?

2. **Should `provider` be configurable per named agent?**
   - More complex - different providers have different APIs
   - Recommendation: NO, keep provider global

3. **Environment variable support:**
   - Current: `env:"PICOCLAW_AGENTS_DEFAULTS_*"` tags
   - New: How to support env vars for named agents?
   - Option: `PICOCLAW_AGENTS_NAMED_ANALYST_MAX_TOKENS=2048`

---

## 9. Recommendation

**Proceed with implementation** but keep it minimal:

1. Add `AgentConfig` struct with pointer fields (for optional override)
2. Add `Subagents *AgentConfig` and `Named map[string]AgentConfig`
3. Add `ResolveAgentConfig()` method with fallback for backward compat
4. Update `SubagentManager` to use resolved config

**KISS Constraint Check:**
- ~135 lines is acceptable (<200 line guideline)
- No mega-architecture changes
- Backward compatible

---

## Self-Rating

| Aspect | Score (1-5) | Notes |
|--------|-------------|-------|
| Completeness | 5 | All files identified, code shown |
| Accuracy | 5 | Code snippets verified from actual source |
| Feasibility | 4 | Straightforward implementation, minor risk |
| Clarity | 5 | Clear examples, priority rules explicit |
| Backward Compat | 5 | Clear migration strategy |

**Overall Confidence:** 4.8/5

---

## Appendix A: Current Code Snippets

### A.1 AgentDefaults (current)

```go
type AgentDefaults struct {
    Workspace               string  `json:"workspace" env:"PICOCLAW_AGENTS_DEFAULTS_WORKSPACE"`
    RestrictToWorkspace     bool    `json:"restrict_to_workspace" env:"PICOCLAW_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE"`
    Provider                string  `json:"provider" env:"PICOCLAW_AGENTS_DEFAULTS_PROVIDER"`
    Model                   string  `json:"model" env:"PICOCLAW_AGENTS_DEFAULTS_MODEL"`
    MaxTokens               int     `json:"max_tokens" env:"PICOCLAW_AGENTS_DEFAULTS_MAX_TOKENS"`
    ContextWindow           int     `json:"context_window" env:"PICOCLAW_AGENTS_DEFAULTS_CONTEXT_WINDOW"`
    Temperature             float64 `json:"temperature" env:"PICOCLAW_AGENTS_DEFAULTS_TEMPERATURE"`
    MaxToolIterations       int     `json:"max_tool_iterations" env:"PICOCLAW_AGENTS_DEFAULTS_MAX_TOOL_ITERATIONS"`
    MaxIterationsSubagent   int     `json:"max_iterations_subagent"`
    MaxTokensSubagent       int     `json:"max_tokens_subagent"`
    LLMTimeout              int     `json:"llm_timeout"`
    MemoryThreshold         float64 `json:"memory_threshold" env:"PICOCLAW_AGENTS_DEFAULTS_MEMORY_THRESHOLD"`
    HistoryMessageThreshold int     `json:"history_message_threshold" env:"PICOCLAW_AGENTS_DEFAULTS_HISTORY_MESSAGE_THRESHOLD"`
}
```

### A.2 SubagentManager Creation (current)

```go
func NewSubagentManager(provider providers.LLMProvider, cfg *config.Config, workspace string, bus *bus.MessageBus) *SubagentManager {
    contextLimit := cfg.Agents.Defaults.MaxTokensSubagent * 20
    if contextLimit <= 0 {
        contextLimit = 4096 * 20
    }

    return &SubagentManager{
        tasks:         make(map[string]*SubagentTask),
        provider:      provider,
        defaultModel:  cfg.Agents.Defaults.Model,
        bus:           bus,
        workspace:     workspace,
        tools:         NewToolRegistry(),
        cfg:           cfg,
        maxIterations: cfg.Agents.Defaults.MaxIterationsSubagent,
        maxTokens:     cfg.Agents.Defaults.MaxTokensSubagent,
        contextLimit:  contextLimit,
        nextID:        1,
    }
}
```

---

*Report generated by subagent analysis task*
