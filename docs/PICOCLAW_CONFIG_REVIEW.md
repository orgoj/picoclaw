# PicoClaw Config Plan Review

**Review Date:** 2026-02-16  
**Document Reviewed:** PICOCLAW_CONFIG_PLAN.md  
**Reviewer:** Automated Code Review

---

## Executive Summary

This review identifies issues, edge cases, and improvement opportunities in the PicoClaw configuration plan. The analysis compares proposed configuration changes against the actual implementation in the codebase.

**Overall Assessment:** The config plan is generally well-structured but contains several discrepancies with the actual implementation and missing validation logic.

---

## Critical Issues

### 1. Model Field Default Value Inconsistency

**Location:** Config Plan Section 2.1 - Agent Defaults

**Issue:** The config plan specifies `glm-4.7` as the default model, but the `Provider` field defaults to empty string.

**Actual Code (`pkg/config/config.go`):**
```go
Provider:  "",  // Empty by default
Model:     "glm-4.7",
```

**Problem:** When `Provider` is empty, the `CreateProvider()` function falls back to model-name detection. For `glm-4.7`, this correctly routes to Zhipu, but:
- No validation that the user has configured a Zhipu API key
- Error message at runtime is unclear: `"no API key configured for model: glm-4.7"`

**Recommendation:**
- Either set `Provider: "zhipu"` as default when `Model: "glm-4.7"`
- Or add startup validation that checks if default model has valid provider credentials

---

### 2. Example Config Missing `provider` Field

**Location:** `config/config.example.json`

**Issue:** The example config is missing the `provider` field in `agents.defaults`:

```json
"agents": {
  "defaults": {
    "workspace": "~/.picoclaw/workspace",
    "restrict_to_workspace": true,
    "model": "glm-4.7",
    // Missing: "provider": "zhipu"
  }
}
```

**Actual Code expects this field:**
```go
Provider string `json:"provider" env:"PICOCLAW_AGENTS_DEFAULTS_PROVIDER"`
```

**Recommendation:** Add `provider` field to example config with documented options.

---

### 3. Port Conflict: MaixCam and Gateway Same Default Port

**Location:** Config Plan Section 2.2, 2.5

**Issue:** Both `maixcam.port` and `gateway.port` default to `18790`.

```json
"maixcam": {
  "port": 18790
},
"gateway": {
  "port": 18790
}
```

**Problem:** If both are enabled, they will conflict. The code doesn't validate this.

**Recommendation:**
- Change MaixCam default port to `18791` or
- Add startup validation to detect port conflicts

---

### 4. Missing Providers in Example Config

**Location:** `config/config.example.json` vs actual `ProvidersConfig`

**Missing Providers in Example:**
- `shengsuanyun`
- `deepseek`
- `github_copilot`

**Actual Code (`pkg/config/config.go`):**
```go
type ProvidersConfig struct {
    // ... existing ...
    ShengSuanYun  ProviderConfig `json:"shengsuanyun"`
    DeepSeek      ProviderConfig `json:"deepseek"`
    GitHubCopilot ProviderConfig `json:"github_copilot"`
}
```

**Recommendation:** Add all supported providers to the example config for completeness.

---

## Edge Cases

### 5. Empty AllowFrom Behavior

**Location:** All channel configs

**Issue:** When `allow_from` is an empty array `[]`, the behavior is undefined in the config plan.

**Questions to Clarify:**
- Does empty array mean "allow nobody" or "allow everyone"?
- Should there be a wildcard option like `["*"]`?

**Current Implementation Concern:** Security implication if empty defaults to "allow all".

**Recommendation:** 
- Document explicit behavior for empty arrays
- Consider adding `"allow_all": false` as an explicit flag

---

### 6. Temperature Validation Missing

**Location:** Agent defaults

**Issue:** No validation for `temperature` field.

**Edge Cases:**
- `temperature: -0.5` → Should be rejected
- `temperature: 3.0` → Some providers cap at 2.0
- `temperature: 0` → Valid for deterministic output

**Actual Code:**
```go
Temperature float64 `json:"temperature" env:"PICOCLAW_AGENTS_DEFAULTS_TEMPERATURE"`
```

No validation exists.

**Recommendation:** Add validation: `0.0 <= temperature <= 2.0`

---

### 7. MaxToolIterations = 0 Edge Case

**Location:** Agent defaults

**Issue:** What happens if `max_tool_iterations: 0`?

**Actual Code (`pkg/agent/loop.go`):**
```go
for iteration < al.maxIterations {
    iteration++
    // ...
}
```

**Behavior:** Loop never executes, agent returns empty response immediately.

**Recommendation:** Add minimum value validation (e.g., `>= 1`)

---

### 8. Workspace Path Expansion Edge Cases

**Location:** `workspace` field

**Edge Cases Not Handled:**
- `workspace: ""` → Empty string
- `workspace: "~"` → Just tilde (handled)
- `workspace: "~otheruser"` → Other user's home (NOT handled)
- `workspace: "$HOME/workspace"` → Environment variable (NOT expanded)

**Actual Code:**
```go
func expandHome(path string) string {
    if path == "" {
        return path  // Returns empty string!
    }
    if path[0] == '~' {
        home, _ := os.UserHomeDir()
        if len(path) > 1 && path[1] == '/' {
            return home + path[1:]
        }
        return home
    }
    return path
}
```

