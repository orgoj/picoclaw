package channels

import (
	"context"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/audit"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/sipeed/picoclaw/pkg/version"
)

type TelegramCommander interface {
	Help(ctx context.Context, message telego.Message) error
	Start(ctx context.Context, message telego.Message) error
	Status(ctx context.Context, message telego.Message) error
	Kill(ctx context.Context, message telego.Message) error
	Inject(ctx context.Context, message telego.Message) error
	First(ctx context.Context, message telego.Message) error
	Urgent(ctx context.Context, message telego.Message) error
	Models(ctx context.Context, message telego.Message) error
	Channels(ctx context.Context, message telego.Message) error
}

type cmd struct {
	bot             *telego.Bot
	bus             *bus.MessageBus
	config          *config.Config
	subagentManager *tools.SubagentManager
	agentLoop       *agent.AgentLoop
}

func NewTelegramCommands(bot *telego.Bot, msgBus *bus.MessageBus, cfg *config.Config, subagentManager *tools.SubagentManager, agentLoop *agent.AgentLoop) TelegramCommander {
	return &cmd{
		bot:             bot,
		bus:             msgBus,
		config:          cfg,
		subagentManager: subagentManager,
		agentLoop:       agentLoop,
	}
}

func (c *cmd) Help(ctx context.Context, message telego.Message) error {
	prefix := "+"
	if c.config != nil {
		p := strings.TrimSpace(c.config.Ingress.ConcatPrefix)
		if p != "" {
			prefix = string([]rune(p)[0])
		}
	}
	lines := []string{
		"🦞 <b>PicoClaw Commands</b>",
		"",
		"/start - Start the bot",
		"/help - Show this help message",
		"/status - Show system and subagents status",
		"/models - List available models",
		"/channels - List enabled channels",
		"",
		fmt.Sprintf("%sinject MESSAGE - Immediate priority inject (all channels)", prefix),
		fmt.Sprintf("%sfirst MESSAGE - Put message at inbound queue head (all channels)", prefix),
	}
	if c.config != nil && c.config.Tools.Spawn.Enabled {
		lines = append(lines, fmt.Sprintf("%skill TASK_ID - Cancel running subagent task (all channels)", prefix))
	}
	msg := strings.Join(lines, "\n")
	_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: message.Chat.ID},
		Text:      msg,
		ParseMode: telego.ModeHTML,
		ReplyParameters: &telego.ReplyParameters{
			MessageID: message.MessageID,
		},
	})
	return err
}

func (c *cmd) Start(ctx context.Context, message telego.Message) error {
	_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: message.Chat.ID},
		Text:   "Hello! I am PicoClaw 🦞",
		ReplyParameters: &telego.ReplyParameters{
			MessageID: message.MessageID,
		},
	})
	return err
}

func (c *cmd) Models(ctx context.Context, message telego.Message) error {
	mainModel := c.config.Agents.Defaults.Model
	mainProvider := resolvedProviderName(c.config, mainModel)
	subagentModel := c.config.Agents.Defaults.Model
	subagentProvider := resolvedProviderName(c.config, subagentModel)

	response := fmt.Sprintf(
		"🦞 <b>Configured Models</b>\n\n"+
			"<b>Main Agent</b>\n"+
			"- Model: <code>%s</code>\n"+
			"- Provider: <code>%s</code>\n\n"+
			"<b>Subagents</b>\n"+
			"- Model: <code>%s</code> (inherits <code>agents.defaults.model</code>)\n"+
			"- Provider: <code>%s</code>",
		html.EscapeString(mainModel),
		html.EscapeString(mainProvider),
		html.EscapeString(subagentModel),
		html.EscapeString(subagentProvider),
	)

	_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: message.Chat.ID},
		Text:      response,
		ParseMode: telego.ModeHTML,
		ReplyParameters: &telego.ReplyParameters{
			MessageID: message.MessageID,
		},
	})
	return err
}

