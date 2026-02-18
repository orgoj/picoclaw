// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/utils"
)

// ToolLoopConfig configures the tool execution loop.
type ToolLoopConfig struct {
	Provider      providers.LLMProvider
	Model         string
	Tools         *ToolRegistry
	MaxIterations int
	LLMOptions    map[string]any
	ContextLimit  int // Max total chars in message history. 0 = no limit.
}

// ToolLoopResult contains the result of running the tool loop.
type ToolLoopResult struct {
	Content    string
	Iterations int
}

// RunToolLoop executes the LLM + tool call iteration loop.
// This is the core agent logic that can be reused by both main agent and subagents.
func RunToolLoop(ctx context.Context, config ToolLoopConfig, messages []providers.Message, channel, chatID string) (*ToolLoopResult, error) {
	iteration := 0
	var finalContent string

	for iteration < config.MaxIterations {
		iteration++

		// Trim context if limit configured
		if config.ContextLimit > 0 {
			messages = trimMessages(messages, config.ContextLimit)
		}

		logger.DebugCF("toolloop", "LLM iteration",
			map[string]any{
				"iteration": iteration,
				"max":       config.MaxIterations,
			})

		// 1. Build tool definitions
		var providerToolDefs []providers.ToolDefinition
		if config.Tools != nil {
			providerToolDefs = config.Tools.ToProviderDefs()
		}

		// 2. Use LLM options from config (required)
		llmOpts := config.LLMOptions

		// 3. Call LLM
		response, err := config.Provider.Chat(ctx, messages, providerToolDefs, config.Model, llmOpts)
		if err != nil {
			logger.ErrorCF("toolloop", "LLM call failed",
				map[string]any{
					"iteration": iteration,
					"error":     err.Error(),
				})
			return nil, fmt.Errorf("LLM call failed: %w", err)
		}

		// 4. If no tool calls, we're done
		if len(response.ToolCalls) == 0 {
			finalContent = response.Content
			logger.InfoCF("toolloop", "LLM response without tool calls (direct answer)",
				map[string]any{
					"iteration":     iteration,
					"content_chars": len(finalContent),
				})
			break
		}

		// 5. Log tool calls
		toolNames := make([]string, 0, len(response.ToolCalls))
		for _, tc := range response.ToolCalls {
			toolNames = append(toolNames, tc.Name)
		}
		logger.InfoCF("toolloop", "LLM requested tool calls",
			map[string]any{
				"tools":     toolNames,
				"count":     len(response.ToolCalls),
				"iteration": iteration,
			})

		// 6. Build assistant message with tool calls
		assistantMsg := providers.Message{
			Role:    "assistant",
			Content: response.Content,
		}
		for _, tc := range response.ToolCalls {
			argumentsJSON, _ := json.Marshal(tc.Arguments)
			assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, providers.ToolCall{
				ID:   tc.ID,
				Type: "function",
				Function: &providers.FunctionCall{
					Name:      tc.Name,
					Arguments: string(argumentsJSON),
				},
			})
		}
		messages = append(messages, assistantMsg)

		// 7. Execute tool calls
		for _, tc := range response.ToolCalls {
			argsJSON, _ := json.Marshal(tc.Arguments)
			argsPreview := utils.Truncate(string(argsJSON), 200)
			logger.InfoCF("toolloop", fmt.Sprintf("Tool call: %s(%s)", tc.Name, argsPreview),
				map[string]any{
					"tool":      tc.Name,
					"iteration": iteration,
				})

			// Execute tool (no async callback for subagents - they run independently)
			var toolResult *ToolResult
			if config.Tools != nil {
				toolResult = config.Tools.ExecuteWithContext(ctx, tc.Name, tc.Arguments, channel, chatID, nil)
			} else {
				toolResult = ErrorResult("No tools available")
			}

			// Determine content for LLM
			contentForLLM := toolResult.ForLLM
			if contentForLLM == "" && toolResult.Err != nil {
				contentForLLM = toolResult.Err.Error()
			}

			// Add tool result message
			toolResultMsg := providers.Message{
				Role:       "tool",
				Content:    contentForLLM,
				ToolCallID: tc.ID,
			}
			messages = append(messages, toolResultMsg)
		}
	}

	return &ToolLoopResult{
		Content:    finalContent,
		Iterations: iteration,
	}, nil
}

// trimMessages reduces message history when total chars exceed maxChars.
// It preserves messages[0] (system prompt) and messages[1] (initial user message),
// and removes the oldest complete "rounds" (assistant + tool results) from the middle.
// A round is one assistant message with tool_calls plus all immediately following tool messages.
// This ensures tool_call_id references are never broken.
func trimMessages(messages []providers.Message, maxChars int) []providers.Message {
	if maxChars <= 0 || len(messages) <= 2 {
		return messages
	}

	total := 0
	for _, m := range messages {
		total += len(m.Content)
	}
	if total <= maxChars {
		return messages
	}

	// Find complete rounds starting from index 2 (after system + initial user).
	// A round = one assistant message + all immediately following tool messages.
	type roundSpan struct{ start, end int }
	var rounds []roundSpan
	i := 2
	for i < len(messages) {
		if messages[i].Role == "assistant" {
			j := i + 1
			for j < len(messages) && messages[j].Role == "tool" {
				j++
			}
			rounds = append(rounds, roundSpan{i, j})
			i = j
		} else {
			i++
		}
	}

	if len(rounds) <= 1 {
		return messages
	}

	// Drop oldest rounds one at a time until under limit.
	trimFrom := 1
	for trimFrom < len(rounds) {
		total = 0
		for _, m := range messages[:2] {
			total += len(m.Content)
		}
		for _, r := range rounds[trimFrom:] {
			for _, m := range messages[r.start:r.end] {
				total += len(m.Content)
			}
		}
		if total <= maxChars || trimFrom == len(rounds)-1 {
			break
		}
		trimFrom++
	}

	result := make([]providers.Message, 0, len(messages))
	result = append(result, messages[:2]...)
	for _, r := range rounds[trimFrom:] {
		result = append(result, messages[r.start:r.end]...)
	}

	logger.InfoCF("toolloop", "Context trimmed",
		map[string]any{
			"rounds_dropped": trimFrom,
			"rounds_kept":    len(rounds) - trimFrom,
			"new_msgs":       len(result),
		})

	return result
}