**Recommendation:**
- Add validation for non-empty workspace
- Add environment variable expansion
- Handle `~username` syntax or document limitation

---

### 9. Heartbeat Interval Minimum Not Enforced

**Location:** Config Plan mentions "min 5" but code doesn't enforce

**Actual Code Comment:**
```go
Interval int `json:"interval" env:"PICOCLAW_HEARTBEAT_INTERVAL"` // minutes, min 5
```

**Issue:** Comment says "min 5" but no actual validation.

**Recommendation:** Add validation:
```go
if cfg.Heartbeat.Interval < 5 {
    cfg.Heartbeat.Interval = 5
}
```

---

### 10. Proxy URL Validation Missing

**Location:** Provider configs with `proxy` field

**Issue:** Invalid proxy URLs cause silent failures.

**Actual Code:**
```go
if proxy != "" {
    proxyURL, err := url.Parse(proxy)
    if err == nil {  // Only applies proxy if parse succeeds
        client.Transport = &http.Transport{
            Proxy: http.ProxyURL(proxyURL),
        }
    }
    // No warning if parse fails!
}
```

**Recommendation:** Log warning when proxy URL is invalid.

---

## Implementation Gaps

### 11. QQ Channel in Code but Not Example Config

**Location:** `pkg/config/config.go` defines `QQConfig` but example config doesn't include it.

**Actual Code:**
```go
type ChannelsConfig struct {
    // ...
    QQ QQConfig `json:"qq"`  // Missing from example!
}
```

**Recommendation:** Add QQ channel to example config or remove from code.

---

### 12. Web Search Tool Config Structure Mismatch

**Config Plan Section:** Mentions nested `web.search` structure

**Example Config:**
```json
"tools": {
  "web": {
    "search": {
      "api_key": "YOUR_BRAVE_API_KEY"
    }
  }
}
```

**Actual Code Structure:**
```go
type ToolsConfig struct {
    Web WebToolsConfig `json:"web"`
}

type WebToolsConfig struct {
    Brave      BraveConfig      `json:"brave"`
    DuckDuckGo DuckDuckGoConfig `json:"duckduckgo"`
}
```

**Issue:** The example uses `web.search.api_key` but actual structure is `web.brave.api_key`.

**Recommendation:** Fix example config to match actual structure.

---

### 13. FlexibleStringSlice Edge Cases

**Location:** `allow_from` field handling

**Issue:** The `FlexibleStringSlice` type accepts both strings and numbers:

```go
case float64:
    result = append(result, fmt.Sprintf("%.0f", val))
```

**Edge Case:** Very large numbers lose precision (JavaScript's `Number.MAX_SAFE_INTEGER` = 2^53-1)

**Recommendation:** Document that user IDs should be strings in JSON to avoid precision loss.

---

## Missing Features

### 14. No Config Schema Validation

**Recommendation:** Add JSON schema validation for config files:
- Validate required fields
- Validate field types
- Validate value ranges
- Provide helpful error messages

---

### 15. No Config Migration Path

**Issue:** When config structure changes between versions, old configs break.

**Recommendation:** 
- Add config version field
- Implement migration logic for backward compatibility
- Log deprecation warnings for old fields

---

### 16. No Environment Variable Documentation

**Issue:** Each field has an env var (e.g., `PICOCLAW_AGENTS_DEFAULTS_MODEL`) but these aren't documented.

**Recommendation:** Generate or document all supported environment variables.

---

## Security Considerations

### 17. API Keys in Plain Text

**Issue:** API keys stored in plain text in config file.

**Recommendation:**
- Support reading from environment variables (already done)
- Consider supporting encrypted storage
- Add `.gitignore` entry for `config.json` (already in place)

---

### 18. RestrictToWorkspace Bypass

**Issue:** `restrict_to_workspace: false` allows accessing any file.

**Recommendation:**
- Log warning when workspace restriction is disabled
- Consider audit logging for file operations outside workspace

---

## Summary of Recommendations

| Priority | Issue | Action |
|----------|-------|--------|
| Critical | #1, #2 | Add provider field, validate credentials at startup |
| Critical | #3 | Fix port conflict (MaixCam vs Gateway) |
| High | #4, #11 | Update example config with all providers/channels |
| High | #7 | Add max_tool_iterations minimum validation |
| Medium | #5 | Document empty allow_from behavior |
| Medium | #6, #9 | Add temperature and interval validation |
| Medium | #8 | Improve workspace path handling |
| Medium | #10 | Add proxy URL validation with warnings |
| Low | #12 | Fix web search config structure in example |
| Low | #13 | Document string requirement for user IDs |
| Low | #14-16 | Add schema validation, migration, env var docs |
| Low | #17-18 | Security hardening |

---

## Conclusion

The PicoClaw configuration plan is functional but needs:
1. **Alignment** between documentation, example config, and actual code
2. **Validation** for critical fields (credentials, ports, ranges)
3. **Completeness** in example config (missing providers/channels)
4. **Edge case handling** for empty values and boundary conditions

Most issues are minor and can be addressed incrementally. The critical issues (#1-3) should be addressed before the next release to prevent user confusion and runtime errors.