func (c *cmd) Channels(ctx context.Context, message telego.Message) error {
	var channels []string
	addChan := func(name string, enabled bool) {
		status := ""
		if enabled {
			status = " *"
		}
		channels = append(channels, fmt.Sprintf("- %s%s", name, status))
	}

	addChan("telegram", c.config.Channels.Telegram.Enabled)
	addChan("whatsapp", c.config.Channels.WhatsApp.Enabled)
	addChan("feishu", c.config.Channels.Feishu.Enabled)
	addChan("discord", c.config.Channels.Discord.Enabled)
	addChan("slack", c.config.Channels.Slack.Enabled)
	addChan("maixcam", c.config.Channels.MaixCam.Enabled)
	addChan("qq", c.config.Channels.QQ.Enabled)
	addChan("dingtalk", c.config.Channels.DingTalk.Enabled)
	addChan("line", c.config.Channels.LINE.Enabled)
	addChan("onebot", c.config.Channels.OneBot.Enabled)

	response := fmt.Sprintf("🦞 <b>Available Channels</b>\n\n%s\n\n* = active", html.EscapeString(strings.Join(channels, "\n")))

	_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: message.Chat.ID},
		Text:      response,
		ParseMode: telego.ModeHTML,
		ReplyParameters: &telego.ReplyParameters{
			MessageID: message.MessageID,
		},
	})
	return err
}

