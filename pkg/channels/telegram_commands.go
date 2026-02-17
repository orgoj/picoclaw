package channels

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/tools"
)

type TelegramCommander interface {
	Help(ctx context.Context, message telego.Message) error
	Start(ctx context.Context, message telego.Message) error
	Status(ctx context.Context, message telego.Message) error
	Models(ctx context.Context, message telego.Message) error
	Channels(ctx context.Context, message telego.Message) error
}

type cmd struct {
	bot             *telego.Bot
	config          *config.Config
	subagentManager *tools.SubagentManager
}

func NewTelegramCommands(bot *telego.Bot, cfg *config.Config, subagentManager *tools.SubagentManager) TelegramCommander {
	return &cmd{
		bot:             bot,
		config:          cfg,
		subagentManager: subagentManager,
	}
}

func (c *cmd) Help(ctx context.Context, message telego.Message) error {
	msg := `🦞 *PicoClaw Commands*

/start - Start the bot
/help - Show this help message
/status - Show running and stopped subagents
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

	tasks := c.subagentManager.ListTasks()

	var running []*tools.SubagentTask
	var stopped []*tools.SubagentTask

	for _, t := range tasks {
		if t.Status == "running" {
			running = append(running, t)
		} else {
			stopped = append(stopped, t)
		}
	}

	// Sort stopped by created time descending
	for i := 0; i < len(stopped); i++ {
		for j := i + 1; j < len(stopped); j++ {
			if stopped[i].Created < stopped[j].Created {
				stopped[i], stopped[j] = stopped[j], stopped[i]
			}
		}
	}

	// Limit to last 10
	if len(stopped) > 10 {
		stopped = stopped[:10]
	}

	var sb strings.Builder
	sb.WriteString("🦞 *PicoClaw Status*\n\n")

	sb.WriteString("🟢 *Running Subagents:*\n")
	if len(running) == 0 {
		sb.WriteString("- None\n")
	} else {
		for _, t := range running {
			created := time.UnixMilli(t.Created).Format("15:04:05")
			label := t.Label
			if label == "" {
				label = "(unnamed)"
			}
			sb.WriteString(fmt.Sprintf("- `%s`: %s (since %s)\n", t.ID, label, created))
		}
	}

	sb.WriteString("\n⚪ *Last 10 Stopped:*\n")
	if len(stopped) == 0 {
		sb.WriteString("- None\n")
	} else {
		for _, t := range stopped {
			label := t.Label
			if label == "" {
				label = "(unnamed)"
			}
			statusIcon := "⚪"
			if t.Status == "completed" {
				statusIcon = "✅"
			} else if t.Status == "failed" {
				statusIcon = "🔴"
			} else if t.Status == "cancelled" {
				statusIcon = "🚫"
			}
			sb.WriteString(fmt.Sprintf("- %s `%s`: %s\n", statusIcon, t.ID, label))
		}
	}

	_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: message.Chat.ID},
		Text:      sb.String(),
		ParseMode: telego.ModeMarkdown,
		ReplyParameters: &telego.ReplyParameters{
			MessageID: message.MessageID,
		},
	})
	return err
}
