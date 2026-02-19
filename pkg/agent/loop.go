// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/constants"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/mcp"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/session"
	"github.com/sipeed/picoclaw/pkg/state"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/sipeed/picoclaw/pkg/utils"
)

type AgentLoop struct {
	cfg                     *config.Config
	bus                     *bus.MessageBus
	provider                providers.LLMProvider
	workspace               string
	model                   string
	contextWindow           int     // Maximum context window size in tokens
	maxIterations           int     // Max tool iterations for main agent
	maxTokens               int     // Max tokens for LLM responses
	temperature             float64 // LLM temperature
	llmTimeout              int     // LLM API timeout in seconds
	llmMaxRetries           int     // Max retry attempts after the initial failed LLM call
	llmRetryBackoffSeconds  int     // Base backoff for retryable errors (seconds, exponential)
	llmRetryMaxBackoff      int     // Max backoff for retryable errors (seconds)
	llmRateLimitBackoff     int     // Base backoff for 429/rate-limit errors (seconds, linear)
	llmRateLimitMaxBackoff  int     // Max backoff for 429/rate-limit errors (seconds)
	llmRetryMaxElapsed      int     // Max total wait time spent retrying a single LLM call (seconds)
	memoryThreshold         float64 // Threshold for memory summarization (0.0-1.0)
	historyMessageThreshold int     // Number of messages before triggering summarization
	sessions                *session.SessionManager
	state                   *state.Manager
	contextBuilder          *ContextBuilder
	tools                   *tools.ToolRegistry
	subagentManager         *tools.SubagentManager
	running                 atomic.Bool
	summarizing             sync.Map // Tracks which sessions are currently being summarized
	idleEnabled             bool
	idleTimeout             time.Duration
	idleRepeat              bool
	idleRecentSubagents     int // Number of recent subagents to show in IDLE prompt
	runCounter              atomic.Uint64
}

// processOptions configures how a message is processed
type processOptions struct {
	SessionKey      string // Session identifier for history/context
	Channel         string // Target channel for tool execution
	ChatID          string // Target chat ID for tool execution
	UserMessage     string // User message content (may include prefix)
	DefaultResponse string // Response when LLM returns empty
	EnableSummary   bool   // Whether to trigger summarization
	SendResponse    bool   // Whether to send response via bus
	NoHistory       bool   // If true, don't load session history (for heartbeat)
}

// createToolRegistry creates a tool registry with common tools.
// This is shared between main agent and subagents.
func createToolRegistry(workspace string, restrict bool, cfg *config.Config, msgBus *bus.MessageBus) *tools.ToolRegistry {
	registry := tools.NewToolRegistry()

	// File system tools
	registry.Register(tools.NewReadFileTool(workspace, restrict))
	registry.Register(tools.NewWriteFileTool(workspace, restrict))
	registry.Register(tools.NewListDirTool(workspace, restrict))
	registry.Register(tools.NewEditFileTool(workspace, restrict))
	registry.Register(tools.NewAppendFileTool(workspace, restrict))

	// Shell execution
	registry.Register(tools.NewExecTool(workspace, restrict))

	// Web search - initialize ZAI client if enabled
	var zaiSearchClient *mcp.Client
	var zaiFetchClient *mcp.Client

	if cfg.Tools.Web.ZAI.Enabled && cfg.Tools.Web.ZAI.APIKey != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Search client
		if cfg.Tools.Web.ZAI.Endpoint != "" {
			client, err := mcp.NewClientWithTimeout(ctx, cfg.Tools.Web.ZAI.Endpoint, cfg.Tools.Web.ZAI.APIKey, cfg.Tools.Web.ZAI.Timeout)
			if err != nil {
				logger.WarnCF("agent", "Failed to connect ZAI Search MCP client: %v", map[string]interface{}{"error": err.Error(), "endpoint": cfg.Tools.Web.ZAI.Endpoint})
			} else {
				zaiSearchClient = client
				logger.InfoCF("agent", "ZAI Search MCP client connected", map[string]interface{}{"endpoint": cfg.Tools.Web.ZAI.Endpoint})
			}
		}

		// Fetch client (optional, fallback to Search client if not specified)
		if cfg.Tools.Web.ZAI.EndpointFetch != "" {
			client, err := mcp.NewClientWithTimeout(ctx, cfg.Tools.Web.ZAI.EndpointFetch, cfg.Tools.Web.ZAI.APIKey, cfg.Tools.Web.ZAI.Timeout)
			if err != nil {
				logger.WarnCF("agent", "Failed to connect ZAI Fetch MCP client: %v", map[string]interface{}{"error": err.Error(), "endpoint": cfg.Tools.Web.ZAI.EndpointFetch})
			} else {
				zaiFetchClient = client
				logger.InfoCF("agent", "ZAI Fetch MCP client connected", map[string]interface{}{"endpoint": cfg.Tools.Web.ZAI.EndpointFetch})
			}
		} else {
			zaiFetchClient = zaiSearchClient
		}
	}

	if searchTool := tools.NewWebSearchTool(tools.WebSearchToolOptions{
		BraveAPIKey:          cfg.Tools.Web.Brave.APIKey,
		BraveMaxResults:      cfg.Tools.Web.Brave.MaxResults,
		BraveEnabled:         cfg.Tools.Web.Brave.Enabled,
		DuckDuckGoMaxResults: cfg.Tools.Web.DuckDuckGo.MaxResults,
		DuckDuckGoEnabled:    cfg.Tools.Web.DuckDuckGo.Enabled,
		ZAIEnabled:           cfg.Tools.Web.ZAI.Enabled,
		ZAIMaxResults:        cfg.Tools.Web.ZAI.MaxResults,
		ZAIClient:            zaiSearchClient,
	}); searchTool != nil {
		registry.Register(searchTool)
	}
	registry.Register(tools.NewWebFetchTool(50000, zaiFetchClient, cfg.Tools.Web.ZAI.Timeout))

	// Hardware tools (I2C, SPI) - Linux only, returns error on other platforms
	registry.Register(tools.NewI2CTool())
	registry.Register(tools.NewSPITool())

	// Message tool - available to both agent and subagent
	// Subagent uses it to communicate directly with user
	messageTool := tools.NewMessageTool()
	messageTool.SetSendCallback(func(channel, chatID, content string) error {
		if ok := msgBus.PublishOutbound(bus.OutboundMessage{
			Channel: channel,
			ChatID:  chatID,
			Content: content,
		}); !ok {
			return fmt.Errorf("outbound queue timeout")
		}
		return nil
	})
	registry.Register(messageTool)

	return registry
}

