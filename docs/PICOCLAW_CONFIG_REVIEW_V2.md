# PicoClaw Configuration Review - KISS Analysis

## Scope

This review focuses only on the 3 parameters that need implementation:
- `max_iterations` (default 50)
- `max_tokens` (default 8192)
- `llm_timeout` (default 120s)

## Problems with Proposed Changes

### 1. Parameter Duplication (Critical)

**Problem:** The plan introduces new `execution.*` parameters while keeping existing `agents.defaults.*` parameters:

| Existing | Proposed New | Same Purpose? |
|----------|--------------|---------------|
| `agents.defaults.max_tool_iterations` | `execution.max_iterations` | YES |
| `agents.defaults.max_tokens` | `execution.max_tokens` | YES |

**Why it's wrong:**
- Two config paths for the same thing = confusion
- Users won't know which one to use
- Fallback chain adds unnecessary complexity
- Maintenance burden: need to keep both in sync

**KISS Solution:** 
- Either rename existing params OR add new ones, not both
- Prefer extending existing `agents.defaults` over new `execution` section

---

### 2. Over-Engineering: Too Many Sub-parameters

**Problem:** The plan splits simple parameters into nested hierarchies:

```json
"execution": {
  "max_iterations": 50,
  "max_tokens": 8192,
  "llm_timeout": 120,
  "subagent": {
    "max_iterations": 10,
    "max_tokens": 4096
  },
  "summarization": {
    "max_tokens": 1024,
    "timeout": 120
  }
}
```

**Why it's wrong:**
- 7 parameters when user only asked for 3
- Subagent/summarization tuning is advanced use case
- Most users just want to set "max tokens" globally
- YAGNI (You Ain't Gonna Need It)

**KISS Solution:**
```json
"execution": {
  "max_iterations": 50,
  "max_tokens": 8192,
  "llm_timeout": 120
}
```
That's it. If subagent needs different values, derive them from main values (e.g., `max_tokens / 2`).

---

### 3. Inconsistent Default Values

**Problem:** Default changed without justification:

| Parameter | Old Default | New Default | Why? |
|-----------|-------------|-------------|------|
| `max_iterations` | 20 | 50 | Not explained |

**Why it's wrong:**
- Changing defaults can break existing behavior
- No rationale provided for the increase
- Could lead to longer execution times for existing users

**KISS Solution:**
- Keep existing defaults unless there's a documented reason to change
- Or: document WHY the change is needed

---

### 4. Environment Variable Naming

**Problem:** Environment variable names are verbose:

```
PICOCLAW_EXECUTION_MAX_ITERATIONS
PICOCLAW_EXECUTION_MAX_TOKENS
PICOCLAW_EXECUTION_LLM_TIMEOUT
PICOCLAW_EXECUTION_SUBAGENT_MAX_ITERATIONS
PICOCLAW_EXECUTION_SUBAGENT_MAX_TOKENS
PICOCLAW_EXECUTION_SUMMARIZATION_MAX_TOKENS
PICOCLAW_EXECUTION_SUMMARIZATION_TIMEOUT
```

**Why it's wrong:**
- Excessively long
- Redundant `EXECUTION` in every name
- Hard to type, hard to remember

**KISS Solution:**
```
PICOCLAW_MAX_ITERATIONS
PICOCLAW_MAX_TOKENS
PICOCLAW_LLM_TIMEOUT
```
Or even shorter with a prefix convention.

---

### 5. Separate Timeout for Summarization

**Problem:** The plan adds a separate `summarization.timeout` (120s) when `llm_timeout` already exists.

**Why it's wrong:**
- Users set `llm_timeout` expecting it to apply to ALL LLM calls
- Summarization is just another LLM call
- Adds cognitive overhead

**KISS Solution:**
- Use single `llm_timeout` for ALL LLM operations
- If summarization needs different timeout, derive it internally (e.g., `llm_timeout / 2`)

---

### 6. Backward Compatibility Complexity

**Problem:** The fallback chain adds complexity:

```
execution.max_iterations → agents.defaults.max_tool_iterations → default (50)
```

**Why it's wrong:**
- Requires migration logic
- Two places to check for value
- Testing complexity increases
- Code becomes harder to reason about

**KISS Solution:**
- Single source of truth
- Clean migration (one-time) or just use new path

---

## Recommended Minimal Implementation

### Config Structure

```json
{
  "execution": {
    "max_iterations": 50,
    "max_tokens": 8192,
    "llm_timeout": 120
  }
}
```

### Environment Variables

```
PICOCLAW_MAX_ITERATIONS=50
PICOCLAW_MAX_TOKENS=8192
PICOCLAW_LLM_TIMEOUT=120
```

### Go Struct

```go
type ExecutionConfig struct {
    MaxIterations int `json:"max_iterations" env:"PICOCLAW_MAX_ITERATIONS"`
    MaxTokens     int `json:"max_tokens" env:"PICOCLAW_MAX_TOKENS"`
    LLMTimeout    int `json:"llm_timeout" env:"PICOCLAW_LLM_TIMEOUT"`
}

// Defaults
Execution: ExecutionConfig{
    MaxIterations: 50,
    MaxTokens:     8192,
    LLMTimeout:    120,
}
```

### Internal Derivation (No Extra Config)

```go
// For subagents - derive from main config
subagentMaxTokens := config.Execution.MaxTokens / 2  // 4096

// For summarization - derive from main config  
summarizationMaxTokens := config.Execution.MaxTokens / 8  // 1024
summarizationTimeout := config.Execution.LLMTimeout / 2  // 60s
```

---

## Summary

| Issue | Severity | Fix |
|-------|----------|-----|
| Parameter duplication | Critical | Use single config path |
| Over-engineering sub-params | High | Remove subagent/summarization params |
| Unexplained default changes | Medium | Keep or document |
| Verbose env vars | Low | Shorten names |
| Separate summarization timeout | Medium | Derive from llm_timeout |
| Complex fallback chain | High | Single source of truth |

**Bottom Line:** The plan adds too much complexity for what should be a simple "make 3 values configurable" change. Remove the nesting, remove the duplication, derive subagent/summarization values internally.