func (c *cmd) Status(ctx context.Context, message telego.Message) error {
	if c.subagentManager == nil {
		_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   "Subagent manager is not initialized.",
		})
		return err
	}

	// Get enhanced task information
	running := c.subagentManager.GetRunningTasks()
	recent := c.subagentManager.GetRecentTasks(5)
	queueCount := c.subagentManager.GetMessageQueueCount()

	var sb strings.Builder
	sb.WriteString("📊 <b>PicoClaw Status</b>\n\n")
	subagentEnabled := c.config != nil && (c.config.Tools.Spawn.Enabled || c.config.Tools.Subagent.Enabled)

	// Version info
	sb.WriteString(fmt.Sprintf("📦 <b>Version:</b> <code>%s</code>\n", html.EscapeString(version.Format())))
	sb.WriteString(fmt.Sprintf("🔧 <b>Go:</b> <code>%s</code>\n\n", html.EscapeString(version.GetGoVersion())))

	// Capabilities info (tools, skills, named agents)
	if c.agentLoop != nil {
		startupInfo := c.agentLoop.GetRuntimeInfo()
		toolsInfo := startupInfo["tools"].(map[string]interface{})
		skillsInfo := startupInfo["skills"].(map[string]interface{})
		agentsInfo := startupInfo["agents"].(map[string]interface{})

		sb.WriteString("🧰 <b>Capabilities:</b>\n")
		sb.WriteString(fmt.Sprintf("  Tools: %d\n", toolsInfo["count"]))
		sb.WriteString(fmt.Sprintf("  Skills: %d/%d\n", skillsInfo["available"], skillsInfo["total"]))
		sb.WriteString(fmt.Sprintf("  Named Agents: %d\n", agentsInfo["count"]))
		sb.WriteString("\n")
	}

	if subagentEnabled {
		// Running subagents with enhanced details
		sb.WriteString("🔄 <b>Running Subagents:</b>\n")
		if len(running) == 0 {
			sb.WriteString("  None\n")
		} else {
			for _, t := range running {
				// Format time range
				startTime := formatTimeShort(t.Started)
				timeRange := fmt.Sprintf("[%s - ...]", startTime)

				sb.WriteString(fmt.Sprintf("• <code>%s</code> %s\n", html.EscapeString(t.ID), html.EscapeString(timeRange)))

				// Label (if set)
				if t.Label != "" {
					sb.WriteString(fmt.Sprintf("  Label: %s\n", html.EscapeString(t.Label)))
				}
				if t.Name != "" {
					sb.WriteString(fmt.Sprintf("  Agent: %s\n", html.EscapeString(t.Name)))
				}

				// Task preview (first 30 chars)
				taskPreview := html.EscapeString(truncateStr(t.Task, 30))
				sb.WriteString(fmt.Sprintf("  Task: %s\n", taskPreview))

				// Pending messages count for this task
				if len(t.PendingMsgs) > 0 {
					sb.WriteString(fmt.Sprintf("  📬 Pending: %d message(s)\n", len(t.PendingMsgs)))
				}
				sb.WriteString("\n")
			}
		}

		// Recent completed tasks
		sb.WriteString(fmt.Sprintf("\n✅ <b>Recent (%d):</b>\n", len(recent)))
		if len(recent) == 0 {
			sb.WriteString("  None\n")
		} else {
			for _, t := range recent {
				startTime := formatTimeShort(t.Started)
				endTime := formatTimeShort(t.Ended)

				// Status icon
				statusIcon := "✅"
				if t.Status == "failed" {
					statusIcon = "❌"
				} else if t.Status == "cancelled" {
					statusIcon = "🚫"
				}

				timeRange := fmt.Sprintf("[%s - %s]", startTime, endTime)
				sb.WriteString(fmt.Sprintf("• <code>%s</code> %s %s\n", html.EscapeString(t.ID), html.EscapeString(timeRange), statusIcon))

				if t.Label != "" {
					sb.WriteString(fmt.Sprintf("  Label: %s\n", html.EscapeString(t.Label)))
				}
				if t.Name != "" {
					sb.WriteString(fmt.Sprintf("  Agent: %s\n", html.EscapeString(t.Name)))
				}
			}
		}

		// Message queue info
		sb.WriteString(fmt.Sprintf("\n📬 <b>Message Queue:</b> %d\n", queueCount))
		if queueCount > 0 {
			firstMsg := c.subagentManager.GetFirstQueuedMessage()
			if firstMsg != "" {
				sb.WriteString(fmt.Sprintf("  Next: \"%s\"\n", html.EscapeString(truncateStr(firstMsg, 80))))
			}
		}
	}

	// Main agent session stats
	sessionKey := fmt.Sprintf("telegram:%d", message.Chat.ID)
	if c.agentLoop != nil {
		stats := c.agentLoop.GetSessionStats(sessionKey)
		var memPct int
		if stats.ContextWindow > 0 {
			memPct = int(float64(stats.TokenEstimate) / float64(stats.ContextWindow) * 100)
		}
		summaryInfo := "no"
		if stats.HasSummary {
			summaryInfo = "yes"
		}
		summarizingInfo := ""
		if stats.IsSummarizing {
			summarizingInfo = " (summarizing...)"
		}
		sb.WriteString("\n🤖 <b>Main Agent (this session):</b>\n")
		sb.WriteString(fmt.Sprintf("  Model: <code>%s</code>\n", html.EscapeString(c.config.Agents.Defaults.Model)))
		sb.WriteString(fmt.Sprintf("  Messages: %d\n", stats.MessageCount))
		sb.WriteString(fmt.Sprintf("  Context: ~%d/%d tokens (%d%%)%s\n",
			stats.TokenEstimate, stats.ContextWindow, memPct, html.EscapeString(summarizingInfo)))
		sb.WriteString(fmt.Sprintf("  Summary: %s\n", summaryInfo))
	}

	params := &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: message.Chat.ID},
		Text:      sb.String(),
		ParseMode: telego.ModeHTML,
		ReplyParameters: &telego.ReplyParameters{
			MessageID: message.MessageID,
		},
	}

	_, err := c.bot.SendMessage(ctx, params)
	if err == nil {
		return nil
	}

	// HTML parse failed; fallback to plain text to keep /status usable.
	logger.ErrorCF("telegram", "Failed to send /status with HTML, retrying plain text", map[string]interface{}{
		"error":   err.Error(),
		"chat_id": message.Chat.ID,
		"text":    truncateStr(sb.String(), 400),
	})

	_, plainErr := c.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: message.Chat.ID},
		Text:   sb.String(),
		ReplyParameters: &telego.ReplyParameters{
			MessageID: message.MessageID,
		},
	})
	if plainErr != nil {
		return plainErr
	}
	return nil
}