func NewAgentLoop(cfg *config.Config, msgBus *bus.MessageBus, provider providers.LLMProvider) *AgentLoop {
	workspace := cfg.WorkspacePath()
	os.MkdirAll(workspace, 0755)

	restrict := cfg.Agents.Defaults.RestrictToWorkspace

	// Create tool registry for main agent
	toolsRegistry := createToolRegistry(workspace, restrict, cfg, msgBus)

	// Create subagent manager with its own tool registry
	subagentManager := tools.NewSubagentManager(provider, cfg, workspace, msgBus)
	subagentTools := createToolRegistry(workspace, restrict, cfg, msgBus)
	// Subagent doesn't need spawn/subagent tools to avoid recursion
	subagentManager.SetTools(subagentTools)

	// Register spawn/subagent tools (configurable)
	if cfg.Tools.Spawn.Enabled {
		spawnTool := tools.NewSpawnTool(subagentManager)
		toolsRegistry.Register(spawnTool)
	}
	if cfg.Tools.Subagent.Enabled {
		subagentTool := tools.NewSubagentTool(subagentManager)
		toolsRegistry.Register(subagentTool)
	}

	// Register subagent management tools only when async spawn is enabled.
	if cfg.Tools.Spawn.Enabled {
		toolsRegistry.Register(tools.NewSubagentStatusTool(subagentManager))
		toolsRegistry.Register(tools.NewSubagentHistoryTool(subagentManager))
		toolsRegistry.Register(tools.NewSubagentMessageTool(subagentManager))
		toolsRegistry.Register(tools.NewSubagentCancelTool(subagentManager))
	}

	sessionsManager := session.NewSessionManager(filepath.Join(workspace, "sessions"))

	// Create state manager for atomic state persistence
	stateManager := state.NewManager(workspace)

	// Create context builder and set tools registry
	contextBuilder := NewContextBuilder(workspace)
	contextBuilder.SetToolsRegistry(toolsRegistry)

	// Context window fallback: use ContextWindow if set, otherwise MaxTokens
	contextWindow := cfg.Agents.Defaults.ContextWindow
	if contextWindow <= 0 {
		contextWindow = cfg.Agents.Defaults.MaxTokens
	}

	return &AgentLoop{
		cfg:                     cfg,
		bus:                     msgBus,
		provider:                provider,
		workspace:               workspace,
		model:                   cfg.Agents.Defaults.Model,
		contextWindow:           contextWindow,
		maxIterations:           cfg.Agents.Defaults.MaxToolIterations,
		maxTokens:               cfg.Agents.Defaults.MaxTokens,
		temperature:             cfg.Agents.Defaults.Temperature,
		llmTimeout:              cfg.Agents.Defaults.LLMTimeout,
		llmMaxRetries:           maxInt(cfg.Agents.Defaults.LLMMaxRetries, 0),
		llmRetryBackoffSeconds:  maxInt(cfg.Agents.Defaults.LLMRetryBackoffSeconds, 1),
		llmRetryMaxBackoff:      maxInt(cfg.Agents.Defaults.LLMRetryMaxBackoff, 1),
		llmRateLimitBackoff:     maxInt(cfg.Agents.Defaults.LLMRateLimitBackoff, 1),
		llmRateLimitMaxBackoff:  maxInt(cfg.Agents.Defaults.LLMRateLimitMaxBackoff, 1),
		llmRetryMaxElapsed:      maxInt(cfg.Agents.Defaults.LLMRetryMaxElapsed, 0),
		memoryThreshold:         cfg.Agents.Defaults.MemoryThreshold,
		historyMessageThreshold: cfg.Agents.Defaults.HistoryMessageThreshold,
		idleEnabled:             cfg.Idle.Enabled,
		idleTimeout: func() time.Duration {
			minutes := cfg.Idle.TimeoutMinutes
			if minutes <= 0 {
				minutes = 5
			}
			return time.Duration(minutes) * time.Minute
		}(),
		idleRepeat:          cfg.Idle.Repeat,
		idleRecentSubagents: cfg.Idle.RecentSubagents,
		sessions:            sessionsManager,
		state:               stateManager,
		contextBuilder:      contextBuilder,
		tools:               toolsRegistry,
		subagentManager:     subagentManager,
		summarizing:         sync.Map{},
	}
}

func (al *AgentLoop) Run(ctx context.Context) error {
	al.running.Store(true)

	// Log config at session start
	if al.cfg != nil {
		summary := al.cfg.Summary()
		if err := config.AppendToDailyLog(al.workspace, summary); err != nil {
			logger.WarnCF("agent", "Failed to log config to daily log: %v", map[string]interface{}{"error": err.Error()})
		}
	}

	idleTriggered := false

	for al.running.Load() {
		select {
		case <-ctx.Done():
			return nil
		default:
			var msg bus.InboundMessage
			var ok bool

			if al.idleEnabled && al.idleTimeout > 0 {
				var timedOut bool
				msg, ok, timedOut = al.bus.ConsumeInboundWithTimeout(ctx, al.idleTimeout)
				if timedOut {
					if !idleTriggered || al.idleRepeat {
						idleTriggered = true
						al.triggerIdle(ctx)
					}
					continue
				}
			} else {
				msg, ok = al.bus.ConsumeInbound(ctx)
			}

			if !ok {
				continue
			}

			// Got a message — reset idle state
			idleTriggered = false

			// Recover from panics to prevent agent crash
			func() {
				defer func() {
					if r := recover(); r != nil {
						logger.ErrorCF("agent", "Recovered from panic in message processing",
							map[string]interface{}{
								"panic":   fmt.Sprintf("%v", r),
								"channel": msg.Channel,
								"chat_id": msg.ChatID,
								"session": msg.SessionKey,
							})
						// Send user-friendly error message
						if ok := al.bus.PublishOutbound(bus.OutboundMessage{
							Channel: msg.Channel,
							ChatID:  msg.ChatID,
							Content: "⚠️ An internal error occurred. The agent has recovered and is still running. Please try again.",
						}); !ok {
							logger.WarnCF("agent", "Failed to publish panic recovery message: outbound queue timeout",
								map[string]interface{}{
									"channel": msg.Channel,
									"chat_id": msg.ChatID,
								})
						}
					}
				}()

				response, err := al.processMessage(ctx, msg)
				if err != nil {
					// Check if it's an API/network error and provide user-friendly message
					response = al.formatErrorMessage(err, msg.SessionKey)
				}

				if response != "" {
					if ok := al.bus.PublishOutbound(bus.OutboundMessage{
						Channel: msg.Channel,
						ChatID:  msg.ChatID,
						Content: response,
					}); !ok {
						logger.WarnCF("agent", "Failed to publish agent response: outbound queue timeout",
							map[string]interface{}{
								"channel": msg.Channel,
								"chat_id": msg.ChatID,
							})
					}
				}
			}()
		}
	}

	return nil
}

