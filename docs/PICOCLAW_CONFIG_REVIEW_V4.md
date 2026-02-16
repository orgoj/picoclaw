# PicoClaw Configuration KISS Plan V4 - Review

**Review Date:** 2026-02-16  
**Status:** ⚠️ PLAN CORRECT, IMPLEMENTATION PENDING

## Summary

The KISS plan V4 is well-designed and follows KISS principles. However, the **implementation has NOT been executed yet** - the new config fields do not exist in the codebase. Additionally, there is one deviation from requirements regarding environment variables.

---

## Verification Results

### ✅ 1. Plan Respects Existing Config Structure

**VERIFIED** - The plan correctly preserves existing fields:

| Existing Field | Location | Usage |
|----------------|----------|-------|
| `max_tool_iterations` | `AgentDefaults.MaxToolIterations` | Main loop (`loop.go:90`) |
| `max_tokens` | `AgentDefaults.MaxTokens` | Main loop context window (`loop.go:89`) |

**Code Evidence:**
```go
// loop.go:88-90
maxIterations:  cfg.Agents.Defaults.MaxToolIterations,
contextWindow:  cfg.Agents.Defaults.MaxTokens,
```

The plan explicitly states: "Do NOT change existing MaxToolIterations or MaxTokens - main loop uses these as-is."

---

### ✅ 2. New Parameters Correctly Defined

**PLAN IS CORRECT** - The proposed parameters are:

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `max_iterations_subagent` | int | 20 | Max tool iterations for subagents |
| `max_tokens_subagent` | int | 4096 | Max LLM response tokens for subagents |
| `llm_timeout` | int | 120 | LLM API timeout in seconds |

**Current State (NOT IMPLEMENTED):**
- `subagent.go:47` - hardcoded: `maxIterations: 10`
- `subagent.go:115` - hardcoded: `"max_tokens": 4096`
- `http_provider.go:31` - hardcoded: `Timeout: 120 * time.Second`

---

### ❌ 3. Environment Variables Issue

**PROBLEM FOUND** - The KISS plan includes environment variable tags:

```go
MaxIterationsSubagent int `json:"max_iterations_subagent" env:"PICOCLAW_AGENTS_DEFAULTS_MAX_ITERATIONS_SUBAGENT"`
MaxTokensSubagent     int `json:"max_tokens_subagent" env:"PICOCLAW_AGENTS_DEFAULTS_MAX_TOKENS_SUBAGENT"`
LLMTimeout            int `json:"llm_timeout" env:"PICOCLAW_AGENTS_DEFAULTS_LLM_TIMEOUT"`
```

**Requirement says:** "Žádné env vars - vše v config.json" (No env vars - everything in config.json)

**Recommendation:** Remove `env:"..."` tags from the plan, keep only JSON tags:

```go
MaxIterationsSubagent int `json:"max_iterations_subagent"`
MaxTokensSubagent     int `json:"max_tokens_subagent"`
LLMTimeout            int `json:"llm_timeout"`
```

---

### ✅ 4. KISS Principles Verification

**VERIFIED** - The plan follows KISS:

| Aspect | Assessment |
|--------|------------|
| Minimal changes | ✅ Only 5 files to modify |
| No over-engineering | ✅ Simple field additions, no new abstractions |
| Reuses existing patterns | ✅ Follows existing config structure |
| No new dependencies | ✅ Uses existing `env` and `json` tags |
| Clear separation | ✅ Main loop vs subagent params are distinct |

---

## Implementation Status

| File | Status | Action Needed |
|------|--------|---------------|
| `pkg/config/config.go` | ❌ Not modified | Add 3 fields to `AgentDefaults` |
| `pkg/tools/subagent.go` | ❌ Not modified | Pass config, use new fields |
| `pkg/agent/loop.go` | ❌ Not modified | Pass cfg to `NewSubagentManager` |
| `pkg/providers/http_provider.go` | ❌ Not modified | Add timeout from config |
| `config/config.example.json` | ❌ Not modified | Add 3 params to `agents.defaults` |

---

## Recommendations

1. **Remove env tags** from the plan to comply with "no env vars" requirement
2. **Proceed with implementation** - the plan is otherwise correct
3. **Update config.example.json** to include the new params for documentation

---

## Corrected Implementation Snippet

### config.go (AgentDefaults struct)
```go
type AgentDefaults struct {
    Workspace             string  `json:"workspace" env:"PICOCLAW_AGENTS_DEFAULTS_WORKSPACE"`
    RestrictToWorkspace   bool    `json:"restrict_to_workspace" env:"PICOCLAW_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE"`
    Provider              string  `json:"provider" env:"PICOCLAW_AGENTS_DEFAULTS_PROVIDER"`
    Model                 string  `json:"model" env:"PICOCLAW_AGENTS_DEFAULTS_MODEL"`
    MaxTokens             int     `json:"max_tokens" env:"PICOCLAW_AGENTS_DEFAULTS_MAX_TOKENS"`
    Temperature           float64 `json:"temperature" env:"PICOCLAW_AGENTS_DEFAULTS_TEMPERATURE"`
    MaxToolIterations     int     `json:"max_tool_iterations" env:"PICOCLAW_AGENTS_DEFAULTS_MAX_TOOL_ITERATIONS"`
    // NEW - no env tags per requirements
    MaxIterationsSubagent int     `json:"max_iterations_subagent"`
    MaxTokensSubagent     int     `json:"max_tokens_subagent"`
    LLMTimeout            int     `json:"llm_timeout"`
}
```

### config.go (DefaultConfig)
```go
Defaults: AgentDefaults{
    Workspace:             "~/.picoclaw/workspace",
    RestrictToWorkspace:   true,
    Provider:              "",
    Model:                 "glm-4.7",
    MaxTokens:             8192,
    Temperature:           0.7,
    MaxToolIterations:     20,
    // NEW defaults
    MaxIterationsSubagent: 20,
    MaxTokensSubagent:     4096,
    LLMTimeout:            120,
},
```

---

## Conclusion

| Criterion | Status |
|-----------|--------|
| Respects existing config | ✅ PASS |
| New params defined correctly | ✅ PASS |
| No env vars | ❌ FAIL - plan includes env tags |
| KISS implementation | ✅ PASS |
| Implementation done | ❌ NOT DONE |

**Overall:** Plan is 90% correct. Remove env tags and proceed with implementation.
