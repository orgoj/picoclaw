package channels

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	"github.com/sipeed/picoclaw/pkg/agent"
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
	Models(ctx context.Context, message telego.Message) error
	Channels(ctx context.Context, message telego.Message) error
}

type cmd struct {
	bot             *telego.Bot
	config          *config.Config
	subagentManager *tools.SubagentManager
	agentLoop       *agent.AgentLoop
}

func NewTelegramCommands(bot *telego.Bot, cfg *config.Config, subagentManager *tools.SubagentManager, agentLoop *agent.AgentLoop) TelegramCommander {
	return &cmd{
		bot:             bot,
		config:          cfg,
		subagentManager: subagentManager,
		agentLoop:       agentLoop,
	}
}

func (c *cmd) Help(ctx context.Context, message telego.Message) error {
	msg := `🦞 *PicoClaw Commands*

/start - Start the bot
/help - Show this help message
/status - Show system and subagents status
/kill <task_id> - Cancel running subagent task
/models - List available models
/channels - List enabled channels
	`
	_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: message.Chat.ID},
		Text:      msg,
		ParseMode: telego.ModeMarkdown,
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
	provider := c.config.Agents.Defaults.Provider
	if provider == "" {
		provider = "configured default"
	}
	response := fmt.Sprintf("🦞 *Configured Model*\n\n- Model: `%s` *\n- Provider: `%s`",
		c.config.Agents.Defaults.Model, provider)

	_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: message.Chat.ID},
		Text:      response,
		ParseMode: telego.ModeMarkdown,
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

	response := fmt.Sprintf("🦞 *Available Channels*\n\n%s\n\n\\* = active", strings.Join(channels, "\n"))

	_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: message.Chat.ID},
		Text:      response,
		ParseMode: telego.ModeMarkdown,
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
	sb.WriteString("📊 *PicoClaw Status*\n\n")

	// Version info
	sb.WriteString(fmt.Sprintf("📦 *Version:* `%s`\n", version.Format()))
	sb.WriteString(fmt.Sprintf("🔧 *Go:* `%s`\n\n", version.GetGoVersion()))

	// Capabilities info (tools, skills, named agents)
	if c.agentLoop != nil {
		startupInfo := c.agentLoop.GetStartupInfo()
		toolsInfo := startupInfo["tools"].(map[string]interface{})
		skillsInfo := startupInfo["skills"].(map[string]interface{})
		agentsInfo := startupInfo["agents"].(map[string]interface{})

		sb.WriteString("🧰 *Capabilities:*\n")
		sb.WriteString(fmt.Sprintf("  Tools: %d\n", toolsInfo["count"]))
		sb.WriteString(fmt.Sprintf("  Skills: %d/%d\n", skillsInfo["available"], skillsInfo["total"]))
		sb.WriteString(fmt.Sprintf("  Named Agents: %d\n", agentsInfo["count"]))

		if names, ok := agentsInfo["names"].([]string); ok {
			if len(names) == 0 {
				sb.WriteString("  Agents: None\n")
			} else {
				sb.WriteString(fmt.Sprintf("  Agents: %s\n", escapeMD(strings.Join(names, ", "))))
			}
		}
		sb.WriteString("\n")
	}

	// Running subagents with enhanced details
	sb.WriteString("🔄 *Running Subagents:*\n")
	if len(running) == 0 {
		sb.WriteString("  None\n")
	} else {
		for _, t := range running {
			// Format time range
			startTime := formatTimeShort(t.Started)
			timeRange := fmt.Sprintf("[%s - ...]", startTime)

			sb.WriteString(fmt.Sprintf("• `%s` %s\n", t.ID, timeRange))

			// Label (if set)
			if t.Label != "" {
				sb.WriteString(fmt.Sprintf("  Label: %s\n", escapeMD(t.Label)))
			}

			// Task preview (first 30 chars) - escape BEFORE truncate to avoid broken markdown
			taskPreview := truncateStr(escapeMD(t.Task), 30)
			sb.WriteString(fmt.Sprintf("  Task: %s\n", taskPreview))

			// Pending messages count for this task
			if len(t.PendingMsgs) > 0 {
				sb.WriteString(fmt.Sprintf("  📬 Pending: %d message(s)\n", len(t.PendingMsgs)))
			}
			sb.WriteString("\n")
		}
	}

	// Recent completed tasks
	sb.WriteString(fmt.Sprintf("\n✅ *Recent \\(%d\\):*\n", len(recent)))
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
			sb.WriteString(fmt.Sprintf("• `%s` %s %s\n", t.ID, timeRange, statusIcon))

			if t.Label != "" {
				sb.WriteString(fmt.Sprintf("  Label: %s\n", escapeMD(t.Label)))
			}
		}
	}

	// Message queue info
	sb.WriteString(fmt.Sprintf("\n📬 *Message Queue:* %d\n", queueCount))
	if queueCount > 0 {
		firstMsg := c.subagentManager.GetFirstQueuedMessage()
		if firstMsg != "" {
			// IMPORTANT: escape BEFORE truncate to avoid breaking markdown entities
			sb.WriteString(fmt.Sprintf("  Next: \"%s\"\n", truncateStr(escapeMD(firstMsg), 80)))
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
			summarizingInfo = " _(summarizing...)_"
		}
		sb.WriteString("\n🤖 *Main Agent \\(this session\\):*\n")
		sb.WriteString(fmt.Sprintf("  Model: `%s`\n", c.config.Agents.Defaults.Model))
		sb.WriteString(fmt.Sprintf("  Messages: %d\n", stats.MessageCount))
		sb.WriteString(fmt.Sprintf("  Context: \\~%d/%d tokens \\(%d%%\\)%s\n",
			stats.TokenEstimate, stats.ContextWindow, memPct, summarizingInfo))
		sb.WriteString(fmt.Sprintf("  Summary: %s\n", summaryInfo))
	}

	params := &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: message.Chat.ID},
		Text:      sb.String(),
		ParseMode: telego.ModeMarkdownV2,
		ReplyParameters: &telego.ReplyParameters{
			MessageID: message.MessageID,
		},
	}

	_, err := c.bot.SendMessage(ctx, params)
	if err == nil {
		return nil
	}

	// MarkdownV2 is strict; fallback to plain text to keep /status usable.
	logger.ErrorCF("telegram", "Failed to send /status with MarkdownV2, retrying plain text", map[string]interface{}{
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
	return err
}

func (c *cmd) Kill(ctx context.Context, message telego.Message) error {
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

// escapeMD escapes special characters for MarkdownV2
func escapeMD(text string) string {
	// MarkdownV2 special characters that need escaping
	specialChars := "_*[]()~`>#+-=|{}.!"
	var result strings.Builder
	for _, c := range text {
		if strings.ContainsRune(specialChars, c) {
			result.WriteRune('\\')
		}
		result.WriteRune(c)
	}
	return result.String()
}