// formatErrorMessage converts technical errors into user-friendly messages
// and injects error details into session history so the agent can react
func (al *AgentLoop) formatErrorMessage(err error, sessionKey string) string {
	errStr := err.Error()
	var userMessage string
	var errorDetails string

	// Check for timeout errors
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "context deadline exceeded") {
		userMessage = "⏱️ API timeout (z.ai) - please try again"
		errorDetails = fmt.Sprintf("API_TIMEOUT: %s", errStr)
		logger.ErrorCF("agent", "API timeout error", map[string]interface{}{
			"error": errStr,
		})
	}
	// Check for rate limit (429)
	if strings.Contains(errStr, "429") || strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "too many requests") {
		userMessage = "🚦 API rate limited - waiting..."
		errorDetails = fmt.Sprintf("API_RATE_LIMIT: %s", errStr)
		logger.ErrorCF("agent", "API rate limited", map[string]interface{}{
			"error": errStr,
		})
	}

	// Check for 5xx server errors
	if userMessage == "" && (strings.Contains(errStr, "status=5") || strings.Contains(errStr, "500") || strings.Contains(errStr, "502") || strings.Contains(errStr, "503") || strings.Contains(errStr, "504")) {
		// Extract status code if present
		statusCode := "500"
		for _, code := range []string{"500", "502", "503", "504"} {
			if strings.Contains(errStr, code) {
				statusCode = code
				break
			}
		}
		userMessage = fmt.Sprintf("⚠️ API error: status=%s - temporary issue", statusCode)
		errorDetails = fmt.Sprintf("API_SERVER_ERROR: %s", errStr)
		logger.ErrorCF("agent", "API server error", map[string]interface{}{
			"error": errStr,
		})
	}

	// Check for network errors
	if userMessage == "" && (strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "network is unreachable") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "DNS") ||
		strings.Contains(errStr, "network")) {
		// Extract specific network error type
		if strings.Contains(errStr, "connection refused") {
			userMessage = "🔌 Network error: connection refused"
		} else if strings.Contains(errStr, "network is unreachable") {
			userMessage = "🔌 Network error: network unreachable"
		} else {
			userMessage = "🔌 Network error: connection failed"
		}
		errorDetails = fmt.Sprintf("NETWORK_ERROR: %s", errStr)
		logger.ErrorCF("agent", "Network error occurred", map[string]interface{}{
			"error": errStr,
		})
	}

	// Check for connection reset/EOF (could be network or API)
	if userMessage == "" && (strings.Contains(errStr, "unexpected EOF") || strings.Contains(errStr, "connection reset")) {
		userMessage = "🔌 Network error: connection interrupted"
		errorDetails = fmt.Sprintf("CONNECTION_ERROR: %s", errStr)
		logger.ErrorCF("agent", "Connection error", map[string]interface{}{
			"error": errStr,
		})
	}

	// Check for generic API request failures
	if userMessage == "" && strings.Contains(errStr, "API request failed") {
		userMessage = "⚠️ API request failed - please try again"
		errorDetails = fmt.Sprintf("API_REQUEST_FAILED: %s", errStr)
		logger.ErrorCF("agent", "API request failed", map[string]interface{}{
			"error": errStr,
		})
	}

	// Check for LLM call failures
	if userMessage == "" && strings.Contains(errStr, "LLM call failed") {
		userMessage = "⚠️ Failed to communicate with AI service - please try again"
		errorDetails = fmt.Sprintf("LLM_CALL_FAILED: %s", errStr)
		logger.ErrorCF("agent", "LLM call failed", map[string]interface{}{
			"error": errStr,
		})
	}

	// Fallback: Generic error message
	if userMessage == "" {
		userMessage = fmt.Sprintf("⚠️ An error occurred: %s", errStr)
		errorDetails = fmt.Sprintf("GENERIC_ERROR: %s", errStr)
		logger.ErrorCF("agent", "Error processing message", map[string]interface{}{
			"error": errStr,
		})
	}

	// Inject error details into session history so agent knows what happened
	if sessionKey != "" && errorDetails != "" {
		al.sessions.AddMessage(sessionKey, "system", errorDetails)
	}

	return userMessage
}

func (al *AgentLoop) Stop() {
	al.running.Store(false)
}

