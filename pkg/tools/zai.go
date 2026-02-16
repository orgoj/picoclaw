package tools

import (
	"context"

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
