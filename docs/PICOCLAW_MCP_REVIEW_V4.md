# PicoClaw MCP KISS Review V4

**Review Date**: 2026-02-16  
**Document Reviewed**: PICOCLAW_MCP_KISS.md (V4)  
**Reviewer**: Subagent Task

---

## Executive Summary

The V4 KISS plan is **technically correct** and aligns with the official MCP Go SDK API. All issues from V3 review have been addressed. The implementation is ready to proceed.

---

## ✅ SDK API Verification

### 1. Client Creation - CORRECT ✓

**V4 Plan:**
```go
client := mcp.NewClient(&mcp.Implementation{
    Name:    "picoclaw",
    Version: "v1.0.0",
}, nil)
```

**Official SDK (from GitHub README):**
```go
client := mcp.NewClient(&mcp.Implementation{Name: "mcp-client", Version: "v1.0.0"}, nil)
```

**Status**: ✅ CORRECT - Matches official API

---

### 2. Session Connection - CORRECT ✓

**V4 Plan:**
```go
session, err := client.Connect(ctx, transport, nil)
if err != nil {
    return nil, fmt.Errorf("failed to connect to MCP server: %w", err)
}
```

**Official SDK (from GitHub README):**
```go
session, err := client.Connect(ctx, transport, nil)
if err != nil {
    log.Fatal(err)
}
defer session.Close()
```

**Status**: ✅ CORRECT - Matches official API

---

### 3. HTTP Transport - CORRECT ✓

**V4 Plan:**
```go
transport := &mcp.StreamableClientTransport{
    Endpoint: endpoint,
    HTTPClient: &http.Client{
        Timeout: 60 * time.Second,
        Transport: &authTransport{
            apiKey:     apiKey,
            underlying: http.DefaultTransport,
        },
    },
}
```

**Official SDK (from examples/http/main.go):**
```go
session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: url}, nil)
```

**Status**: ✅ CORRECT - Uses SDK's `StreamableClientTransport` with custom HTTPClient for auth

---

### 4. Tool Calling - CORRECT ✓

**V4 Plan:**
```go
result, err := c.session.CallTool(ctx, &mcp.CallToolParams{
    Name:      name,
    Arguments: args,
})
```

**Official SDK (from GitHub README):**
```go
params := &mcp.CallToolParams{
    Name:      "greet",
    Arguments: map[string]any{"name": "you"},
}
res, err := session.CallTool(ctx, params)
```

**Status**: ✅ CORRECT - Matches official API

---

### 5. Content Parsing - CORRECT ✓

**V4 Plan:**
```go
for _, content := range result.Content {
    if tc, ok := content.(*mcp.TextContent); ok {
        text += tc.Text
    }
}
```

**Official SDK (from GitHub README):**
```go
for _, c := range res.Content {
    log.Print(c.(*mcp.TextContent).Text)
}
```

**Status**: ✅ CORRECT - Uses type assertion with ok check (safer than direct cast)

---

## ✅ HTTP Transport Implementation

### Auth Header Handling

The V4 plan correctly implements custom transport for API key injection:

```go
type authTransport struct {
    apiKey     string
    underlying http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    req.Header.Set("Authorization", "Bearer "+t.apiKey)
    return t.underlying.RoundTrip(req)
}
```

**Status**: ✅ CORRECT - Standard Go pattern for adding auth headers

---

## ✅ ZAI Provider Integration

### Tool Names

| Tool | Endpoint | MCP Tool Name | Status |
|------|----------|---------------|--------|
| Search | `https://api.z.ai/api/mcp/web_search_prime/mcp` | `webSearchPrime` | ✅ |
| Fetch | `https://api.z.ai/api/mcp/web_reader/mcp` | `webReader` | ✅ |

### Tool Parameters

**webSearchPrime:**
```go
map[string]any{
    "search_query": query,
    "count":        count,
}
```

**webReader:**
```go
map[string]any{
    "url":           url,
    "return_format": mode,
}
```

**Status**: ✅ CORRECT - Matches ZAI MCP specification

---

## ✅ Config Verification

### API Key Source

**V4 Plan:**
```
Auth: Via `providers.zhipu.api_key` from config (ZAI uses Zhipu credentials)
```

**Existing Config (pkg/config/config.go):**
```go
type ProvidersConfig struct {
    Zhipu ProviderConfig `json:"zhipu"`
    // ... other providers
}

type ProviderConfig struct {
    APIKey string `json:"api_key" env:"PICOCLAW_PROVIDERS_ZHIPU_API_KEY"`
    // ...
}
```

**Status**: ✅ CORRECT - Uses existing `providers.zhipu.api_key`, no new env vars

### ZAIConfig Addition

**V4 Plan:**
```go
type ZAIConfig struct {
    Enabled    bool   `json:"enabled" env:"PICOCLAW_TOOLS_WEB_ZAI_ENABLED"`
    MaxResults int    `json:"max_results" env:"PICOCLAW_TOOLS_WEB_ZAI_MAX_RESULTS"`
    Timeout    int    `json:"timeout" env:"PICOCLAW_TOOLS_WEB_ZAI_TIMEOUT"`
}
```

**Existing WebToolsConfig:**
```go
type WebToolsConfig struct {
    Brave      BraveConfig      `json:"brave"`
    DuckDuckGo DuckDuckGoConfig `json:"duckduckgo"`
}
```