// triggerIdle runs IDLE.md as a prompt when the agent has been idle.
// Uses heartbeat-style processing: no session history, result sent to last channel.
func (al *AgentLoop) triggerIdle(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			logger.ErrorCF("agent", "Recovered from panic in idle processing",
				map[string]interface{}{"panic": fmt.Sprintf("%v", r)})
		}
	}()

	idlePath := filepath.Join(al.workspace, "IDLE.md")
	data, err := os.ReadFile(idlePath)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.ErrorCF("agent", "Error reading IDLE.md", map[string]interface{}{"error": err.Error()})
		}
		return
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		return
	}

	// LastChannel stores "platform:chatID" format (same as heartbeat)
	lastChannel := al.state.GetLastChannel()
	if lastChannel == "" {
		logger.DebugC("agent", "No last channel for idle trigger, skipping")
		return
	}
	parts := strings.SplitN(lastChannel, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		logger.WarnCF("agent", "Invalid last channel format for idle", map[string]interface{}{"last_channel": lastChannel})
		return
	}
	channel, chatID := parts[0], parts[1]

	if constants.IsInternalChannel(channel) {
		logger.DebugC("agent", "No external channel for idle trigger, skipping")
		return
	}

	logger.InfoCF("agent", "Triggering idle processing", map[string]interface{}{
		"channel": channel,
		"chat_id": chatID,
	})

	now := time.Now().Format("2006-01-02 15:04:05")

	// Build subagent status context
	subagentStatus := al.buildSubagentStatus()

	prompt := fmt.Sprintf("# Idle Check\n\nCurrent time: %s\n\n%s", now, content)
	if subagentStatus != "" {
		prompt = fmt.Sprintf("# Idle Check\n\nCurrent time: %s\n\n%s\n\n%s", now, subagentStatus, content)
	}

	response, err := al.runAgentLoop(ctx, processOptions{
		SessionKey:      "idle",
		Channel:         channel,
		ChatID:          chatID,
		UserMessage:     prompt,
		DefaultResponse: "",
		EnableSummary:   false,
		SendResponse:    false,
		NoHistory:       true,
	})

	if err != nil {
		logger.ErrorCF("agent", "Idle processing error", map[string]interface{}{"error": err.Error()})
		return
	}

	if response != "" && response != "IDLE_OK" {
		if ok := al.bus.PublishOutbound(bus.OutboundMessage{
			Channel: channel,
			ChatID:  chatID,
			Content: response,
		}); !ok {
			logger.WarnCF("agent", "Failed to publish idle response: outbound queue timeout", map[string]interface{}{
				"channel": channel,
				"chat_id": chatID,
			})
		}
	}
}

// buildSubagentStatus creates a formatted string with active and recent subagent status.
// Returns empty string if no subagent activity.
func (al *AgentLoop) buildSubagentStatus() string {
	var parts []string

	// Get active (running) subagents
	runningTasks := al.subagentManager.GetRunningTasks()
	if len(runningTasks) > 0 {
		var activeLines []string
		for _, task := range runningTasks {
			name := task.Label
			if name == "" {
				name = task.ID
			}
			startTime := "unknown"
			if task.Started > 0 {
				startTime = time.Unix(task.Started, 0).Format("15:04")
			}
			activeLines = append(activeLines, fmt.Sprintf("• %s [running since %s]", name, startTime))
		}
		parts = append(parts, fmt.Sprintf("📊 Active Subagents:\n%s", strings.Join(activeLines, "\n")))
	}

	// Get recent completed subagents
	recentLimit := al.idleRecentSubagents
	if recentLimit <= 0 {
		recentLimit = 5
	}
	recentTasks := al.subagentManager.GetRecentTasks(recentLimit)

	// Filter to only completed/failed/cancelled tasks
	var completedTasks []*tools.SubagentTask
	for _, task := range recentTasks {
		if task.Status != "running" && task.Status != "pending" {
			completedTasks = append(completedTasks, task)
		}
	}

	if len(completedTasks) > 0 {
		var recentLines []string
		for _, task := range completedTasks {
			name := task.Label
			if name == "" {
				name = task.ID
			}
			timeRange := "unknown"
			if task.Started > 0 {
				startStr := time.Unix(task.Started, 0).Format("15:04")
				if task.Ended > 0 {
					endStr := time.Unix(task.Ended, 0).Format("15:04")
					timeRange = fmt.Sprintf("%s - %s", startStr, endStr)
				} else {
					timeRange = startStr
				}
			}

			status := "❓"
			switch task.Status {
			case "completed":
				status = "✅"
			case "failed":
				status = "❌"
			case "cancelled":
				status = "⏹️"
			}

			recentLines = append(recentLines, fmt.Sprintf("• %s [%s] %s", name, timeRange, status))
		}
		parts = append(parts, fmt.Sprintf("📊 Recent Completed (%d):\n%s", len(completedTasks), strings.Join(recentLines, "\n")))
	}

	return strings.Join(parts, "\n\n")
}

func (al *AgentLoop) RegisterTool(tool tools.Tool) {
	al.tools.Register(tool)
}

func (al *AgentLoop) GetSubagentManager() *tools.SubagentManager {
	return al.subagentManager
}

// AgentStats holds runtime statistics for a specific agent session.
type AgentStats struct {
	MessageCount    int
	TokenEstimate   int
	ContextWindow   int
	MemoryThreshold float64
	HasSummary      bool
	IsSummarizing   bool
}

// GetSessionStats returns runtime statistics for the given session key.
func (al *AgentLoop) GetSessionStats(sessionKey string) AgentStats {
	history := al.sessions.GetHistory(sessionKey)
	summary := al.sessions.GetSummary(sessionKey)
	_, isSummarizing := al.summarizing.Load(sessionKey)
	return AgentStats{
		MessageCount:    len(history),
		TokenEstimate:   al.estimateTokens(history),
		ContextWindow:   al.contextWindow,
		MemoryThreshold: al.memoryThreshold,
		HasSummary:      summary != "",
		IsSummarizing:   isSummarizing,
	}
}

// RecordLastChannel records the last active channel for this workspace.
// This uses the atomic state save mechanism to prevent data loss on crash.
func (al *AgentLoop) RecordLastChannel(channel string) error {
	return al.state.SetLastChannel(channel)
}

// RecordLastChatID records the last active chat ID for this workspace.
// This uses the atomic state save mechanism to prevent data loss on crash.
func (al *AgentLoop) RecordLastChatID(chatID string) error {
	return al.state.SetLastChatID(chatID)
}

func (al *AgentLoop) ProcessDirect(ctx context.Context, content, sessionKey string) (string, error) {
	return al.ProcessDirectWithChannel(ctx, content, sessionKey, "cli", "direct")
}

func (al *AgentLoop) ProcessDirectWithChannel(ctx context.Context, content, sessionKey, channel, chatID string) (string, error) {
	msg := bus.InboundMessage{
		Channel:    channel,
		SenderID:   "cron",
		ChatID:     chatID,
		Content:    content,
		SessionKey: sessionKey,
	}

	return al.processMessage(ctx, msg)
}

// ProcessHeartbeat processes a heartbeat request without session history.
// Each heartbeat is independent and doesn't accumulate context.
func (al *AgentLoop) ProcessHeartbeat(ctx context.Context, content, channel, chatID string) (string, error) {
	return al.runAgentLoop(ctx, processOptions{
		SessionKey:      "heartbeat",
		Channel:         channel,
		ChatID:          chatID,
		UserMessage:     content,
		DefaultResponse: "I've completed processing but have no response to give.",
		EnableSummary:   false,
		SendResponse:    false,
		NoHistory:       true, // Don't load session history for heartbeat
	})
}

