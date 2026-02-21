// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package llm

import (
	"context"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// RetryConfig controls retry behavior for LLM calls.
type RetryConfig struct {
	MaxRetries              int
	RetryBackoffSeconds     int
	RetryMaxBackoffSeconds  int
	RateLimitBackoffSeconds int
	RateLimitMaxBackoffSecs int
	RetryMaxElapsedSeconds  int
}

// CallConfig defines one LLM call with retry policy.
type CallConfig struct {
	Provider     providers.LLMProvider
	Messages     []providers.Message
	Tools        []providers.ToolDefinition
	Model        string
	Options      map[string]any
	Retry        RetryConfig
	LogComponent string
	LogFields    map[string]any
}

// CallWithRetry executes one provider chat call with unified retry policy.
func CallWithRetry(ctx context.Context, cfg CallConfig) (*providers.LLMResponse, error) {
	retryCfg := normalizeRetryConfig(cfg.Retry)
	retryStart := time.Now()
	maxRetries := retryCfg.MaxRetries

	var response *providers.LLMResponse
	var err error

	for retry := 0; retry <= maxRetries; retry++ {
		response, err = cfg.Provider.Chat(ctx, cfg.Messages, cfg.Tools, cfg.Model, cfg.Options)
		if err == nil {
			return response, nil
		}

		errStr := err.Error()
		isRateLimit := isRateLimitError(errStr)
		isRetryable := isRetryableError(errStr)

		if isRateLimit && retry < maxRetries {
			waitSeconds := retryCfg.RateLimitBackoffSeconds * (retry + 1)
			if waitSeconds > retryCfg.RateLimitMaxBackoffSecs {
				waitSeconds = retryCfg.RateLimitMaxBackoffSecs
			}
			waitTime := time.Duration(waitSeconds) * time.Second
			waitTime, canWait := clampRetryWait(waitTime, retryCfg.RetryMaxElapsedSeconds, retryStart)
			if !canWait {
				break
			}

			logger.WarnCF(cfg.LogComponent, "Rate limited, waiting", mergeLogFields(cfg.LogFields, map[string]any{
				"retry":        retry + 1,
				"max_retries":  maxRetries,
				"wait_seconds": waitTime.Seconds(),
				"error":        errStr,
			}))

			if err := waitWithContext(ctx, waitTime); err != nil {
				return nil, err
			}
			continue
		}

		if isRetryable && retry < maxRetries {
			waitSeconds := retryCfg.RetryBackoffSeconds << retry
			if waitSeconds > retryCfg.RetryMaxBackoffSeconds {
				waitSeconds = retryCfg.RetryMaxBackoffSeconds
			}
			waitTime := time.Duration(waitSeconds) * time.Second
			waitTime, canWait := clampRetryWait(waitTime, retryCfg.RetryMaxElapsedSeconds, retryStart)
			if !canWait {
				break
			}

			logger.WarnCF(cfg.LogComponent, "LLM call failed, retrying", mergeLogFields(cfg.LogFields, map[string]any{
				"retry":        retry + 1,
				"max_retries":  maxRetries,
				"wait_seconds": waitTime.Seconds(),
				"error":        errStr,
			}))

			if err := waitWithContext(ctx, waitTime); err != nil {
				return nil, err
			}
			continue
		}

		break
	}

	return nil, err
}

func normalizeRetryConfig(c RetryConfig) RetryConfig {
	if c.MaxRetries < 0 {
		c.MaxRetries = 0
	}
	if c.RetryBackoffSeconds < 1 {
		c.RetryBackoffSeconds = 1
	}
	if c.RetryMaxBackoffSeconds < 1 {
		c.RetryMaxBackoffSeconds = 1
	}
	if c.RateLimitBackoffSeconds < 1 {
		c.RateLimitBackoffSeconds = 1
	}
	if c.RateLimitMaxBackoffSecs < 1 {
		c.RateLimitMaxBackoffSecs = 1
	}
	return c
}

func isRetryableError(errStr string) bool {
	lowerErr := strings.ToLower(errStr)
	return strings.Contains(lowerErr, "unexpected eof") ||
		strings.Contains(lowerErr, "connection reset") ||
		strings.Contains(lowerErr, "timeout") ||
		strings.Contains(lowerErr, "context deadline exceeded") ||
		strings.Contains(lowerErr, "client.timeout exceeded") ||
		strings.Contains(lowerErr, "temporary failure") ||
		strings.Contains(lowerErr, "status=5") ||
		strings.Contains(lowerErr, "500") ||
		strings.Contains(lowerErr, "502") ||
		strings.Contains(lowerErr, "503") ||
		strings.Contains(lowerErr, "504")
}

func isRateLimitError(errStr string) bool {
	return strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "too many requests")
}

func clampRetryWait(wait time.Duration, retryMaxElapsedSeconds int, retryStart time.Time) (time.Duration, bool) {
	if retryMaxElapsedSeconds <= 0 {
		return wait, true
	}

	maxElapsed := time.Duration(retryMaxElapsedSeconds) * time.Second
	elapsed := time.Since(retryStart)
	remaining := maxElapsed - elapsed
	if remaining <= 0 {
		return 0, false
	}
	if wait > remaining {
		wait = remaining
	}
	return wait, true
}

func waitWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func mergeLogFields(base map[string]any, extra map[string]any) map[string]any {
	if len(base) == 0 && len(extra) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}
