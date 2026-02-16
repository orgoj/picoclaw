# PicoClaw MCP KISS Plan

ZAI MCP integration using official Go SDK.

## Stack
- **SDK**: `github.com/modelcontextprotocol/go-sdk`
- **Search**: `api.z.ai/api/mcp/web_search_prime/mcp` → `webSearchPrime`
- **Fetch**: `api.z.ai/api/mcp/web_reader/mcp` → `webReader`
- **Auth**: Via `providers.zhipu.api_key` from config (ZAI uses Zhipu credentials)

## Files
```
pkg/mcp/client.go   # MCP client using official SDK
pkg/tools/zai.go    # Search + Fetch tools
```

## MCP Client (pkg/mcp/client.go)

```go
package mcp

import (
    "context"
    "fmt"
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

    // Use SDK's StreamableClientTransport with auth header
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

    session, err := client.Connect(ctx, transport, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to MCP server: %w", err)
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
        return "", fmt.Errorf("tool call failed: %w", err)
    }

    var text string
    for _, content := range result.Content {
        if tc, ok := content.(*mcp.TextContent); ok {
            text += tc.Text
        }
    }
    return text, nil
}

// authTransport adds Authorization header to requests
type authTransport struct {
    apiKey     string
    underlying http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    req.Header.Set("Authorization", "Bearer "+t.apiKey)
    return t.underlying.RoundTrip(req)
}
```

## ZAI Tools (pkg/tools/zai.go)

```go
package tools

import (
    "context"
    "fmt"
    
    "github.com/sipeed/picoclaw/pkg/mcp"
)

type ZAISearchProvider struct {
    client *mcp.Client
}

func NewZAISearchProvider(client *mcp.Client) *ZAISearchProvider {
    return &ZAISearchProvider{client: client}
}

func (p *ZAISearchProvider) Search(ctx context.Context, query string, count int) (string, error) {
    return p.client.CallTool(ctx, "webSearchPrime", map[string]any{
        "search_query": query,
        "count":        count,
    })
}

type ZAIFetchProvider struct {
    client *mcp.Client
}

func NewZAIFetchProvider(client *mcp.Client) *ZAIFetchProvider {
    return &ZAIFetchProvider{client: client}
}

func (p *ZAIFetchProvider) Fetch(ctx context.Context, url, mode string) (string, error) {
    return p.client.CallTool(ctx, "webReader", map[string]any{
        "url":           url,
        "return_format": mode,
    })
}
```

## Config Changes

Add ZAI provider to `tools.web` section in `pkg/config/config.go`:

```go
type ZAIConfig struct {
    Enabled    bool   `json:"enabled" env:"PICOCLAW_TOOLS_WEB_ZAI_ENABLED"`
    MaxResults int    `json:"max_results" env:"PICOCLAW_TOOLS_WEB_ZAI_MAX_RESULTS"`
    Timeout    int    `json:"timeout" env:"PICOCLAW_TOOLS_WEB_ZAI_TIMEOUT"`
}

type WebToolsConfig struct {
    Brave      BraveConfig      `json:"brave"`
    DuckDuckGo DuckDuckGoConfig `json:"duckduckgo"`
    ZAI        ZAIConfig        `json:"zai"`
}
```

**Note**: ZAI uses `providers.zhipu.api_key` for authentication. No separate API key env var needed.

Default config:
```go
ZAI: ZAIConfig{
    Enabled:    false,
    MaxResults: 5,
    Timeout:    60,
},
```

## Web Tool Integration (pkg/tools/web.go)

Update `NewWebSearchTool` to support ZAI provider:

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

func NewWebSearchTool(opts WebSearchToolOptions) *WebSearchTool {
    var provider SearchProvider
    maxResults := 5

    // Priority: ZAI > Brave > DuckDuckGo
    if opts.ZAIEnabled && opts.ZAIClient != nil {
        provider = NewZAISearchProvider(opts.ZAIClient)
        if opts.ZAIMaxResults > 0 {
            maxResults = opts.ZAIMaxResults
        }
    } else if opts.BraveEnabled && opts.BraveAPIKey != "" {
        provider = &BraveSearchProvider{apiKey: opts.BraveAPIKey}
        if opts.BraveMaxResults > 0 {
            maxResults = opts.BraveMaxResults
        }
    } else if opts.DuckDuckGoEnabled {
        provider = &DuckDuckGoSearchProvider{}
        if opts.DuckDuckGoMaxResults > 0 {
            maxResults = opts.DuckDuckGoMaxResults
        }
    } else {
        return nil
    }

    return &WebSearchTool{
        provider:   provider,
        maxResults: maxResults,
    }
}
```

## Example Config (config.json)

```json
{
  "providers": {
    "zhipu": {
      "api_key": "your-zhipu-api-key"
    }
  },
  "tools": {
    "web": {
      "brave": {
        "enabled": false,
        "api_key": "",
        "max_results": 5
      },
      "duckduckgo": {
        "enabled": false,
        "max_results": 5
      },
      "zai": {
        "enabled": true,
        "max_results": 10,
        "timeout": 60
      }
    }
  }
}
```

## MCP Endpoints

| Tool | Endpoint | MCP Tool Name |
|------|----------|---------------|
| Search | `https://api.z.ai/api/mcp/web_search_prime/mcp` | `webSearchPrime` |
| Fetch | `https://api.z.ai/api/mcp/web_reader/mcp` | `webReader` |

## Checklist

- [ ] `go get github.com/modelcontextprotocol/go-sdk`
- [ ] Create `pkg/mcp/client.go` with correct SDK API:
  - [ ] Use `mcp.NewClient()` to create client
  - [ ] Use `client.Connect(ctx, transport, opts)` to get session
  - [ ] Use `&mcp.CallToolParams{Name: ..., Arguments: ...}` for tool calls
  - [ ] Use type assertion `content.(*mcp.TextContent)` for content parsing
  - [ ] Use `mcp.StreamableClientTransport` for HTTP transport
- [ ] Create `pkg/tools/zai.go` with ZAI providers
- [ ] Add `ZAIConfig` to `pkg/config/config.go`
- [ ] Update `NewWebSearchTool` in `pkg/tools/web.go`
- [ ] Use `providers.zhipu.api_key` for ZAI authentication
- [ ] Tests

## SDK API Reference

### Client Creation
```go
// Create client
client := mcp.NewClient(&mcp.Implementation{
    Name:    "picoclaw",
    Version: "v1.0.0",
}, nil)

// Connect to get session
session, err := client.Connect(ctx, transport, nil)
```

### Tool Calling
```go
// Call tool with params struct
result, err := session.CallTool(ctx, &mcp.CallToolParams{
    Name:      "webSearchPrime",
    Arguments: map[string]any{"search_query": query, "count": count},
})

// Parse content with type assertion
for _, content := range result.Content {
    if tc, ok := content.(*mcp.TextContent); ok {
        text += tc.Text
    }
}
```

### Transport
```go
// Use SDK's built-in transport
transport := &mcp.StreamableClientTransport{
    Endpoint:   "https://api.z.ai/api/mcp/web_search_prime/mcp",
    HTTPClient: &http.Client{Timeout: 60 * time.Second},
}
```