func (al *AgentLoop) processMessage(ctx context.Context, msg bus.InboundMessage) (string, error) {
	// Add message preview to log (show full content for error messages)
	var logContent string
	if strings.Contains(msg.Content, "Error:") || strings.Contains(msg.Content, "error") {
		logContent = msg.Content // Full content for errors
	} else {
		logContent = utils.Truncate(msg.Content, 80)
	}
	logger.InfoCF("agent", fmt.Sprintf("Processing message from %s:%s: %s", msg.Channel, msg.SenderID, logContent),
		map[string]interface{}{
			"channel":     msg.Channel,
			"chat_id":     msg.ChatID,
			"sender_id":   msg.SenderID,
			"session_key": msg.SessionKey,
		})

	// Route system messages to processSystemMessage
	if msg.Channel == "system" {
		return al.processSystemMessage(ctx, msg)
	}

	// Process as user message
	return al.runAgentLoop(ctx, processOptions{
		SessionKey:      msg.SessionKey,
		Channel:         msg.Channel,
		ChatID:          msg.ChatID,
		UserMessage:     msg.Content,
		DefaultResponse: "I've completed processing but have no response to give.",
		EnableSummary:   true,
		SendResponse:    false,
	})
}

func (al *AgentLoop) processSystemMessage(ctx context.Context, msg bus.InboundMessage) (string, error) {
	// Verify this is a system message
	if msg.Channel != "system" {
		return "", fmt.Errorf("processSystemMessage called with non-system message channel: %s", msg.Channel)
	}

	logger.InfoCF("agent", "Processing system message",
		map[string]interface{}{
			"sender_id": msg.SenderID,
			"chat_id":   msg.ChatID,
		})

	// Parse origin channel from chat_id (format: "channel:chat_id")
	var originChannel string
	if idx := strings.Index(msg.ChatID, ":"); idx > 0 {
		originChannel = msg.ChatID[:idx]
	} else {
		// Fallback
		originChannel = "cli"
	}

	// Extract subagent result from message content
	// Format: "Task 'label' completed.\n\nResult:\n<actual content>"
	content := msg.Content
	if idx := strings.Index(content, "Result:\n"); idx >= 0 {
		content = content[idx+8:] // Extract just the result part
	}

	// Extract directory from subagent task if available
	var directory string
	if strings.HasPrefix(msg.SenderID, "subagent:") {
		taskID := strings.TrimPrefix(msg.SenderID, "subagent:")
		if task, ok := al.subagentManager.GetTask(taskID); ok {
			directory = task.Directory
		}
	}

	// Skip internal channels - only log, don't send to user
	if constants.IsInternalChannel(originChannel) {
		logger.InfoCF("agent", "Subagent completed (internal channel)",
			map[string]interface{}{
				"sender_id":   msg.SenderID,
				"content_len": len(content),
				"channel":     originChannel,
				"directory":   directory,
			})
		return "", nil
	}

	// Agent acts as dispatcher only - subagent handles user interaction via message tool
	// Don't forward result here, subagent should use message tool to communicate with user
	logger.InfoCF("agent", "Subagent completed",
		map[string]interface{}{
			"sender_id":   msg.SenderID,
			"channel":     originChannel,
			"content_len": len(content),
			"directory":   directory,
		})

	// Return notification for main agent - it decides what to do with it
	notification := fmt.Sprintf("📢 Subagent %s completed:\n\n%s", msg.SenderID, content)
	return notification, nil
}

// runAgentLoop is the core message processing logic.
// It handles context building, LLM calls, tool execution, and response handling.
func (al *AgentLoop) runAgentLoop(ctx context.Context, opts processOptions) (string, error) {
	runID := al.nextRunID(opts.SessionKey)
	logger.InfoCF("agent", "Run started", map[string]interface{}{
		"run_id":      runID,
		"session_key": opts.SessionKey,
		"channel":     opts.Channel,
		"chat_id":     opts.ChatID,
	})

	// 0. Record last channel for heartbeat notifications (skip internal channels)
	if opts.Channel != "" && opts.ChatID != "" {
		// Don't record internal channels (cli, system, subagent)
		if !constants.IsInternalChannel(opts.Channel) {
			channelKey := fmt.Sprintf("%s:%s", opts.Channel, opts.ChatID)
			if err := al.RecordLastChannel(channelKey); err != nil {
				logger.WarnCF("agent", "Failed to record last channel: %v", map[string]interface{}{"error": err.Error()})
			}
		}
	}

	// 1. Build messages (skip history for heartbeat)
	var history []providers.Message
	var summary string
	if !opts.NoHistory {
		history = al.sessions.GetHistory(opts.SessionKey)
		summary = al.sessions.GetSummary(opts.SessionKey)
	}
	messages := al.contextBuilder.BuildMessages(
		history,
		summary,
		opts.UserMessage,
		nil,
		opts.Channel,
		opts.ChatID,
	)

	// 2. Save user message to session
	if !opts.NoHistory {
		al.sessions.AddMessage(opts.SessionKey, "user", opts.UserMessage)
	}

	// 3. Run LLM iteration loop
	finalContent, iteration, sentUserViaTool, err := al.runLLMIteration(ctx, messages, opts, runID)
	if err != nil {
		return "", err
	}

	// If last tool had ForUser content and we already sent it, we might not need to send final response
	// This is controlled by the tool's Silent flag and ForUser content

	// 4. Handle empty response
	if finalContent == "" {
		finalContent = opts.DefaultResponse
	}

	// 5. Save final assistant message to session
	if !opts.NoHistory {
		al.sessions.AddMessage(opts.SessionKey, "assistant", finalContent)
		al.sessions.Save(opts.SessionKey)
	}

	// 6. Optional: summarization
	if opts.EnableSummary {
		al.maybeSummarize(opts.SessionKey)
	}

	// 7. Optional: send response via bus
	if opts.SendResponse {
		if ok := al.bus.PublishOutbound(bus.OutboundMessage{
			Channel: opts.Channel,
			ChatID:  opts.ChatID,
			Content: finalContent,
		}); !ok {
			logger.WarnCF("agent", "Failed to publish runAgentLoop response: outbound queue timeout", map[string]interface{}{
				"channel": opts.Channel,
				"chat_id": opts.ChatID,
			})
		}
	}

	// 8. Log response
	responsePreview := utils.Truncate(finalContent, 120)
	logger.InfoCF("agent", fmt.Sprintf("Response: %s", responsePreview),
		map[string]interface{}{
			"run_id":       runID,
			"session_key":  opts.SessionKey,
			"iterations":   iteration,
			"final_length": len(finalContent),
		})

	if sentUserViaTool {
		return "", nil
	}

	return finalContent, nil
}