func (c *cmd) Kill(ctx context.Context, message telego.Message) error {
	if c.config == nil || !c.config.Tools.Spawn.Enabled {
		_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   "Kill is unavailable: spawn tool is disabled in config.",
			ReplyParameters: &telego.ReplyParameters{
				MessageID: message.MessageID,
			},
		})
		return err
	}

	if c.subagentManager == nil {
		_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   "Subagent manager is not initialized.",
		})
		return err
	}

	fields := strings.Fields(message.Text)
	taskID := ""
	if len(fields) > 1 {
		taskID = strings.TrimSpace(fields[1])
	}

	if taskID == "" {
		running := c.subagentManager.GetRunningTasks()
		if len(running) == 0 {
			_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
				ChatID: telego.ChatID{ID: message.Chat.ID},
				Text:   "Usage: /kill <task_id>\nNo running subagents.",
				ReplyParameters: &telego.ReplyParameters{
					MessageID: message.MessageID,
				},
			})
			return err
		}

		ids := make([]string, 0, len(running))
		for _, t := range running {
			ids = append(ids, t.ID)
		}
		_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   fmt.Sprintf("Usage: /kill <task_id>\nRunning task IDs: %s", strings.Join(ids, ", ")),
			ReplyParameters: &telego.ReplyParameters{
				MessageID: message.MessageID,
			},
		})
		return err
	}

	if err := c.subagentManager.Cancel(taskID); err != nil {
		_, sendErr := c.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   fmt.Sprintf("Failed to cancel task '%s': %v", taskID, err),
			ReplyParameters: &telego.ReplyParameters{
				MessageID: message.MessageID,
			},
		})
		return sendErr
	}

	_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: message.Chat.ID},
		Text:   fmt.Sprintf("Cancelled subagent task '%s'.", taskID),
		ReplyParameters: &telego.ReplyParameters{
			MessageID: message.MessageID,
		},
	})
	return err
}

