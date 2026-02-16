# PicoClaw ZAI MCP Integration Plan

## Overview

This document proposes integrating ZAI MCP (Model Context Protocol) providers for `web_search` and `web_fetch` tools in PicoClaw, allowing users to choose between Brave Search, DuckDuckGo, and ZAI MCP providers.

---

## 1. Go MCP Library Selection

### Recommended: Official Go MCP SDK

**Repository**: [github.com/modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk)

**Validation Criteria**:

| Criterion | Status | Details |
|-----------|--------|---------|
| **Active** | ✅ | Official SDK maintained by MCP org in collaboration with Google |
| **Well-maintained** | ✅ | Regular commits, active development |
| **Documentation** | ✅ | Comprehensive docs at [pkg.go.dev](https://pkg.go.dev/github.com/modelcontextprotocol/go-sdk/mcp) |
| **License** | ✅ | Apache 2.0 / MIT dual license |
| **Version** | ✅ | v1.2.0+ supports MCP spec 2025-06-18 |

### SDK Structure

```
github.com/modelcontextprotocol/go-sdk/
├── mcp/        # Primary API for clients and servers
├── jsonrpc/    # Custom transport implementation
├── auth/       # OAuth primitives
└── oauthex/    # OAuth extensions
```

### Key Features

- Full MCP spec implementation
- `StdioTransport` for local process communication
- `CommandTransport` for spawning server processes
- HTTP transport support via `jsonrpc` package
- Type-safe tool registration with JSON schema

### Alternative Libraries (Not Recommended)

| Library | Reason |
|---------|--------|
| `mcp-go` (Ed Zynda) | Third-party, official SDK preferred |
| `go-mcp` | Less mature, smaller community |
| `mcp-golang` | Less active maintenance |

---

## 2. Nanobot ZAI MCP Usage Analysis

### Reference Implementation

**File**: `nanobot-repo/nanobot/agent/tools/zai_web.py`

### ZAI MCP Endpoints

| Tool | Endpoint | MCP Tool Name |
|------|----------|---------------|
| Web Search | `https://api.z.ai/api/mcp/web_search_prime/mcp` | `webSearchPrime` |
| Web Fetch | `https://api.z.ai/api/mcp/web_reader/mcp` | `webReader` |

### Authentication

- **Method**: Bearer token in HTTP headers
- **Env Variable**: `Z_AI_API_KEY`
- **Header**: `Authorization: Bearer <api_key>`

### MCP Protocol Flow

```
1. Create HTTP client with auth headers
2. Connect to MCP endpoint via streamable_http_client
3. Initialize ClientSession
4. Call tool with session.call_tool(tool_name, arguments)
5. Extract text from response content
6. Parse JSON response
```

### Tool Parameters

#### web_search (webSearchPrime)
```json
{
  "search_query": "string",
  "count": "integer (optional, default 10)"
}
```

#### web_fetch (webReader)
```json
{
  "url": "string",
  "return_format": "string (markdown|text)"
}
```

---

## 3. Implementation Proposal

### 3.1 Go MCP Client Package

**New File**: `pkg/mcp/client.go`

```go
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPClient handles communication with MCP servers via HTTP
type MCPClient struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
}

// NewMCPClient creates a new MCP client
func NewMCPClient(baseURL, apiKey string, timeout time.Duration) *MCPClient {
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	return &MCPClient{
		httpClient: &http.Client{Timeout: timeout},
		apiKey:     apiKey,
		baseURL:    baseURL,
	}
}

// CallTool invokes a tool on the MCP server
func (c *MCPClient) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
	// Implementation using mcp.ClientSession
	// Similar pattern to Python implementation
}
```

### 3.2 ZAI Search Provider

**New File**: `pkg/tools/zai_search.go`

```go
package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

const (
	ZaiSearchURL = "https://api.z.ai/api/mcp/web_search_prime/mcp"
	ZaiReaderURL = "https://api.z.ai/api/mcp/web_reader/mcp"
)

// ZaiSearchProvider implements SearchProvider using ZAI MCP
type ZaiSearchProvider struct {
	client *mcp.MCPClient
}

// NewZaiSearchProvider creates a new ZAI search provider
func NewZaiSearchProvider(apiKey string, timeout time.Duration) *ZaiSearchProvider {
	client := mcp.NewMCPClient(ZaiSearchURL, apiKey, timeout)
	return &ZaiSearchProvider{client: client}
}

// Search performs web search via ZAI MCP
func (p *ZaiSearchProvider) Search(ctx context.Context, query string, count int) (string, error) {
	result, err := p.client.CallTool(ctx, "webSearchPrime", map[string]interface{}{
		"search_query": query,
	})
	if err != nil {
		return "", fmt.Errorf("ZAI MCP search failed: %w", err)
	}

	// Parse results - ZAI returns array of search results
	results, ok := result.([]interface{})
	if !ok {
		return fmt.Sprintf("Results for: %s\nUnexpected response format", query), nil
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Results for: %s (via ZAI MCP)", query))

	for i, item := range results {
		if i >= count {
			break
		}
		if m, ok := item.(map[string]interface{}); ok {
			title := getString(m, "title", "No Title")
			link := getString(m, "link", getString(m, "url", ""))
			content := getString(m, "content", getString(m, "summary", ""))
			
			lines = append(lines, fmt.Sprintf("%d. %s\n   %s\n   %s", i+1, title, link, content))
		}
	}

	return strings.Join(lines, "\n"), nil
}

func getString(m map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if v, ok := m[key].(string); ok && v != "" {
			return v
		}
	}
	return ""
}
```

### 3.3 ZAI Fetch Provider

**New File**: `pkg/tools/zai_fetch.go`

```go
package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

// ZaiFetchProvider implements web content fetching via ZAI MCP
type ZaiFetchProvider struct {
	client *mcp.MCPClient
}

// NewZaiFetchProvider creates a new ZAI fetch provider
func NewZaiFetchProvider(apiKey string, timeout time.Duration) *ZaiFetchProvider {
	client := mcp.NewMCPClient(ZaiReaderURL, apiKey, timeout)
	return &ZaiFetchProvider{client: client}
}

// Fetch retrieves and extracts content from a URL via ZAI MCP
func (p *ZaiFetchProvider) Fetch(ctx context.Context, url string, extractMode string) (string, error) {
	if extractMode == "" {
		extractMode = "markdown"
	}

	result, err := p.client.CallTool(ctx, "webReader", map[string]interface{}{
		"url":           url,
		"return_format": extractMode,
	})
	if err != nil {
		return "", fmt.Errorf("ZAI MCP fetch failed: %w", err)
	}

	response := map[string]interface{}{
		"url":       url,
		"status":    200,
		"text":      result,
		"extractor": "zai-mcp-http",
	}

	jsonBytes, _ := json.MarshalIndent(response, "", "  ")
	return string(jsonBytes), nil
}
```

### 3.4 Updated Web Tools Configuration

**File**: `pkg/config/config.go`

Add new ZAI configuration structs:

```go
// Add to ToolsConfig
type ZaiConfig struct {
	Enabled    bool   `json:"enabled" env:"PICOCLAW_TOOLS_WEB_ZAI_ENABLED"`
	APIKey     string `json:"api_key" env:"PICOCLAW_TOOLS_WEB_ZAI_API_KEY"`
	MaxResults int    `json:"max_results" env:"PICOCLAW_TOOLS_WEB_ZAI_MAX_RESULTS"`
	Timeout    int    `json:"timeout" env:"PICOCLAW_TOOLS_WEB_ZAI_TIMEOUT"` // seconds
}

type WebToolsConfig struct {
	Brave      BraveConfig      `json:"brave"`
	DuckDuckGo DuckDuckGoConfig `json:"duckduckgo"`
	Zai        ZaiConfig        `json:"zai"`  // NEW
}
```

Update default configuration:

```go
func DefaultConfig() *Config {
	return &Config{
		// ... existing config ...
		Tools: ToolsConfig{
			Web: WebToolsConfig{
				Brave: BraveConfig{
					Enabled:    false,
					APIKey:     "",
					MaxResults: 5,
				},
				DuckDuckGo: DuckDuckGoConfig{
					Enabled:    true,
					MaxResults: 5,
				},
				Zai: ZaiConfig{  // NEW
					Enabled:    false,
					APIKey:     "",
					MaxResults: 10,
					Timeout:    60,
				},
			},
		},
	}
}
```

### 3.5 Updated WebSearchTool Factory

**File**: `pkg/tools/web.go`

```go
// Updated NewWebSearchTool with ZAI support
func NewWebSearchTool(opts WebSearchToolOptions) *WebSearchTool {
	var provider SearchProvider
	maxResults := 5

	// Priority: ZAI > Brave > DuckDuckGo
	if opts.ZaiEnabled && opts.ZaiAPIKey != "" {
		provider = NewZaiSearchProvider(opts.ZaiAPIKey, time.Duration(opts.ZaiTimeout)*time.Second)
		if opts.ZaiMaxResults > 0 {
			maxResults = opts.ZaiMaxResults
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

// Updated options struct
type WebSearchToolOptions struct {
	// Brave
	BraveEnabled         bool
	BraveAPIKey          string
	BraveMaxResults      int
	
	// DuckDuckGo
	DuckDuckGoEnabled    bool
	DuckDuckGoMaxResults int
	
	// ZAI MCP (NEW)
	ZaiEnabled           bool
	ZaiAPIKey            string
	ZaiMaxResults        int
	ZaiTimeout           int
}
```

### 3.6 Updated WebFetchTool with Provider Pattern

**File**: `pkg/tools/web.go`

```go
// FetchProvider interface for different fetch backends
type FetchProvider interface {
	Fetch(ctx context.Context, url string, extractMode string) (string, error)
}

// WebFetchTool using provider pattern
type WebFetchTool struct {
	provider  FetchProvider
	maxChars  int
}

type WebFetchToolOptions struct {
	ZaiEnabled  bool
	ZaiAPIKey   string
	ZaiTimeout  int
	MaxChars    int
}

func NewWebFetchTool(opts WebFetchToolOptions) *WebFetchTool {
	maxChars := 50000
	if opts.MaxChars > 0 {
		maxChars = opts.MaxChars
	}

	var provider FetchProvider
	if opts.ZaiEnabled && opts.ZaiAPIKey != "" {
		provider = NewZaiFetchProvider(opts.ZaiAPIKey, time.Duration(opts.ZaiTimeout)*time.Second)
	} else {
		provider = &NativeFetchProvider{} // Existing implementation
	}

	return &WebFetchTool{
		provider: provider,
		maxChars: maxChars,
	}
}
```

---

## 4. JSON Configuration Schema

### Complete `tools.web` Configuration

```json
{
  "tools": {
    "web": {
      "zai": {
        "enabled": {
          "type": "boolean",
          "default": false,
          "description": "Enable ZAI MCP for web search and fetch"
        },
        "api_key": {
          "type": "string",
          "default": "",
          "description": "Z.AI API key (or set Z_AI_API_KEY env var)"
        },
        "max_results": {
          "type": "integer",
          "default": 10,
          "description": "Maximum search results (1-20)",
          "minimum": 1,
          "maximum": 20
        },
        "timeout": {
          "type": "integer",
          "default": 60,
          "description": "Request timeout in seconds",
          "minimum": 10,
          "maximum": 300
        }
      },
      "brave": {
        "enabled": {
          "type": "boolean",
          "default": false,
          "description": "Enable Brave Search API"
        },
        "api_key": {
          "type": "string",
          "default": "",
          "description": "Brave Search API key"
        },
        "max_results": {
          "type": "integer",
          "default": 5,
          "minimum": 1,
          "maximum": 20
        }
      },
      "duckduckgo": {
        "enabled": {
          "type": "boolean",
          "default": true,
          "description": "Enable DuckDuckGo HTML scraping (no API key needed)"
        },
        "max_results": {
          "type": "integer",
          "default": 5,
          "minimum": 1,
          "maximum": 10
        }
      }
    }
  }
}
```

### Example Configuration

**File**: `config/config.example.json`

```json
{
  "tools": {
    "web": {
      "zai": {
        "enabled": true,
        "api_key": "",
        "max_results": 10,
        "timeout": 60
      },
      "brave": {
        "enabled": false,
        "api_key": "",
        "max_results": 5
      },
      "duckduckgo": {
        "enabled": true,
        "max_results": 5
      }
    }
  }
}
```

### Provider Priority

When multiple providers are enabled:

| Priority | Provider | Condition |
|----------|----------|-----------|
| 1 (Highest) | ZAI MCP | `zai.enabled=true` AND `zai.api_key` is set |
| 2 | Brave Search | `brave.enabled=true` AND `brave.api_key` is set |
| 3 (Default) | DuckDuckGo | `duckduckgo.enabled=true` (no API key required) |

---

## 5. Environment Variables

| Variable | Description |
|----------|-------------|
| `PICOCLAW_TOOLS_WEB_ZAI_ENABLED` | Enable ZAI MCP |
| `PICOCLAW_TOOLS_WEB_ZAI_API_KEY` | Z.AI API key |
| `PICOCLAW_TOOLS_WEB_ZAI_MAX_RESULTS` | Max search results |
| `PICOCLAW_TOOLS_WEB_ZAI_TIMEOUT` | Request timeout (seconds) |

Alternative (fallback): `Z_AI_API_KEY` (same as nanobot uses)

---

## 6. Implementation Checklist

### Phase 1: MCP Client Infrastructure
- [ ] Add `github.com/modelcontextprotocol/go-sdk` dependency to `go.mod`
- [ ] Create `pkg/mcp/client.go` with HTTP transport MCP client
- [ ] Add error handling and retry logic
- [ ] Write unit tests for MCP client

### Phase 2: ZAI Providers
- [ ] Create `pkg/tools/zai_search.go` with `ZaiSearchProvider`
- [ ] Create `pkg/tools/zai_fetch.go` with `ZaiFetchProvider`
- [ ] Update `pkg/tools/web.go` with new provider options
- [ ] Add `FetchProvider` interface for web fetch

### Phase 3: Configuration
- [ ] Add `ZaiConfig` struct to `pkg/config/config.go`
- [ ] Update `DefaultConfig()` with ZAI defaults
- [ ] Update `config/config.example.json`
- [ ] Add environment variable support

### Phase 4: Testing
- [ ] Unit tests for ZAI providers (mocked MCP responses)
- [ ] Integration tests (manual, requires API key)
- [ ] Test fallback to DuckDuckGo when ZAI unavailable

### Phase 5: Documentation
- [ ] Update README.md with ZAI configuration instructions
- [ ] Add troubleshooting section for common MCP errors

---

## 7. Benefits

| Feature | Benefit |
|---------|---------|
| **Quality** | ZAI Prime search provides higher quality results |
| **Reliability** | MCP protocol with proper error handling |
| **Flexibility** | Easy switching between providers via config |
| **Consistency** | Same pattern as nanobot for code familiarity |
| **No Scraping** | ZAI uses official API, no HTML scraping needed |

---

## 8. Migration Notes

### For Existing Users

1. **No breaking changes**: Existing configs work unchanged
2. **Opt-in**: ZAI is disabled by default
3. **Fallback**: If ZAI fails, can fall back to DuckDuckGo

### Recommended Setup

```json
{
  "tools": {
    "web": {
      "zai": {
        "enabled": true,
        "api_key": "",
        "max_results": 10
      },
      "duckduckgo": {
        "enabled": true,
        "max_results": 5
      }
    }
  }
}
```

Set `Z_AI_API_KEY` environment variable or configure `api_key` in config file.

---

## 9. References

- [Official Go MCP SDK](https://github.com/modelcontextprotocol/go-sdk)
- [MCP Specification](https://modelcontextprotocol.io/docs/sdk)
- [Go SDK Documentation](https://pkg.go.dev/github.com/modelcontextprotocol/go-sdk/mcp)
- [Nanobot ZAI Implementation](../nanobot-repo/nanobot/agent/tools/zai_web.py)
