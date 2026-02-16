# PicoClaw ZAI MCP Integration - Technical Review V2

## Scope

This review covers **only** the proposed ZAI MCP integration changes for PicoClaw:
- ZAI MCP provider for `web_search`
- ZAI MCP provider for `web_fetch`
- Configuration for provider selection (ZAI vs Brave vs DuckDuckGo)

---

## Critical Issues

### 1. MCP Client Implementation is Incomplete

**Location**: `pkg/mcp/client.go`

**Problem**: The `CallTool` method is just a stub:

```go
func (c *MCPClient) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
	// Implementation using mcp.ClientSession
	// Similar pattern to Python implementation
}
```

**Issues**:
- No actual MCP protocol implementation
- Missing session initialization (MCP requires `initialize` handshake)
- Missing capability negotiation
- Go MCP SDK uses different patterns than Python SDK

**What's missing**:
```go
// Actual MCP session flow:
// 1. Connect to HTTP endpoint
// 2. Send initialize request with client capabilities
// 3. Receive server capabilities  
// 4. Send initialized notification
// 5. THEN call tools
```

---

### 2. Wrong Response Parsing Assumptions

**Location**: `pkg/tools/zai_search.go`

**Problem**: Code assumes ZAI MCP returns raw array:

```go
results, ok := result.([]interface{})
```

**Reality**: MCP protocol wraps tool responses in structured format:
```json
{
  "content": [
    {"type": "text", "text": "...actual JSON data here..."}
  ]
}
```

The actual search results are **nested inside** `content[0].text` as a JSON string that needs to be parsed again.

---

### 3. Missing Go SDK Transport Implementation

**Problem**: Plan mentions `streamable_http_client` (Python concept) but Go SDK handles HTTP differently.

Go MCP SDK transport options:
- `StdioTransport` - for local process communication
- `InProcTransport` - for in-process servers
- HTTP transport requires custom implementation via `jsonrpc` package

The plan provides no implementation for HTTP MCP transport in Go.

---

### 4. Provider Priority is Hardcoded

**Location**: `pkg/tools/web.go` - `NewWebSearchTool`

**Problem**: Priority order is hardcoded:
```go
// Priority: ZAI > Brave > DuckDuckGo
if opts.ZaiEnabled && opts.ZaiAPIKey != "" {
    // ...
} else if opts.BraveEnabled && opts.BraveAPIKey != "" {
    // ...
```

**Issues**:
- No way to configure preferred provider
- No fallback chain configuration
- If ZAI is enabled but fails, no automatic fallback

**Better approach**: Configurable priority list or explicit provider selection.

---

### 5. Incomplete FetchProvider Pattern

**Problem**: `FetchProvider` interface introduced but existing native implementation not refactored:

```go
provider = &NativeFetchProvider{} // Existing implementation
```

This type doesn't exist in the plan. The existing `WebFetchTool` implementation needs to be wrapped or rewritten to implement `FetchProvider`.

---

## Moderate Issues

### 6. Missing Import in Code Snippet

**Location**: `pkg/tools/zai_search.go`

```go
return strings.Join(lines, "\n"), nil
```

`strings` package is used but not imported.

---

### 7. No Retry Logic Despite Claiming It

**Checklist item**: "Add error handling and retry logic"

**Reality**: No retry implementation shown. MCP over HTTP can fail transiently - retry is important.

---

### 8. Timeout Handling Gap

**Problem**: `ZaiConfig.Timeout` is per-request, but MCP session initialization adds overhead.

- First call: session init + tool call = potentially > timeout
- Subsequent calls: just tool call

Need connection/session-level timeout separate from request timeout.

---

### 9. API Key Fallback Not Implemented

**Config shows**:
```go
Env Variable: `Z_AI_API_KEY`
Alternative (fallback): `Z_AI_API_KEY` (same as nanobot uses)
```

**Code shows**:
```go
ZaiAPIKey    string
```

No fallback logic from `Z_AI_API_KEY` env var when `ZaiAPIKey` config is empty.

---

### 10. Missing MCP Session Reuse

**Problem**: Each tool call seems to create new MCP client/session.

MCP session establishment has overhead:
1. HTTP connection
2. Initialize handshake
3. Capability exchange

Should reuse session across multiple calls.

---

## Minor Issues

### 11. Response Format Inconsistency

**Location**: `pkg/tools/zai_fetch.go`

```go
response := map[string]interface{}{
    "url":       url,
    "status":    200,
    "text":      result,
    "extractor": "zai-mcp-http",
}
```

The `status: 200` is hardcoded. ZAI MCP could fail - need actual status from response.

---

### 12. Max Results Not Passed to ZAI

**Location**: `pkg/tools/zai_search.go`

```go
result, err := p.client.CallTool(ctx, "webSearchPrime", map[string]interface{}{
    "search_query": query,
})
```

The `count` parameter exists in MCP tool but isn't passed:
```go
// Should be:
result, err := p.client.CallTool(ctx, "webSearchPrime", map[string]interface{}{
    "search_query": query,
    "count": count,
})
```

Instead, results are truncated client-side after fetching.

---

## Recommendations Summary

| Priority | Issue | Action Required |
|----------|-------|-----------------|
| **P0** | MCP client stub | Implement actual MCP protocol with session management |
| **P0** | Response parsing | Handle MCP content wrapper structure |
| **P0** | HTTP transport | Implement Go MCP HTTP transport |
| **P1** | Session reuse | Pool/reuse MCP sessions |
| **P1** | Provider fallback | Implement automatic fallback on failure |
| **P2** | Configurable priority | Allow user to set provider preference |
| **P2** | API key fallback | Check `Z_AI_API_KEY` env var |

---

## Conclusion

The plan identifies the right components but the implementation details have significant gaps:

1. **MCP protocol not actually implemented** - just placeholder stubs
2. **Response format mismatch** - doesn't account for MCP envelope structure
3. **No HTTP transport** - critical for communicating with ZAI MCP endpoints
4. **Session management missing** - will be inefficient without reuse

The core issue is that the plan shows high-level structure but lacks working MCP client code. The Go MCP SDK is referenced but actual usage patterns for HTTP-based MCP servers are not demonstrated.