// runLLMIteration executes the LLM call loop with tool handling.
// Returns the final content, iteration count, and any error.
func (al *AgentLoop) runLLMIteration(ctx context.Context, messages []providers.Message, opts processOptions, runID string) (string, int, bool, error) {
	iteration := 0
	var finalContent string
	sentUserViaTool := false

	for iteration < al.maxIterations {
		iteration++

		logger.DebugCF("agent", "LLM iteration",
			map[string]interface{}{
				"run_id":      runID,
				"session_key": opts.SessionKey,
				"iteration":   iteration,
				"max":         al.maxIterations,
			})

		// Build tool definitions
		providerToolDefs := al.tools.ToProviderDefs()

		// Log LLM request details
		logger.DebugCF("agent", "LLM request",
			map[string]interface{}{
				"run_id":            runID,
				"session_key":       opts.SessionKey,
				"iteration":         iteration,
				"model":             al.model,
				"messages_count":    len(messages),
				"tools_count":       len(providerToolDefs),
				"max_tokens":        al.maxTokens,
				"temperature":       al.temperature,
				"system_prompt_len": len(messages[0].Content),
			})

		// Log full messages (detailed)
		logger.DebugCF("agent", "Full LLM request",
			map[string]interface{}{
				"run_id":        runID,
				"session_key":   opts.SessionKey,
				"iteration":     iteration,
				"messages_json": formatMessagesForLog(messages),
				"tools_json":    formatToolsForLog(providerToolDefs),
			})

		// Call LLM with retry logic for transient errors
		var response *providers.LLMResponse
		var err error
		maxRetries := al.llmMaxRetries
		retryStart := time.Now()
		for retry := 0; retry <= maxRetries; retry++ {
			response, err = al.provider.Chat(ctx, messages, providerToolDefs, al.model, map[string]interface{}{
				"max_tokens":  al.maxTokens,
				"temperature": al.temperature,
			})

			if err == nil {
				break // Success
			}

			// Check if error is retryable (network/transient errors + 5xx server errors)
			errStr := err.Error()
			isRetryable := strings.Contains(errStr, "unexpected EOF") ||
				strings.Contains(errStr, "connection reset") ||
				strings.Contains(errStr, "timeout") ||
				strings.Contains(errStr, "temporary failure") ||
				strings.Contains(errStr, "status=5") ||
				strings.Contains(errStr, "500") ||
				strings.Contains(errStr, "502") ||
				strings.Contains(errStr, "503") ||
				strings.Contains(errStr, "504")

			// Check for rate limit (429) - needs special handling with longer wait
			isRateLimit := strings.Contains(errStr, "429") ||
				strings.Contains(errStr, "rate limit") ||
				strings.Contains(errStr, "too many requests")

			if isRateLimit && retry < maxRetries {
				// Rate limit: linear backoff, bounded by configured cap.
				waitSeconds := al.llmRateLimitBackoff * (retry + 1)
				if waitSeconds > al.llmRateLimitMaxBackoff {
					waitSeconds = al.llmRateLimitMaxBackoff
				}
				waitTime := time.Duration(waitSeconds) * time.Second
				waitTime, canWait := al.clampRetryWait(waitTime, retryStart)
				if !canWait {
					break
				}
				logger.WarnCF("agent", "Rate limited, waiting",
					map[string]interface{}{
						"run_id":       runID,
						"session_key":  opts.SessionKey,
						"iteration":    iteration,
						"retry":        retry + 1,
						"max_retries":  maxRetries,
						"wait_seconds": waitTime.Seconds(),
						"error":        errStr,
					})
				if err := waitWithContext(ctx, waitTime); err != nil {
					return "", iteration, sentUserViaTool, err
				}
				continue
			}

			if isRetryable && retry < maxRetries {
				// Retryable errors: exponential backoff, bounded by configured cap.
				waitSeconds := al.llmRetryBackoffSeconds << retry
				if waitSeconds > al.llmRetryMaxBackoff {
					waitSeconds = al.llmRetryMaxBackoff
				}
				waitTime := time.Duration(waitSeconds) * time.Second
				waitTime, canWait := al.clampRetryWait(waitTime, retryStart)
				if !canWait {
					break
				}
				logger.WarnCF("agent", "LLM call failed, retrying",
					map[string]interface{}{
						"run_id":       runID,
						"session_key":  opts.SessionKey,
						"iteration":    iteration,
						"retry":        retry + 1,
						"max_retries":  maxRetries,
						"wait_seconds": waitTime.Seconds(),
						"error":        errStr,
					})
				if err := waitWithContext(ctx, waitTime); err != nil {
					return "", iteration, sentUserViaTool, err
				}
				continue
			}

			// Non-retryable error or max retries exceeded
			break
		}

		if err != nil {
			logger.ErrorCF("agent", "LLM call failed",
				map[string]interface{}{
					"run_id":      runID,
					"session_key": opts.SessionKey,
					"iteration":   iteration,
					"error":       err.Error(),
				})
			return "", iteration, sentUserViaTool, fmt.Errorf("LLM call failed: %w", err)
		}

		// Check if no tool calls - we're done
		if len(response.ToolCalls) == 0 {
			finalContent = response.Content
			if strings.TrimSpace(finalContent) == "" {
				logger.WarnCF("agent", "LLM returned empty direct response, requesting retry",
					map[string]interface{}{
						"run_id":      runID,
						"session_key": opts.SessionKey,
						"iteration":   iteration,
					})
				if iteration < al.maxIterations {
					messages = append(messages, providers.Message{
						Role: "user",
						Content: "Your previous response was empty. " +
							"Return a concise, non-empty response for the user now.",
					})
					continue
				}
			}
			logger.InfoCF("agent", "LLM response without tool calls (direct answer)",
				map[string]interface{}{
					"run_id":        runID,
					"session_key":   opts.SessionKey,
					"iteration":     iteration,
					"content_chars": len(finalContent),
				})
			break
		}

		// Log tool calls
		toolNames := make([]string, 0, len(response.ToolCalls))
		for _, tc := range response.ToolCalls {
			toolNames = append(toolNames, tc.Name)
		}
		logger.InfoCF("agent", "LLM requested tool calls",
			map[string]interface{}{
				"run_id":      runID,
				"session_key": opts.SessionKey,
				"tools":       toolNames,
				"count":       len(response.ToolCalls),
				"iteration":   iteration,
			})

		// Build assistant message with tool calls
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

		// Save assistant message with tool calls to session
		al.sessions.AddFullMessage(opts.SessionKey, assistantMsg)

		// Execute tool calls
		for _, tc := range response.ToolCalls {
			// Log tool call with arguments preview
			argsJSON, _ := json.Marshal(tc.Arguments)
			argsPreview := utils.Truncate(string(argsJSON), 200)
			logger.InfoCF("agent", fmt.Sprintf("Tool call: %s(%s)", tc.Name, argsPreview),
				map[string]interface{}{
					"run_id":      runID,
					"session_key": opts.SessionKey,
					"tool":        tc.Name,
					"iteration":   iteration,
				})

			// Create async callback for tools that implement AsyncTool
			// NOTE: Following openclaw's design, async tools do NOT send results directly to users.
			// Instead, they notify the agent via PublishInbound, and the agent decides
			// whether to forward the result to the user (in processSystemMessage).
			asyncCallback := func(callbackCtx context.Context, result *tools.ToolResult) {
				// Log the async completion but don't send directly to user
				// The agent will handle user notification via processSystemMessage
				if !result.Silent && result.ForUser != "" {
					logger.InfoCF("agent", "Async tool completed, agent will handle notification",
						map[string]interface{}{
							"run_id":      runID,
							"session_key": opts.SessionKey,
							"tool":        tc.Name,
							"content_len": len(result.ForUser),
						})
				}
			}

			toolResult := al.tools.ExecuteWithContext(ctx, tc.Name, tc.Arguments, opts.Channel, opts.ChatID, asyncCallback)
			if tc.Name == "message" && toolResult.Silent && !toolResult.IsError {
				sentUserViaTool = true
			}

			// Send ForUser content to user immediately if not Silent
			if !toolResult.Silent && toolResult.ForUser != "" && opts.SendResponse {
				if ok := al.bus.PublishOutbound(bus.OutboundMessage{
					Channel: opts.Channel,
					ChatID:  opts.ChatID,
					Content: toolResult.ForUser,
				}); !ok {
					logger.WarnCF("agent", "Failed to publish tool result: outbound queue timeout", map[string]interface{}{
						"run_id":      runID,
						"session_key": opts.SessionKey,
						"tool":        tc.Name,
						"channel":     opts.Channel,
						"chat_id":     opts.ChatID,
					})
				}
				logger.DebugCF("agent", "Sent tool result to user",
					map[string]interface{}{
						"run_id":      runID,
						"session_key": opts.SessionKey,
						"tool":        tc.Name,
						"content_len": len(toolResult.ForUser),
					})
			}

			// Determine content for LLM based on tool result
			contentForLLM := toolResult.ForLLM
			if contentForLLM == "" && toolResult.Err != nil {
				contentForLLM = toolResult.Err.Error()
			}

			toolResultMsg := providers.Message{
				Role:       "tool",
				Content:    contentForLLM,
				ToolCallID: tc.ID,
			}
			messages = append(messages, toolResultMsg)

			// Save tool result message to session
			al.sessions.AddFullMessage(opts.SessionKey, toolResultMsg)
		}
	}

	return finalContent, iteration, sentUserViaTool, nil
}