func (c *cmd) Inject(ctx context.Context, message telego.Message) error {
	if c.bus == nil && c.agentLoop == nil {
		_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   "Inject is unavailable: neither bus nor agent loop is initialized.",
			ReplyParameters: &telego.ReplyParameters{
				MessageID: message.MessageID,
			},
		})
		return err
	}

	body := commandBody(message.Text)
	if body == "" {
		_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   "Usage: /inject MESSAGE",
			ReplyParameters: &telego.ReplyParameters{
				MessageID: message.MessageID,
			},
		})
		return err
	}

	sessionKey := fmt.Sprintf("telegram:%d", message.Chat.ID)
	content := fmt.Sprintf(
		"<urgent_message priority=\"high\" source=\"telegram:/inject\">\n%s\n</urgent_message>\nRespond immediately to this urgent instruction before less urgent tasks.",
		body,
	)

	if c.agentLoop != nil && c.agentLoop.InjectUrgent(sessionKey, content) {
		audit.Record("telegram_inject_active_run", map[string]interface{}{
			"session_key": sessionKey,
			"chat_id":     message.Chat.ID,
		})
		_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   "Injected into active run.",
			ReplyParameters: &telego.ReplyParameters{
				MessageID: message.MessageID,
			},
		})
		return err
	}

	if c.agentLoop != nil {
		chatID := fmt.Sprintf("%d", message.Chat.ID)
		senderID := fmt.Sprintf("%d", message.From.ID)

		_, ackErr := c.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   "Inject accepted. Processing now (queue bypass).",
			ReplyParameters: &telego.ReplyParameters{
				MessageID: message.MessageID,
			},
		})
		if ackErr != nil {
			return ackErr
		}

		go func(chatID, senderID, sessionKey, content string, telegramChatID int64) {
			audit.Record("telegram_inject_immediate_start", map[string]interface{}{
				"session_key": sessionKey,
				"chat_id":     telegramChatID,
			})
			resp, err := c.agentLoop.ProcessImmediate(context.Background(), "telegram", chatID, senderID, sessionKey, content)
			if err != nil {
				logger.ErrorCF("telegram", "Immediate inject processing failed", map[string]interface{}{
					"chat_id":     telegramChatID,
					"session_key": sessionKey,
					"error":       err.Error(),
				})
				resp = "Inject processing failed. Please retry."
			} else if strings.TrimSpace(resp) == "" {
				resp = "Inject processed."
			}
			audit.Record("telegram_inject_immediate_done", map[string]interface{}{
				"session_key":  sessionKey,
				"chat_id":      telegramChatID,
				"response_len": len(resp),
				"error":        errString(err),
			})

			if _, sendErr := c.bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: telego.ChatID{ID: telegramChatID},
				Text:   resp,
			}); sendErr != nil {
				logger.ErrorCF("telegram", "Failed to send immediate inject response", map[string]interface{}{
					"chat_id":     telegramChatID,
					"session_key": sessionKey,
					"error":       sendErr.Error(),
				})
			}
		}(chatID, senderID, sessionKey, content, message.Chat.ID)

		return nil
	}

	if c.bus != nil {
		id, ok := c.bus.PublishInboundWithID(bus.InboundMessage{
			Channel:    "telegram",
			SenderID:   fmt.Sprintf("%d", message.From.ID),
			ChatID:     fmt.Sprintf("%d", message.Chat.ID),
			Content:    content,
			SessionKey: sessionKey,
			Metadata: map[string]string{
				"urgent": "true",
				"source": "telegram:/inject",
			},
		})
		if ok {
			audit.Record("telegram_inject_queued_fallback", map[string]interface{}{
				"session_key": sessionKey,
				"chat_id":     message.Chat.ID,
				"queue_id":    id,
			})
			_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
				ChatID: telego.ChatID{ID: message.Chat.ID},
				Text:   fmt.Sprintf("Inject queued (fallback). Queue ID: %s", id),
				ReplyParameters: &telego.ReplyParameters{
					MessageID: message.MessageID,
				},
			})
			return err
		}
		logger.WarnCF("telegram", "Inject message dropped: inbound queue timeout", map[string]interface{}{
			"chat_id": message.Chat.ID,
		})
	}

	_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: message.Chat.ID},
		Text:   "Failed to process inject message.",
		ReplyParameters: &telego.ReplyParameters{
			MessageID: message.MessageID,
		},
	})
	return err
}

func (c *cmd) First(ctx context.Context, message telego.Message) error {
	if c.bus == nil {
		_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   "Queue is unavailable.",
			ReplyParameters: &telego.ReplyParameters{
				MessageID: message.MessageID,
			},
		})
		return err
	}

	body := commandBody(message.Text)
	if body == "" {
		_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   "Usage: /first MESSAGE",
			ReplyParameters: &telego.ReplyParameters{
				MessageID: message.MessageID,
			},
		})
		return err
	}

	sessionKey := fmt.Sprintf("telegram:%d", message.Chat.ID)
	id, ok := c.bus.InsertInboundFirst(bus.InboundMessage{
		Channel:    "telegram",
		SenderID:   fmt.Sprintf("%d", message.From.ID),
		ChatID:     fmt.Sprintf("%d", message.Chat.ID),
		Content:    body,
		SessionKey: sessionKey,
		Metadata: map[string]string{
			"source": "telegram:/first",
		},
	})
	if !ok {
		_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   "Failed to move message to queue head.",
			ReplyParameters: &telego.ReplyParameters{
				MessageID: message.MessageID,
			},
		})
		return err
	}
	audit.Record("telegram_first_queued_head", map[string]interface{}{
		"session_key": sessionKey,
		"chat_id":     message.Chat.ID,
		"queue_id":    id,
	})

	_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: message.Chat.ID},
		Text:   fmt.Sprintf("Queued at head. Queue ID: %s", id),
		ReplyParameters: &telego.ReplyParameters{
			MessageID: message.MessageID,
		},
	})
	return err
}

