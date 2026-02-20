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
	Provider                providers.LLMProvider
	Model                   string
	Tools                   *ToolRegistry
	MaxIterations           int
	MaxToolIterations       int
	RunID                   string
	PullInjectedMessages    func() []string
	LLMOptions              map[string]any
	ContextLimit            int // Max total chars in message history. 0 = no limit.
	HistoryMessageThreshold int // Max number of messages before trimming. 0 = no limit.
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
	toolIteration := 0
	var finalContent string
	completed := false

	for iteration < config.MaxIterations {
		iteration++

		if config.PullInjectedMessages != nil {
			injected := config.PullInjectedMessages()
			if len(injected) > 0 {
				for _, msg := range injected {
					messages = append(messages, providers.Message{
						Role:    "user",
						Content: msg,
					})
				}
				logger.WarnCF("toolloop", "Injected external message(s) into active loop", map[string]any{
					"run_id":         config.RunID,
					"iteration":      iteration,
					"injected_count": len(injected),
				})
			}
		}

		// Trim by message count if threshold configured
		if config.HistoryMessageThreshold > 0 {
			messages = trimMessagesByCount(messages, config.HistoryMessageThreshold)
		}

		// Trim by character count if limit configured
		if config.ContextLimit > 0 {
			messages = trimMessages(messages, config.ContextLimit)
		}

		logger.DebugCF("toolloop", "LLM iteration",
			map[string]any{
				"run_id":    config.RunID,
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
					"run_id":    config.RunID,
					"iteration": iteration,
					"error":     err.Error(),
				})
			return nil, fmt.Errorf("LLM call failed: %w", err)
		}

		// 4. If no tool calls, we're done
		if len(response.ToolCalls) == 0 {
			finalContent = response.Content
			completed = true
			logger.InfoCF("toolloop", "LLM response without tool calls (direct answer)",
				map[string]any{
					"run_id":        config.RunID,
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
				"run_id":         config.RunID,
				"tools":          toolNames,
				"count":          len(response.ToolCalls),
				"iteration":      iteration,
				"tool_iteration": toolIteration + 1,
			})
		toolIteration++
		if config.MaxToolIterations > 0 && toolIteration > config.MaxToolIterations {
			return nil, fmt.Errorf("tool loop reached max tool iterations (%d) without final response", config.MaxToolIterations)
		}

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
					"run_id":    config.RunID,
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

	if !completed {
		return nil, fmt.Errorf("tool loop reached max iterations (%d) without final response", config.MaxIterations)
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

// trimMessagesByCount trims message history when the message count exceeds maxMessages.
// It preserves messages[0] (system prompt) and messages[1] (initial user message),
// and removes the oldest complete "rounds" (assistant + tool results) from the middle.
// This is similar to trimMessages but based on message count instead of character count.
func trimMessagesByCount(messages []providers.Message, maxMessages int) []providers.Message {
	if maxMessages <= 0 || len(messages) <= maxMessages {
		return messages
	}

	// Always preserve system prompt (index 0) and initial user message (index 1)
	// Find complete rounds starting from index 2
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

	if len(rounds) == 0 {
		return messages
	}

	// Calculate how many messages to keep
	messagesToKeep := maxMessages - 2 // Reserve 2 for system + initial user

	// Count messages in rounds from newest to oldest
	keptMessages := 0
	trimFrom := len(rounds)
	for trimFrom > 0 {
		roundIdx := trimFrom - 1
		roundMsgs := rounds[roundIdx].end - rounds[roundIdx].start
		if keptMessages+roundMsgs > messagesToKeep {
			break
		}
		keptMessages += roundMsgs
		trimFrom = roundIdx
	}

	// If we can't keep any rounds, just return the first 2 messages
	if trimFrom >= len(rounds) {
		logger.InfoCF("toolloop", "Context trimmed by count",
			map[string]any{
				"method":      "count",
				"original":    len(messages),
				"max":         maxMessages,
				"rounds_kept": 0,
				"final_count": 2,
			})
		return messages[:2]
	}

	// Build result
	result := make([]providers.Message, 0, 2+keptMessages)
	result = append(result, messages[:2]...)
	for _, r := range rounds[trimFrom:] {
		result = append(result, messages[r.start:r.end]...)
	}

	logger.InfoCF("toolloop", "Context trimmed by count",
		map[string]any{
			"method":         "count",
			"original":       len(messages),
			"max":            maxMessages,
			"rounds_dropped": trimFrom,
			"rounds_kept":    len(rounds) - trimFrom,
			"final_count":    len(result),
		})

	return result
}
