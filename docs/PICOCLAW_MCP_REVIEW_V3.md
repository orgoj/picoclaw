# PicoClaw MCP KISS Review V3

**Review Date**: 2026-02-16  
**Document Reviewed**: PICOCLAW_MCP_KISS.md

## Summary

The KISS plan is **good in spirit** but contains **several API mismatches** with the official Go SDK that need correction before implementation.

---

## ✅ KISS Compliance Check

| Criterion | Status | Notes |
|-----------|--------|-------|
| Only ZAI MCP provider | ⚠️ PARTIAL | Config shows `zai\|brave\|duckduckgo` options |
| No over-engineering | ✅ PASS | Minimal structure, 2 files |
| Functional MCP client | ❌ FAIL | API mismatch with official SDK |
| Official SDK usage | ✅ PASS | Correct SDK: `github.com/modelcontextprotocol/go-sdk` |

---

## ❌ Critical Issues Found

### 1. MCP Client API Mismatch (BLOCKER)

**Plan shows:**
```go
func NewClient(endpoint, apiKey string) (*Client, error) {
    t := NewHTTPTransport(endpoint, apiKey)
    s := mcp.NewClientSession(t)  // ❌ WRONG API
    s.Initialize(context.Background())
    return &Client{session: s}, nil
}
```

**Actual SDK API:**
```go
client := mcp.NewClient(&mcp.Implementation{Name: "picoclaw", Version: "v1.0.0"}, nil)
session, err := client.Connect(ctx, transport, nil)
```

**Fix Required:**
- Use `mcp.NewClient()` to create client
- Use `client.Connect(ctx, transport, opts)` to get session
- No separate `Initialize()` call needed - `Connect()` handles it

---

### 2. CallTool API Mismatch

**Plan shows:**
```go
func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (string, error) {
    r, _ := c.session.CallTool(ctx, name, args)  // ❌ WRONG SIGNATURE
    var text string
    for _, c := range r.Content {
        if c.Type == "text" { text += c.Text }
    }
    return text, nil
}
```

**Actual SDK API:**
```go
result, err := session.CallTool(ctx, &mcp.CallToolParams{
    Name:      "webSearchPrime",
    Arguments: map[string]any{"search_query": query, "count": count},
})
// result.Content is []mcp.Content (interface types)
for _, c := range result.Content {
    if tc, ok := c.(*mcp.TextContent); ok {
        text += tc.Text
    }
}
```

**Fix Required:**
- Use `&mcp.CallToolParams{Name: ..., Arguments: ...}`
- Content is `[]mcp.Content` interface - use type assertion

---

### 3. Custom HTTP Transport Not Needed

**Plan shows custom Transport:**
```go
func (t *Transport) Send(ctx context.Context, req *Request) (*Response, error)
```

**SDK already provides:**
- `mcp.StreamableClientTransport` - HTTP transport (2025-03-26 spec)
- `mcp.SSEClientTransport` - SSE transport (2024-11-05 spec)

**Recommendation:**
Use `StreamableClientTransport` for ZAI endpoints:
```go
transport := &mcp.StreamableClientTransport{
    Endpoint:   "https://api.z.ai/api/mcp/web_search_prime/mcp",
    HTTPClient: &http.Client{Timeout: 60 * time.Second},
}
```

Note: May need to verify ZAI MCP endpoint compatibility with SDK transports.

---

### 4. Config Inconsistency

**Plan shows:**
```yaml
mcp:
  provider: zai  # zai|brave|duckduckgo  ← Multiple providers listed
```

**KISS requirement:** Only ZAI MCP provider

**Fix:** Remove alternative provider options:
```yaml
mcp:
  zai:
    api_key: ${Z_AI_API_KEY}
    timeout: 60
```

---

## ✅ Correct Aspects

1. **SDK choice**: `github.com/modelcontextprotocol/go-sdk` is the official SDK ✓
2. **ZAI endpoints**: Correct tool names `webSearchPrime` and `webReader` ✓
3. **File structure**: Minimal 2-file approach is KISS-compliant ✓
4. **Auth pattern**: `Z_AI_API_KEY` env var is standard ✓

---

## Corrected Implementation

### pkg/mcp/client.go
```go
package mcp

import (
    "context"
    "net/http"
    "time"

    mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type Client struct {
    session *mcp.ClientSession
}

func NewClient(ctx context.Context, endpoint, apiKey string) (*Client, error) {
    client := mcp.NewClient(&mcp.Implementation{
        Name:    "picoclaw",
        Version: "v1.0.0",
    }, nil)

    // Note: API key handling depends on ZAI's transport requirements
    transport := &mcp.StreamableClientTransport{
        Endpoint: endpoint,
        HTTPClient: &http.Client{
            Timeout: 60 * time.Second,
            // Add auth header via transport wrapper if needed
        },
    }

    session, err := client.Connect(ctx, transport, nil)
    if err != nil {
        return nil, err
    }

    return &Client{session: session}, nil
}

func (c *Client) Close() error {
    return c.session.Close()
}

func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (string, error) {
    result, err := c.session.CallTool(ctx, &mcp.CallToolParams{
        Name:      name,
        Arguments: args,
    })
    if err != nil {
        return "", err
    }

    var text string
    for _, content := range result.Content {
        if tc, ok := content.(*mcp.TextContent); ok {
            text += tc.Text
        }
    }
    return text, nil
}
```

### pkg/tools/zai.go
```go
package tools

import (
    "context"
)

func Search(ctx context.Context, client *mcp.Client, query string, count int) (string, error) {
    return client.CallTool(ctx, "webSearchPrime", map[string]any{
        "search_query": query,
        "count":        count,
    })
}

func Fetch(ctx context.Context, client *mcp.Client, url, mode string) (string, error) {
    return client.CallTool(ctx, "webReader", map[string]any{
        "url":           url,
        "return_format": mode,
    })
}
```

---

## Checklist Update

Original checklist needs revision:

- [x] `go get github.com/modelcontextprotocol/go-sdk` (correct)
- [ ] ~~MCP client + HTTP transport~~ → Use SDK's `StreamableClientTransport`
- [x] Parse `content[].Text` (with type assertion)
- [x] ZAI tools (correct tool names)
- [ ] Provider config (remove alternatives)
- [ ] Tests

---

## Verdict

| Aspect | Rating |
|--------|--------|
| KISS Philosophy | ⭐⭐⭐⭐⭐ Excellent |
| SDK API Accuracy | ⭐⭐ Needs fixes |
| Implementation Readiness | ⭐⭐⭐ Close, needs SDK alignment |

**Status**: ⚠️ **Needs Revision** - Fix SDK API calls before implementation

---

## Action Items

1. [ ] Update `NewClient()` to use `mcp.NewClient()` + `Connect()`
2. [ ] Update `CallTool()` to use `&mcp.CallToolParams{}`
3. [ ] Test ZAI endpoint compatibility with `StreamableClientTransport`
4. [ ] Remove alternative providers from config
5. [ ] Add auth header handling for ZAI API key
