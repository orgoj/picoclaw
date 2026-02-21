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

	"github.com/sipeed/picoclaw/pkg/audit"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/constants"
	"github.com/sipeed/picoclaw/pkg/llm"
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
	maxIterations           int     // Max LLM loop iterations for main agent
	maxToolIterations       int     // Max iterations that include tool calls for main agent
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
	summaryKeepLastMessages int     // Number of recent messages to keep after summarization
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
	idleMu                  sync.Mutex
	idleStreakCount         int
	idleSince               time.Time
	lastUserMessageAt       time.Time
	runCounter              atomic.Uint64
	urgentMu                sync.Mutex
	urgentBySession         map[string][]string
	activeRunsBySession     map[string]int
	runCancelBySession      map[string]context.CancelFunc
}

type idleMetrics struct {
	StreakCount      int
	IdleSince        time.Time
	LastUserMessage  time.Time
	Now              time.Time
	SecondsSinceUser int64
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
	denyPathPatterns := cfg.Agents.Defaults.DenyPathPatterns

	// File system tools
	registry.Register(tools.NewReadFileTool(workspace, restrict, denyPathPatterns...))
	registry.Register(tools.NewWriteFileTool(workspace, restrict, denyPathPatterns...))
	registry.Register(tools.NewListDirTool(workspace, restrict, denyPathPatterns...))
	registry.Register(tools.NewEditFileTool(workspace, restrict, denyPathPatterns...))
	registry.Register(tools.NewAppendFileTool(workspace, restrict, denyPathPatterns...))

	// Shell execution
	registry.Register(tools.NewExecTool(workspace, restrict, denyPathPatterns...))

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
		maxIterations:           cfg.Agents.Defaults.MaxIterations,
		maxToolIterations:       cfg.Agents.Defaults.MaxToolIterations,
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
		summaryKeepLastMessages: maxInt(cfg.Agents.Defaults.SummaryKeepLastMessages, 1),
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
		urgentBySession:     make(map[string][]string),
		activeRunsBySession: make(map[string]int),
		runCancelBySession:  make(map[string]context.CancelFunc),
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
						al.triggerIdle(ctx, al.markIdleTriggered())
					}
					continue
				}
			} else {
				msg, ok = al.bus.ConsumeInbound(ctx)
			}

			if !ok {
				continue
			}

			// Got an external user message — reset idle state/metrics.
			if !constants.IsInternalChannel(msg.Channel) {
				idleTriggered = false
				al.resetIdleTracking()
			}

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
func (al *AgentLoop) triggerIdle(ctx context.Context, m idleMetrics) {
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

	// LastChannel stores "platform:chatID" format.
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
		"channel":            channel,
		"chat_id":            chatID,
		"idle_streak_count":  m.StreakCount,
		"seconds_since_user": m.SecondsSinceUser,
	})

	now := time.Now().Format("2006-01-02 15:04:05")

	// Build subagent status context
	subagentStatus := al.buildSubagentStatus()

	idleCtx := fmt.Sprintf(
		"<idle_context>\nidle_streak_count=%d\nidle_since=%s\nlast_user_message_at=%s\nseconds_since_user_message=%d\n</idle_context>",
		m.StreakCount,
		m.IdleSince.Format(time.RFC3339),
		m.LastUserMessage.Format(time.RFC3339),
		m.SecondsSinceUser,
	)

	idleProtocol := content
	idleMessage := fmt.Sprintf(
		"<idle_message source=\"idle_timer\">\n<current_time>%s</current_time>\n%s\n\n<idle_protocol>\n%s\n</idle_protocol>\n",
		now,
		idleCtx,
		idleProtocol,
	)
	if subagentStatus != "" {
		idleMessage += fmt.Sprintf("\n<subagent_status>\n%s\n</subagent_status>\n", subagentStatus)
	}
	idleMessage += "</idle_message>\nTreat this as a normal message in the main session. You are in IDLE mode."

	// Route IDLE through the same live session and same processing path as normal message.
	sessionKey := fmt.Sprintf("%s:%s", channel, chatID)
	response, err := al.processMessage(ctx, bus.InboundMessage{
		Channel:    channel,
		SenderID:   "idle",
		ChatID:     chatID,
		SessionKey: sessionKey,
		Content:    idleMessage,
		Metadata: map[string]string{
			"idle":   "true",
			"source": "idle_timer",
		},
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

func (al *AgentLoop) markIdleTriggered() idleMetrics {
	al.idleMu.Lock()
	defer al.idleMu.Unlock()

	now := time.Now()
	if al.idleSince.IsZero() {
		al.idleSince = now
	}
	al.idleStreakCount++

	last := al.lastUserMessageAt
	if last.IsZero() {
		// Fallback for early startup before first user message.
		last = now.Add(-al.idleTimeout)
	}
	seconds := int64(now.Sub(last).Seconds())
	if seconds < 0 {
		seconds = 0
	}

	return idleMetrics{
		StreakCount:      al.idleStreakCount,
		IdleSince:        al.idleSince,
		LastUserMessage:  last,
		Now:              now,
		SecondsSinceUser: seconds,
	}
}

func (al *AgentLoop) resetIdleTracking() {
	al.idleMu.Lock()
	defer al.idleMu.Unlock()
	al.idleStreakCount = 0
	al.idleSince = time.Time{}
	al.lastUserMessageAt = time.Now()
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

// GetSessionHistory returns a copy of the current session message history.
func (al *AgentLoop) GetSessionHistory(sessionKey string) []providers.Message {
	history := al.sessions.GetHistory(sessionKey)
	out := make([]providers.Message, len(history))
	copy(out, history)
	return out
}

// ListSessions returns session summaries sorted by last update descending.
func (al *AgentLoop) ListSessions(limit int) []session.SessionSummary {
	if al.sessions == nil {
		return nil
	}
	return al.sessions.ListSummaries(limit)
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

// ProcessImmediate bypasses inbound queue consumption and processes a message now.
// This is intended for operator-level urgent commands that must not wait in queue.
func (al *AgentLoop) ProcessImmediate(ctx context.Context, channel, chatID, senderID, sessionKey, content string) (string, error) {
	channel = strings.TrimSpace(channel)
	chatID = strings.TrimSpace(chatID)
	senderID = strings.TrimSpace(senderID)
	sessionKey = strings.TrimSpace(sessionKey)
	content = strings.TrimSpace(content)

	if channel == "" {
		channel = "cli"
	}
	if chatID == "" {
		chatID = "direct"
	}
	if senderID == "" {
		senderID = "urgent"
	}
	if sessionKey == "" {
		sessionKey = fmt.Sprintf("%s:%s", channel, chatID)
	}

	msg := bus.InboundMessage{
		Channel:    channel,
		SenderID:   senderID,
		ChatID:     chatID,
		Content:    content,
		SessionKey: sessionKey,
		Metadata: map[string]string{
			"source": "direct-immediate",
			"urgent": "true",
		},
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
	audit.Record("agent_process_message", map[string]interface{}{
		"channel":     msg.Channel,
		"chat_id":     msg.ChatID,
		"sender_id":   msg.SenderID,
		"session_key": msg.SessionKey,
	})

	// Route system messages to processSystemMessage
	if msg.Channel == "system" {
		return al.processSystemMessage(ctx, msg)
	}

	userMessage := msg.Content
	if extra := buildTimingContextNote(msg.Metadata); extra != "" {
		userMessage = extra + "\n\n" + userMessage
	}

	// Process as user message
	return al.runAgentLoop(ctx, processOptions{
		SessionKey:      msg.SessionKey,
		Channel:         msg.Channel,
		ChatID:          msg.ChatID,
		UserMessage:     userMessage,
		DefaultResponse: "I've completed processing but have no response to give.",
		EnableSummary:   true,
		SendResponse:    false,
	})
}

func buildTimingContextNote(meta map[string]string) string {
	if len(meta) == 0 {
		return ""
	}

	parts := make([]string, 0, 4)
	if delta := strings.TrimSpace(meta["delta_since_prev_ms"]); delta != "" {
		parts = append(parts, "delta_since_prev_ms="+delta)
	}
	if queueLen := strings.TrimSpace(meta["queue_len_at_enqueue"]); queueLen != "" {
		parts = append(parts, "queue_len_at_enqueue="+queueLen)
	}
	if recv := strings.TrimSpace(meta["received_at"]); recv != "" {
		parts = append(parts, "received_at="+recv)
	}
	if enq := strings.TrimSpace(meta["enqueued_at"]); enq != "" {
		parts = append(parts, "enqueued_at="+enq)
	}
	if len(parts) == 0 && !strings.EqualFold(strings.TrimSpace(meta["gap_notice"]), "true") {
		return ""
	}

	prefix := "<timing_context>"
	if strings.EqualFold(strings.TrimSpace(meta["gap_notice"]), "true") {
		prefix = "<timing_context gap_notice=\"true\">"
	}
	return prefix + "\n" + strings.Join(parts, "\n") + "\n</timing_context>"
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

	// Parse origin channel/chat from chat_id (format: "channel:chat_id")
	var originChannel, originChatID string
	if idx := strings.Index(msg.ChatID, ":"); idx > 0 {
		originChannel = msg.ChatID[:idx]
		originChatID = msg.ChatID[idx+1:]
	} else {
		// Fallback
		originChannel = "cli"
		originChatID = "direct"
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

	logger.InfoCF("agent", "Subagent completed",
		map[string]interface{}{
			"sender_id":   msg.SenderID,
			"channel":     originChannel,
			"chat_id":     originChatID,
			"content_len": len(content),
			"directory":   directory,
		})

	// Deliver completion to main agent session (not directly to user channel).
	notification := fmt.Sprintf("📢 Subagent %s completed:\n\n%s", msg.SenderID, content)
	sessionKey := fmt.Sprintf("%s:%s", originChannel, originChatID)

	// Always persist in session history so completion is never dropped.
	al.sessions.AddMessage(sessionKey, "system", notification)
	al.sessions.Save(sessionKey)

	// Always queue urgent completion context for main-agent session.
	// If a run is active, completion context is appended and picked up in-run.
	if al.InjectUrgent(sessionKey, notification) {
		audit.Record("subagent_completion_injected_active_run", map[string]interface{}{
			"session_key": sessionKey,
			"sender_id":   msg.SenderID,
		})
	} else {
		audit.Record("subagent_completion_queued_next_run", map[string]interface{}{
			"session_key": sessionKey,
			"sender_id":   msg.SenderID,
		})
	}
	return "", nil
}

// runAgentLoop is the core message processing logic.
// It handles context building, LLM calls, tool execution, and response handling.
func (al *AgentLoop) runAgentLoop(ctx context.Context, opts processOptions) (string, error) {
	runID := al.nextRunID(opts.SessionKey)
	audit.Record("agent_run_start", map[string]interface{}{
		"run_id":      runID,
		"session_key": opts.SessionKey,
		"channel":     opts.Channel,
		"chat_id":     opts.ChatID,
	})
	al.markSessionRunStart(opts.SessionKey)
	defer al.markSessionRunEnd(opts.SessionKey)
	runCtx, runCancel := context.WithCancel(ctx)
	al.markSessionRunCancel(opts.SessionKey, runCancel)
	defer al.markSessionRunCancelDone(opts.SessionKey)
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
	finalContent, iteration, sentUserViaTool, err := al.runLLMIteration(runCtx, messages, opts, runID)
	if err != nil {
		audit.Record("agent_run_error", map[string]interface{}{
			"run_id":      runID,
			"session_key": opts.SessionKey,
			"error":       err.Error(),
		})
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
		audit.Record("agent_run_done", map[string]interface{}{
			"run_id":        runID,
			"session_key":   opts.SessionKey,
			"iterations":    iteration,
			"sent_via_tool": true,
		})
		return "", nil
	}
	audit.Record("agent_run_done", map[string]interface{}{
		"run_id":       runID,
		"session_key":  opts.SessionKey,
		"iterations":   iteration,
		"response_len": len(finalContent),
	})

	return finalContent, nil
}

// runLLMIteration executes the LLM call loop with tool handling.
// Returns the final content, iteration count, and any error.
func (al *AgentLoop) runLLMIteration(ctx context.Context, messages []providers.Message, opts processOptions, runID string) (string, int, bool, error) {
	iteration := 0
	toolIteration := 0
	var finalContent string
	sentUserViaTool := false

	for iteration < al.maxIterations {
		iteration++

		if urgent := al.drainUrgentMessages(opts.SessionKey); len(urgent) > 0 {
			for _, u := range urgent {
				messages = append(messages, providers.Message{
					Role:    "user",
					Content: u,
				})
			}
			logger.WarnCF("agent", "Injected urgent message(s) into active run", map[string]interface{}{
				"run_id":       runID,
				"session_key":  opts.SessionKey,
				"iteration":    iteration,
				"urgent_count": len(urgent),
			})
		}

		// Build tool definitions
		providerToolDefs := al.tools.ToProviderDefs()
		tokenEstimate := al.estimateTokens(messages)
		contextRemaining := 0
		if al.contextWindow > tokenEstimate {
			contextRemaining = al.contextWindow - tokenEstimate
		}
		contextState := fmt.Sprintf("%d/%d", tokenEstimate, al.contextWindow)
		logger.DebugCF("agent", "LLM iteration",
			map[string]interface{}{
				"run_id":            runID,
				"session_key":       opts.SessionKey,
				"iteration":         iteration,
				"max":               al.maxIterations,
				"context_state":     contextState,
				"context_tokens":    tokenEstimate,
				"context_window":    al.contextWindow,
				"context_remaining": contextRemaining,
			})

		// Keep full payload in audit JSON for observability/forensics.
		// Console/file logs stay concise to avoid runtime noise.
		audit.Record("llm_request_full", map[string]interface{}{
			"run_id":        runID,
			"session_key":   opts.SessionKey,
			"iteration":     iteration,
			"messages_json": formatMessagesForLog(messages),
			"tools_json":    formatToolsForLog(providerToolDefs),
		})

		response, err := llm.CallWithRetry(ctx, llm.CallConfig{
			Provider: al.provider,
			Messages: messages,
			Tools:    providerToolDefs,
			Model:    al.model,
			Options: map[string]any{
				"max_tokens":  al.maxTokens,
				"temperature": al.temperature,
			},
			Retry: llm.RetryConfig{
				MaxRetries:              al.llmMaxRetries,
				RetryBackoffSeconds:     al.llmRetryBackoffSeconds,
				RetryMaxBackoffSeconds:  al.llmRetryMaxBackoff,
				RateLimitBackoffSeconds: al.llmRateLimitBackoff,
				RateLimitMaxBackoffSecs: al.llmRateLimitMaxBackoff,
				RetryMaxElapsedSeconds:  al.llmRetryMaxElapsed,
			},
			LogComponent: "agent",
			LogFields: map[string]any{
				"run_id":      runID,
				"session_key": opts.SessionKey,
				"iteration":   iteration,
			},
		})

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
			// If urgent content arrived while this direct response was being generated,
			// keep iterating so the urgent instruction is handled in the same run.
			if al.hasUrgentMessages(opts.SessionKey) && iteration < al.maxIterations {
				messages = append(messages, providers.Message{
					Role:    "assistant",
					Content: finalContent,
				})
				logger.WarnCF("agent", "Urgent message pending, continuing iteration in active run", map[string]interface{}{
					"run_id":      runID,
					"session_key": opts.SessionKey,
					"iteration":   iteration,
				})
				continue
			}
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
				"run_id":         runID,
				"session_key":    opts.SessionKey,
				"tools":          toolNames,
				"count":          len(response.ToolCalls),
				"iteration":      iteration,
				"tool_iteration": toolIteration + 1,
			})
		toolIteration++
		if al.maxToolIterations > 0 && toolIteration > al.maxToolIterations {
			return "", iteration, sentUserViaTool, fmt.Errorf("agent loop reached max tool iterations (%d) without final response", al.maxToolIterations)
		}

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

// InjectUrgent queues urgent content for a session.
// Returns true when a run is currently active for the session.
func (al *AgentLoop) InjectUrgent(sessionKey, content string) bool {
	sessionKey = strings.TrimSpace(sessionKey)
	content = strings.TrimSpace(content)
	if sessionKey == "" || content == "" {
		return false
	}

	al.urgentMu.Lock()
	defer al.urgentMu.Unlock()

	al.urgentBySession[sessionKey] = append(al.urgentBySession[sessionKey], content)
	return al.activeRunsBySession[sessionKey] > 0
}

func (al *AgentLoop) drainUrgentMessages(sessionKey string) []string {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return nil
	}
	al.urgentMu.Lock()
	defer al.urgentMu.Unlock()
	items := al.urgentBySession[sessionKey]
	if len(items) == 0 {
		return nil
	}
	delete(al.urgentBySession, sessionKey)
	return items
}

func (al *AgentLoop) markSessionRunStart(sessionKey string) {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return
	}
	al.urgentMu.Lock()
	defer al.urgentMu.Unlock()
	al.activeRunsBySession[sessionKey]++
}

func (al *AgentLoop) markSessionRunEnd(sessionKey string) {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return
	}
	al.urgentMu.Lock()
	defer al.urgentMu.Unlock()
	n := al.activeRunsBySession[sessionKey]
	if n <= 1 {
		delete(al.activeRunsBySession, sessionKey)
		return
	}
	al.activeRunsBySession[sessionKey] = n - 1
}

func (al *AgentLoop) markSessionRunCancel(sessionKey string, cancel context.CancelFunc) {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" || cancel == nil {
		return
	}
	al.urgentMu.Lock()
	defer al.urgentMu.Unlock()
	al.runCancelBySession[sessionKey] = cancel
}

func (al *AgentLoop) markSessionRunCancelDone(sessionKey string) {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return
	}
	al.urgentMu.Lock()
	defer al.urgentMu.Unlock()
	delete(al.runCancelBySession, sessionKey)
}

func (al *AgentLoop) hasUrgentMessages(sessionKey string) bool {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return false
	}
	al.urgentMu.Lock()
	defer al.urgentMu.Unlock()
	return len(al.urgentBySession[sessionKey]) > 0
}