**Status**: ✅ COMPATIBLE - Adds ZAI config without breaking existing structure

---

## ✅ web.go Integration

### Existing WebSearchToolOptions

```go
type WebSearchToolOptions struct {
    BraveAPIKey          string
    BraveMaxResults      int
    BraveEnabled         bool
    DuckDuckGoMaxResults int
    DuckDuckGoEnabled    bool
}
```

### V4 Extended Options

```go
type WebSearchToolOptions struct {
    BraveAPIKey          string
    BraveMaxResults      int
    BraveEnabled         bool
    DuckDuckGoMaxResults int
    DuckDuckGoEnabled    bool
    ZAIMaxResults        int
    ZAIEnabled           bool
    ZAIClient            *mcp.Client  // Pre-initialized MCP client
}
```

**Status**: ✅ COMPATIBLE - Additive change, follows existing pattern

### Provider Priority

```
Priority: ZAI > Brave > DuckDuckGo
```

**Status**: ✅ SENSIBLE - ZAI has highest priority when enabled

---

## ✅ KISS Compliance

| Criterion | Status | Notes |
|-----------|--------|-------|
| Minimal files | ✅ PASS | Only 2 new files: `pkg/mcp/client.go`, `pkg/tools/zai.go` |
| Official SDK | ✅ PASS | Uses `github.com/modelcontextprotocol/go-sdk` |
| No over-engineering | ✅ PASS | No abstraction layers, direct SDK usage |
| Config reuse | ✅ PASS | Uses `providers.zhipu.api_key`, no new env vars |
| Backward compatible | ✅ PASS | Additive changes to existing code |

---

## ❌ Issues Found

### Issue 1: Missing Error Check for result.IsError

**Location:** `pkg/mcp/client.go` - `CallTool()`

**Current:**
```go
result, err := c.session.CallTool(ctx, &mcp.CallToolParams{...})
if err != nil {
    return "", fmt.Errorf("tool call failed: %w", err)
}
// Missing: check result.IsError
```

**Recommendation:**
```go
result, err := c.session.CallTool(ctx, &mcp.CallToolParams{...})
if err != nil {
    return "", fmt.Errorf("tool call failed: %w", err)
}
if result.IsError {
    return "", fmt.Errorf("tool returned error: %s", text)
}
```

**Severity**: MINOR - Can be added during implementation

---

### Issue 2: Missing pkg/mcp Directory

**Current State:** The `pkg/mcp` directory does not exist yet.

**Action Required:** Create directory before implementing `client.go`

**Severity**: N/A - Implementation task

---

## Summary Table

| Component | Status | Notes |
|-----------|--------|-------|
| `mcp.NewClient()` | ✅ CORRECT | Matches SDK API |
| `client.Connect()` | ✅ CORRECT | Matches SDK API |
| `StreamableClientTransport` | ✅ CORRECT | SDK's built-in transport |
| `CallToolParams` struct | ✅ CORRECT | Correct params structure |
| Content type assertion | ✅ CORRECT | Safe type assertion with ok check |
| Auth transport | ✅ CORRECT | Standard Go pattern |
| Config integration | ✅ COMPATIBLE | Uses existing zhipu.api_key |
| web.go integration | ✅ COMPATIBLE | Additive changes |
| ZAI tool names | ✅ CORRECT | webSearchPrime, webReader |
| KISS compliance | ✅ PASS | Minimal, clean design |

---

## Verdict

| Aspect | Rating |
|--------|--------|
| KISS Philosophy | ⭐⭐⭐⭐⭐ Excellent |
| SDK API Accuracy | ⭐⭐⭐⭐⭐ Perfect (V3 issues fixed) |
| Implementation Readiness | ⭐⭐⭐⭐⭐ Ready |

**Status**: ✅ **APPROVED** - Ready for implementation

---

## Implementation Checklist

- [ ] `go get github.com/modelcontextprotocol/go-sdk@latest`
- [ ] Create `pkg/mcp/` directory
- [ ] Create `pkg/mcp/client.go` (copy from plan, add IsError check)
- [ ] Create `pkg/tools/zai.go` (copy from plan)
- [ ] Add `ZAIConfig` to `pkg/config/config.go`
- [ ] Update `WebToolsConfig` to include ZAI
- [ ] Update `NewWebSearchTool()` in `pkg/tools/web.go`
- [ ] Update default config in `DefaultConfig()`
- [ ] Pass `providers.zhipu.api_key` to MCP client at initialization
- [ ] Test with ZAI endpoints

---

## Files Summary

| File | Action | Lines Changed |
|------|--------|---------------|
| `pkg/mcp/client.go` | CREATE | ~60 lines |
| `pkg/tools/zai.go` | CREATE | ~40 lines |
| `pkg/config/config.go` | MODIFY | +10 lines |
| `pkg/tools/web.go` | MODIFY | +20 lines |

**Total**: ~130 new/modified lines - KISS compliant

---

## Conclusion

The V4 MCP KISS plan is technically accurate and ready for implementation. All SDK API calls match the official `github.com/modelcontextprotocol/go-sdk` documentation. The integration with existing PicoClaw code is minimal and backward-compatible.
