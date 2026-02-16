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

	// Check for tool-level errors per V4 review recommendation
	if result.IsError {
		return "", fmt.Errorf("tool returned error: %s", text)
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