func maxInt(v, min int) int {
	if v < min {
		return min
	}
	return v
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

	keepLast := maxInt(al.summaryKeepLastMessages, 1)

	// Keep last N messages for continuity
	if len(history) <= keepLast {
		return
	}

	toSummarize := history[:len(history)-keepLast]

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
		resp, err = llm.CallWithRetry(ctx, llm.CallConfig{
			Provider: al.provider,
			Messages: []providers.Message{{Role: "user", Content: mergePrompt}},
			Model:    al.model,
			Options: map[string]any{
				"max_tokens":  1024,
				"temperature": 0.3,
			},
			Retry: llm.RetryConfig{
				MaxRetries:              al.llmMaxRetries,
				RetryBackoffSeconds:     al.llmRetryBackoffSeconds,
				RetryMaxBackoffSeconds:  al.llmRetryMaxBackoff,
				RateLimitBackoffSeconds: al.llmRateLimitBackoff,
				RateLimitMaxBackoffSecs: al.llmRateLimitMaxBackoff,
				RetryMaxElapsedSeconds:  al.llmRetryMaxElapsed,
			},
			LogComponent: "agent",
			LogFields: map[string]any{
				"session_key": sessionKey,
				"phase":       "summary_merge",
			},
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
		al.sessions.TruncateHistory(sessionKey, keepLast)
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

	response, err := llm.CallWithRetry(ctx, llm.CallConfig{
		Provider: al.provider,
		Messages: []providers.Message{{Role: "user", Content: prompt}},
		Model:    al.model,
		Options: map[string]any{
			"max_tokens":  1024,
			"temperature": 0.3,
		},
		Retry: llm.RetryConfig{
			MaxRetries:              al.llmMaxRetries,
			RetryBackoffSeconds:     al.llmRetryBackoffSeconds,
			RetryMaxBackoffSeconds:  al.llmRetryMaxBackoff,
			RateLimitBackoffSeconds: al.llmRateLimitBackoff,
			RateLimitMaxBackoffSecs: al.llmRateLimitMaxBackoff,
			RetryMaxElapsedSeconds:  al.llmRetryMaxElapsed,
		},
		LogComponent: "agent",
		LogFields: map[string]any{
			"phase": "summary_batch",
		},
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