// Urgent is kept as backward-compatible alias for /inject.
func (c *cmd) Urgent(ctx context.Context, message telego.Message) error {
	return c.Inject(ctx, message)
}

func commandBody(raw string) string {
	raw = strings.TrimSpace(raw)
	if idx := strings.Index(raw, " "); idx >= 0 && idx+1 < len(raw) {
		return strings.TrimSpace(raw[idx+1:])
	}
	return ""
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// formatTimeShort formats Unix millisecond timestamp to HH:MM:SS
func formatTimeShort(unixMilli int64) string {
	if unixMilli == 0 {
		return "--:--:--"
	}
	return time.UnixMilli(unixMilli).Format("15:04:05")
}

// truncateStr truncates a string to maxLen characters
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func resolvedProviderName(cfg *config.Config, model string) string {
	if cfg == nil {
		return "unknown"
	}

	if p := strings.TrimSpace(strings.ToLower(cfg.Agents.Defaults.Provider)); p != "" {
		switch p {
		case "gpt":
			return "openai (explicit)"
		case "glm":
			return "zhipu (explicit)"
		case "claude":
			return "anthropic (explicit)"
		case "google":
			return "gemini (explicit)"
		case "copilot":
			return "github_copilot (explicit)"
		case "claudecode":
			return "claude-cli (explicit)"
		case "codex-code":
			return "codex-cli (explicit)"
		default:
			return p + " (explicit)"
		}
	}

	lowerModel := strings.ToLower(model)
	switch {
	case (strings.Contains(lowerModel, "kimi") || strings.Contains(lowerModel, "moonshot") || strings.HasPrefix(model, "moonshot/")) && cfg.Providers.Moonshot.APIKey != "":
		return "moonshot (auto)"
	case (strings.HasPrefix(model, "openrouter/") || strings.HasPrefix(model, "anthropic/") || strings.HasPrefix(model, "openai/") || strings.HasPrefix(model, "meta-llama/") || strings.HasPrefix(model, "deepseek/") || strings.HasPrefix(model, "google/")) && cfg.Providers.OpenRouter.APIKey != "":
		return "openrouter (auto)"
	case (strings.Contains(lowerModel, "claude") || strings.HasPrefix(model, "anthropic/")) && (cfg.Providers.Anthropic.APIKey != "" || cfg.Providers.Anthropic.AuthMethod != ""):
		return "anthropic (auto)"
	case (strings.Contains(lowerModel, "gpt") || strings.HasPrefix(model, "openai/")) && (cfg.Providers.OpenAI.APIKey != "" || cfg.Providers.OpenAI.AuthMethod != ""):
		return "openai (auto)"
	case (strings.Contains(lowerModel, "gemini") || strings.HasPrefix(model, "google/")) && cfg.Providers.Gemini.APIKey != "":
		return "gemini (auto)"
	case (strings.Contains(lowerModel, "glm") || strings.Contains(lowerModel, "zhipu") || strings.Contains(lowerModel, "zai")) && cfg.Providers.Zhipu.APIKey != "":
		return "zhipu (auto)"
	case (strings.Contains(lowerModel, "groq") || strings.HasPrefix(model, "groq/")) && cfg.Providers.Groq.APIKey != "":
		return "groq (auto)"
	case (strings.Contains(lowerModel, "nvidia") || strings.HasPrefix(model, "nvidia/")) && cfg.Providers.Nvidia.APIKey != "":
		return "nvidia (auto)"
	case (strings.Contains(lowerModel, "ollama") || strings.HasPrefix(model, "ollama/")) && cfg.Providers.Ollama.APIKey != "":
		return "ollama (auto)"
	case cfg.Providers.VLLM.APIBase != "":
		return "vllm (auto)"
	case cfg.Providers.OpenRouter.APIKey != "":
		return "openrouter (fallback)"
	default:
		return "unresolved (no matching provider/api key)"
	}
}