func (al *AgentLoop) nextRunID(sessionKey string) string {
	n := al.runCounter.Add(1)
	base := strings.TrimSpace(sessionKey)
	if base == "" {
		base = "no-session"
	}
	base = strings.ReplaceAll(base, ":", "_")
	return fmt.Sprintf("%s-%06d", base, n)
}

func (al *AgentLoop) clampRetryWait(wait time.Duration, retryStart time.Time) (time.Duration, bool) {
	if al.llmRetryMaxElapsed <= 0 {
		return wait, true
	}

	maxElapsed := time.Duration(al.llmRetryMaxElapsed) * time.Second
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

func maxInt(v, min int) int {
	if v < min {
		return min
	}
	return v
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

// maybeSummarize triggers summarization if the session history exceeds thresholds.
func (al *AgentLoop) maybeSummarize(sessionKey string) {
	newHistory := al.sessions.GetHistory(sessionKey)
	tokenEstimate := al.estimateTokens(newHistory)

	threshold := int(float64(al.contextWindow) * al.memoryThreshold)

	if len(newHistory) > al.historyMessageThreshold || tokenEstimate > threshold {
		if _, loading := al.summarizing.LoadOrStore(sessionKey, true); !loading {
			go func() {
				defer func() {
					if r := recover(); r != nil {
						logger.ErrorCF("agent", "Recovered from panic in summarization",
							map[string]interface{}{
								"panic":       fmt.Sprintf("%v", r),
								"session_key": sessionKey,
							})
					}
					al.summarizing.Delete(sessionKey)
				}()
				al.summarizeSession(sessionKey)
			}()
		}
	}
}

// GetStartupInfo returns information about loaded tools and skills for logging.
func (al *AgentLoop) GetStartupInfo() map[string]interface{} {
	return al.getInfo(true)
}

// GetRuntimeInfo returns current runtime info as known by the main agent prompt state.
// Named agents come from ContextBuilder cache (same source used in system prompt).
func (al *AgentLoop) GetRuntimeInfo() map[string]interface{} {
	return al.getInfo(false)
}

func (al *AgentLoop) getInfo(refreshAgents bool) map[string]interface{} {
	info := make(map[string]interface{})

	// Tools info
	toolList := al.tools.List()
	info["tools"] = map[string]interface{}{
		"count": len(toolList),
		"names": toolList,
	}

	// Skills info
	info["skills"] = al.contextBuilder.GetSkillsInfo()

	// Named agents info (shared with prompt builder cache).
	info["agents"] = al.contextBuilder.GetNamedAgentsInfo(refreshAgents)

	return info
}

// formatMessagesForLog formats messages for logging
func formatMessagesForLog(messages []providers.Message) string {
	if len(messages) == 0 {
		return "[]"
	}

	var result string
	result += "[\n"
	for i, msg := range messages {
		result += fmt.Sprintf("  [%d] Role: %s\n", i, msg.Role)
		if msg.ToolCalls != nil && len(msg.ToolCalls) > 0 {
			result += "  ToolCalls:\n"
			for _, tc := range msg.ToolCalls {
				result += fmt.Sprintf("    - ID: %s, Type: %s, Name: %s\n", tc.ID, tc.Type, tc.Name)
				if tc.Function != nil {
					result += fmt.Sprintf("      Arguments: %s\n", utils.Truncate(tc.Function.Arguments, 200))
				}
			}
		}
		if msg.Content != "" {
			content := utils.Truncate(msg.Content, 200)
			result += fmt.Sprintf("  Content: %s\n", content)
		}
		if msg.ToolCallID != "" {
			result += fmt.Sprintf("  ToolCallID: %s\n", msg.ToolCallID)
		}
		result += "\n"
	}
	result += "]"
	return result
}

// formatToolsForLog formats tool definitions for logging
func formatToolsForLog(tools []providers.ToolDefinition) string {
	if len(tools) == 0 {
		return "[]"
	}

	var result string
	result += "[\n"
	for i, tool := range tools {
		result += fmt.Sprintf("  [%d] Type: %s, Name: %s\n", i, tool.Type, tool.Function.Name)
		result += fmt.Sprintf("      Description: %s\n", tool.Function.Description)
		if len(tool.Function.Parameters) > 0 {
			result += fmt.Sprintf("      Parameters: %s\n", utils.Truncate(fmt.Sprintf("%v", tool.Function.Parameters), 200))
		}
	}
	result += "]"
	return result
}

// summarizeSession summarizes the conversation history for a session.
func (al *AgentLoop) summarizeSession(sessionKey string) {
	defer func() {
		if r := recover(); r != nil {
			logger.ErrorCF("agent", "Recovered from panic in summarizeSession",
				map[string]interface{}{
					"panic":       fmt.Sprintf("%v", r),
					"session_key": sessionKey,
				})
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(al.llmTimeout)*time.Second)
	defer cancel()

	history := al.sessions.GetHistory(sessionKey)
	summary := al.sessions.GetSummary(sessionKey)

	// Keep last 4 messages for continuity
	if len(history) <= 4 {
		return
	}

	toSummarize := history[:len(history)-4]

	// Oversized Message Guard
	// Skip messages larger than 50% of context window to prevent summarizer overflow
	maxMessageTokens := al.contextWindow / 2
	validMessages := make([]providers.Message, 0)
	omitted := false

	for _, m := range toSummarize {
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		// Estimate tokens for this message
		msgTokens := len(m.Content) / 4
		if msgTokens > maxMessageTokens {
			omitted = true
			continue
		}
		validMessages = append(validMessages, m)
	}

	if len(validMessages) == 0 {
		return
	}

	// Multi-Part Summarization
	// Split into two parts if history is significant
	var finalSummary string
	var err error
	if len(validMessages) > 10 {
		mid := len(validMessages) / 2
		part1 := validMessages[:mid]
		part2 := validMessages[mid:]

		s1, _ := al.summarizeBatch(ctx, part1, "")
		s2, _ := al.summarizeBatch(ctx, part2, "")

		// Merge them
		mergePrompt := fmt.Sprintf("Merge these two conversation summaries into one cohesive summary:\n\n1: %s\n\n2: %s", s1, s2)
		var resp *providers.LLMResponse
		resp, err = al.provider.Chat(ctx, []providers.Message{{Role: "user", Content: mergePrompt}}, nil, al.model, map[string]interface{}{
			"max_tokens":  1024,
			"temperature": 0.3,
		})
		if err == nil {
			finalSummary = resp.Content
		} else {
			finalSummary = s1 + " " + s2
			logger.WarnCF("agent", "Failed to merge summaries, using concatenation",
				map[string]interface{}{
					"session_key": sessionKey,
					"error":       err.Error(),
				})
		}
	} else {
		finalSummary, err = al.summarizeBatch(ctx, validMessages, summary)
		if err != nil {
			logger.WarnCF("agent", "Failed to summarize batch",
				map[string]interface{}{
					"session_key": sessionKey,
					"error":       err.Error(),
				})
			return
		}
	}

	if omitted && finalSummary != "" {
		finalSummary += "\n[Note: Some oversized messages were omitted from this summary for efficiency.]"
	}

	if finalSummary != "" {
		al.sessions.SetSummary(sessionKey, finalSummary)
		al.sessions.TruncateHistory(sessionKey, 4)
		al.sessions.Save(sessionKey)
	}
}

// summarizeBatch summarizes a batch of messages.
func (al *AgentLoop) summarizeBatch(ctx context.Context, batch []providers.Message, existingSummary string) (string, error) {
	prompt := "Provide a concise summary of this conversation segment, preserving core context and key points.\n"
	if existingSummary != "" {
		prompt += "Existing context: " + existingSummary + "\n"
	}
	prompt += "\nCONVERSATION:\n"
	for _, m := range batch {
		prompt += fmt.Sprintf("%s: %s\n", m.Role, m.Content)
	}

	response, err := al.provider.Chat(ctx, []providers.Message{{Role: "user", Content: prompt}}, nil, al.model, map[string]interface{}{
		"max_tokens":  1024,
		"temperature": 0.3,
	})
	if err != nil {
		return "", err
	}
	return response.Content, nil
}

// estimateTokens estimates the number of tokens in a message list.
// Uses rune count instead of byte length so that CJK and other multi-byte
// characters are not over-counted (a Chinese character is 3 bytes but roughly
// one token).
func (al *AgentLoop) estimateTokens(messages []providers.Message) int {
	total := 0
	for _, m := range messages {
		total += utf8.RuneCountInString(m.Content) / 3
	}
	return total
}
