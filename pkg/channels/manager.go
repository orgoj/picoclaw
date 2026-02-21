// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package channels

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/audit"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/constants"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/sipeed/picoclaw/pkg/version"
)

type Manager struct {
	channels        map[string]Channel
	bus             *bus.MessageBus
	config          *config.Config
	subagentManager *tools.SubagentManager
	agentLoop       *agent.AgentLoop
	dispatchTask    *asyncTask
	mu              sync.RWMutex
}

type asyncTask struct {
	cancel context.CancelFunc
}

func NewManager(cfg *config.Config, messageBus *bus.MessageBus, subagentManager *tools.SubagentManager, agentLoop *agent.AgentLoop) (*Manager, error) {
	m := &Manager{
		channels:        make(map[string]Channel),
		bus:             messageBus,
		config:          cfg,
		subagentManager: subagentManager,
		agentLoop:       agentLoop,
	}

	if err := m.initChannels(); err != nil {
		return nil, err
	}

	return m, nil
}

func (m *Manager) initChannels() error {
	logger.InfoC("channels", "Initializing channel manager")

	if m.config.Channels.Telegram.Enabled && m.config.Channels.Telegram.Token != "" {
		logger.DebugC("channels", "Attempting to initialize Telegram channel")
		telegram, err := NewTelegramChannel(m.config, m.bus, m.subagentManager, m.agentLoop)
		if err != nil {
			logger.ErrorCF("channels", "Failed to initialize Telegram channel", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			m.channels["telegram"] = telegram
			logger.InfoC("channels", "Telegram channel enabled successfully")
		}
	}

	if m.config.Channels.WhatsApp.Enabled && m.config.Channels.WhatsApp.BridgeURL != "" {
		logger.DebugC("channels", "Attempting to initialize WhatsApp channel")
		whatsapp, err := NewWhatsAppChannel(m.config.Channels.WhatsApp, m.bus)
		if err != nil {
			logger.ErrorCF("channels", "Failed to initialize WhatsApp channel", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			m.channels["whatsapp"] = whatsapp
			logger.InfoC("channels", "WhatsApp channel enabled successfully")
		}
	}

	if m.config.Channels.Feishu.Enabled {
		logger.DebugC("channels", "Attempting to initialize Feishu channel")
		feishu, err := NewFeishuChannel(m.config.Channels.Feishu, m.bus)
		if err != nil {
			logger.ErrorCF("channels", "Failed to initialize Feishu channel", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			m.channels["feishu"] = feishu
			logger.InfoC("channels", "Feishu channel enabled successfully")
		}
	}

	if m.config.Channels.Discord.Enabled && m.config.Channels.Discord.Token != "" {
		logger.DebugC("channels", "Attempting to initialize Discord channel")
		discord, err := NewDiscordChannel(m.config.Channels.Discord, m.bus)
		if err != nil {
			logger.ErrorCF("channels", "Failed to initialize Discord channel", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			m.channels["discord"] = discord
			logger.InfoC("channels", "Discord channel enabled successfully")
		}
	}

	if m.config.Channels.MaixCam.Enabled {
		logger.DebugC("channels", "Attempting to initialize MaixCam channel")
		maixcam, err := NewMaixCamChannel(m.config.Channels.MaixCam, m.bus)
		if err != nil {
			logger.ErrorCF("channels", "Failed to initialize MaixCam channel", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			m.channels["maixcam"] = maixcam
			logger.InfoC("channels", "MaixCam channel enabled successfully")
		}
	}

	if m.config.Channels.QQ.Enabled {
		logger.DebugC("channels", "Attempting to initialize QQ channel")
		qq, err := NewQQChannel(m.config.Channels.QQ, m.bus)
		if err != nil {
			logger.ErrorCF("channels", "Failed to initialize QQ channel", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			m.channels["qq"] = qq
			logger.InfoC("channels", "QQ channel enabled successfully")
		}
	}

	if m.config.Channels.DingTalk.Enabled && m.config.Channels.DingTalk.ClientID != "" {
		logger.DebugC("channels", "Attempting to initialize DingTalk channel")
		dingtalk, err := NewDingTalkChannel(m.config.Channels.DingTalk, m.bus)
		if err != nil {
			logger.ErrorCF("channels", "Failed to initialize DingTalk channel", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			m.channels["dingtalk"] = dingtalk
			logger.InfoC("channels", "DingTalk channel enabled successfully")
		}
	}

	if m.config.Channels.Slack.Enabled && m.config.Channels.Slack.BotToken != "" {
		logger.DebugC("channels", "Attempting to initialize Slack channel")
		slackCh, err := NewSlackChannel(m.config.Channels.Slack, m.bus)
		if err != nil {
			logger.ErrorCF("channels", "Failed to initialize Slack channel", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			m.channels["slack"] = slackCh
			logger.InfoC("channels", "Slack channel enabled successfully")
		}
	}

	if m.config.Channels.LINE.Enabled && m.config.Channels.LINE.ChannelAccessToken != "" {
		logger.DebugC("channels", "Attempting to initialize LINE channel")
		line, err := NewLINEChannel(m.config.Channels.LINE, m.bus)
		if err != nil {
			logger.ErrorCF("channels", "Failed to initialize LINE channel", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			m.channels["line"] = line
			logger.InfoC("channels", "LINE channel enabled successfully")
		}
	}

	if m.config.Channels.OneBot.Enabled && m.config.Channels.OneBot.WSUrl != "" {
		logger.DebugC("channels", "Attempting to initialize OneBot channel")
		onebot, err := NewOneBotChannel(m.config.Channels.OneBot, m.bus)
		if err != nil {
			logger.ErrorCF("channels", "Failed to initialize OneBot channel", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			m.channels["onebot"] = onebot
			logger.InfoC("channels", "OneBot channel enabled successfully")
		}
	}

	logger.InfoCF("channels", "Channel initialization completed", map[string]interface{}{
		"enabled_channels": len(m.channels),
	})
	m.attachInboundControlHandlers()

	return nil
}

func (m *Manager) attachInboundControlHandlers() {
	for _, channel := range m.channels {
		type controlCapable interface {
			SetInboundControlHandler(func(bus.InboundMessage) bool)
		}
		if cc, ok := channel.(controlCapable); ok {
			cc.SetInboundControlHandler(m.handleInboundControl)
		}
	}
}

func (m *Manager) StartAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.channels) == 0 {
		logger.WarnC("channels", "No channels enabled")
		return nil
	}

	logger.InfoC("channels", "Starting all channels")

	dispatchCtx, cancel := context.WithCancel(ctx)
	m.dispatchTask = &asyncTask{cancel: cancel}

	go m.dispatchOutbound(dispatchCtx)

	for name, channel := range m.channels {
		logger.InfoCF("channels", "Starting channel", map[string]interface{}{
			"channel": name,
		})
		if err := channel.Start(ctx); err != nil {
			logger.ErrorCF("channels", "Failed to start channel", map[string]interface{}{
				"channel": name,
				"error":   err.Error(),
			})
		}
	}

	logger.InfoC("channels", "All channels started")
	return nil
}

func (m *Manager) StopAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	logger.InfoC("channels", "Stopping all channels")

	if m.dispatchTask != nil {
		m.dispatchTask.cancel()
		m.dispatchTask = nil
	}

	for name, channel := range m.channels {
		logger.InfoCF("channels", "Stopping channel", map[string]interface{}{
			"channel": name,
		})
		if err := channel.Stop(ctx); err != nil {
			logger.ErrorCF("channels", "Error stopping channel", map[string]interface{}{
				"channel": name,
				"error":   err.Error(),
			})
		}
	}

	logger.InfoC("channels", "All channels stopped")
	return nil
}

func (m *Manager) dispatchOutbound(ctx context.Context) {
	logger.InfoC("channels", "Outbound dispatcher started")

	for {
		select {
		case <-ctx.Done():
			logger.InfoC("channels", "Outbound dispatcher stopped")
			return
		default:
			msg, ok := m.bus.SubscribeOutbound(ctx)
			if !ok {
				continue
			}

			// Silently skip internal channels
			if constants.IsInternalChannel(msg.Channel) {
				continue
			}

			m.mu.RLock()
			channel, exists := m.channels[msg.Channel]
			m.mu.RUnlock()

			if !exists {
				logger.WarnCF("channels", "Unknown channel for outbound message", map[string]interface{}{
					"channel": msg.Channel,
				})
				m.notifyDeliveryFailure(msg, fmt.Errorf("unknown channel: %s", msg.Channel))
				continue
			}

			if err := channel.Send(ctx, msg); err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
					logger.DebugCF("channels", "Skipping outbound send during shutdown", map[string]interface{}{
						"channel": msg.Channel,
						"error":   err.Error(),
					})
					continue
				}
				logger.ErrorCF("channels", "Error sending message to channel", map[string]interface{}{
					"channel": msg.Channel,
					"error":   err.Error(),
				})
				m.notifyDeliveryFailure(msg, err)
			}
		}
	}
}

func (m *Manager) notifyDeliveryFailure(msg bus.OutboundMessage, deliveryErr error) {
	if m.bus == nil || strings.TrimSpace(msg.Channel) == "" || strings.TrimSpace(msg.ChatID) == "" || deliveryErr == nil {
		return
	}
	// Prevent recursive feedback loops when the notice itself fails to deliver.
	if strings.Contains(msg.Content, "<delivery_failure") {
		return
	}

	sessionKey := fmt.Sprintf("%s:%s", msg.Channel, msg.ChatID)
	notice := fmt.Sprintf(
		"<delivery_failure channel=%q chat_id=%q>\nerror: %s\nfailed_content: %q\n</delivery_failure>\nReact to this failure before normal tasks.",
		msg.Channel,
		msg.ChatID,
		deliveryErr.Error(),
		truncateStr(msg.Content, 400),
	)

	inbound := bus.InboundMessage{
		Channel:    msg.Channel,
		SenderID:   "system:delivery",
		ChatID:     msg.ChatID,
		SessionKey: sessionKey,
		Content:    notice,
		Metadata: map[string]string{
			"source": "system:delivery_failure",
			"urgent": "true",
		},
	}
	if _, ok := m.bus.PublishInboundWithID(inbound); !ok {
		logger.WarnCF("channels", "Delivery failure notice dropped: inbound queue timeout", map[string]interface{}{
			"channel": msg.Channel,
			"chat_id": msg.ChatID,
		})
	}
}

func firstRuneString(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	return string([]rune(s)[0])
}

func (m *Manager) controlPrefix() string {
	if m.config == nil {
		return "+"
	}
	prefix := firstRuneString(m.config.Ingress.ConcatPrefix)
	if prefix == "" {
		return "+"
	}
	return prefix
}

func splitControlCommand(prefix, content string) (cmd, body string, ok bool) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = "+"
	}
	text := strings.TrimSpace(content)
	if !strings.HasPrefix(text, prefix) {
		return "", "", false
	}
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return "", "", false
	}
	first := fields[0]
	cmd = strings.TrimPrefix(strings.ToLower(first), strings.ToLower(prefix))
	body = strings.TrimSpace(strings.TrimPrefix(text, first))
	return cmd, body, true
}

func (m *Manager) sendControlReply(msg bus.InboundMessage, content string) {
	if m.bus == nil || strings.TrimSpace(content) == "" {
		return
	}
	if ok := m.bus.PublishOutbound(bus.OutboundMessage{
		Channel: msg.Channel,
		ChatID:  msg.ChatID,
		Content: content,
	}); !ok {
		logger.WarnCF("channels", "Failed to send control reply: outbound queue timeout", map[string]interface{}{
			"channel": msg.Channel,
			"chat_id": msg.ChatID,
		})
	}
}

func (m *Manager) handleInboundControl(msg bus.InboundMessage) bool {
	prefix := m.controlPrefix()
	text := strings.TrimSpace(msg.Content)
	if strings.HasPrefix(text, prefix+prefix) {
		appendBody := strings.TrimSpace(strings.TrimPrefix(text, prefix+prefix))
		if appendBody == "" {
			appendBody = "+"
		}
		return m.handleAppendControl(msg, appendBody)
	}
	if prefix != "+" && strings.HasPrefix(text, prefix+"+") {
		appendBody := strings.TrimSpace(strings.TrimPrefix(text, prefix+"+"))
		if appendBody == "" {
			appendBody = "+"
		}
		return m.handleAppendControl(msg, appendBody)
	}

	cmd, body, ok := splitControlCommand(prefix, msg.Content)
	if !ok {
		return false
	}

	switch cmd {
	case "help":
		return m.handleHelpControl(msg)
	case "status":
		return m.handleStatusControl(msg)
	case "models":
		return m.handleModelsControl(msg)
	case "channels":
		return m.handleChannelsControl(msg)
	case "inject", "urgent":
		return m.handleInjectControl(msg, body)
	case "first":
		return m.handleFirstControl(msg, body)
	case "kill":
		return m.handleKillControl(msg, body)
	case "delete":
		return m.handleDeleteControl(msg)
	default:
		return false
	}
}

func (m *Manager) handleHelpControl(msg bus.InboundMessage) bool {
	prefix := m.controlPrefix()
	lines := []string{
		"PicoClaw Controls",
		fmt.Sprintf("%shelp - Show this help", prefix),
		fmt.Sprintf("%sstatus - Immediate runtime status", prefix),
		fmt.Sprintf("%smodels - Show configured model/provider", prefix),
		fmt.Sprintf("%schannels - Show channel status", prefix),
		fmt.Sprintf("%sinject MESSAGE - Immediate priority inject", prefix),
		fmt.Sprintf("%sfirst MESSAGE - Put message at inbound queue head", prefix),
		fmt.Sprintf("%sdelete - Delete last queued message in this session", prefix),
		fmt.Sprintf("%s%sMESSAGE - Append to previous queued message in this session", prefix, prefix),
	}
	if m.config != nil && m.config.Tools.Spawn.Enabled {
		lines = append(lines, fmt.Sprintf("%skill TASK_ID - Cancel running subagent task", prefix))
	}
	m.sendControlReply(msg, strings.Join(lines, "\n"))
	return true
}

func isUrgentMeta(meta map[string]string) bool {
	if len(meta) == 0 {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(meta["urgent"]), "true")
}

func (m *Manager) canAppendToTail(tail bus.InboundQueueItem, msg bus.InboundMessage) bool {
	return tail.Message.SessionKey != "" &&
		tail.Message.SessionKey == msg.SessionKey &&
		tail.Message.Channel == msg.Channel &&
		tail.Message.ChatID == msg.ChatID &&
		tail.Message.SenderID == msg.SenderID &&
		!isUrgentMeta(tail.Message.Metadata)
}

func (m *Manager) handleAppendControl(msg bus.InboundMessage, appendBody string) bool {
	prefix := m.controlPrefix()
	appendBody = strings.TrimSpace(appendBody)
	if appendBody == "" {
		appendBody = "+"
	}
	items := m.bus.ListInbound()
	if len(items) == 0 {
		m.sendControlReply(msg, "Nothing to append to: inbound queue is empty.")
		return true
	}

	tail := items[len(items)-1]
	if !m.canAppendToTail(tail, msg) {
		m.sendControlReply(msg, "Nothing to append to: last queued message is from a different session/sender.")
		return true
	}

	appendContent := prefix + appendBody
	id, ok := m.bus.PublishInboundWithID(bus.InboundMessage{
		Channel:    msg.Channel,
		SenderID:   msg.SenderID,
		ChatID:     msg.ChatID,
		SessionKey: msg.SessionKey,
		Content:    appendContent,
		Metadata: map[string]string{
			"source": "channel:append",
		},
	})
	if !ok {
		m.sendControlReply(msg, "Failed to append to previous message.")
		return true
	}

	m.sendControlReply(msg, fmt.Sprintf("Appended to previous queued message. Queue ID: %s", id))
	audit.Record("control_append_tail", map[string]interface{}{
		"id":          id,
		"session_key": msg.SessionKey,
		"channel":     msg.Channel,
		"chat_id":     msg.ChatID,
	})
	return true
}

func (m *Manager) handleStatusControl(msg bus.InboundMessage) bool {
	var sb strings.Builder
	sb.WriteString("PicoClaw Status\n\n")
	sb.WriteString(fmt.Sprintf("Version: %s\n", version.Format()))
	sb.WriteString(fmt.Sprintf("Go: %s\n\n", version.GetGoVersion()))

	inbound := m.bus.ListInbound()
	sb.WriteString(fmt.Sprintf("Inbound queue: %d\n", len(inbound)))

	if m.subagentManager != nil {
		running := m.subagentManager.GetRunningTasks()
		recent := m.subagentManager.GetRecentTasks(5)
		sb.WriteString("\nRunning subagents:\n")
		if len(running) == 0 {
			sb.WriteString("- none\n")
		} else {
			for _, t := range running {
				timeRange := fmt.Sprintf("[%s - ...]", formatTimeShort(t.Started))
				sb.WriteString(fmt.Sprintf("- %s %s\n", t.ID, timeRange))
				if t.Label != "" {
					sb.WriteString(fmt.Sprintf("  Label: %s\n", t.Label))
				}
				if t.Name != "" {
					sb.WriteString(fmt.Sprintf("  Agent: %s\n", t.Name))
				}
				sb.WriteString(fmt.Sprintf("  Task: %s\n", truncateStr(t.Task, 60)))
				if len(t.PendingMsgs) > 0 {
					sb.WriteString(fmt.Sprintf("  Pending: %d\n", len(t.PendingMsgs)))
				}
			}
		}
		sb.WriteString(fmt.Sprintf("\nRecent subagents (%d):\n", len(recent)))
		if len(recent) == 0 {
			sb.WriteString("- none\n")
		} else {
			for _, t := range recent {
				timeRange := fmt.Sprintf("[%s - %s]", formatTimeShort(t.Started), formatTimeShort(t.Ended))
				sb.WriteString(fmt.Sprintf("- %s %s [%s]\n", t.ID, timeRange, t.Status))
				if t.Label != "" {
					sb.WriteString(fmt.Sprintf("  Label: %s\n", t.Label))
				}
				if t.Name != "" {
					sb.WriteString(fmt.Sprintf("  Agent: %s\n", t.Name))
				}
			}
		}
		sb.WriteString(fmt.Sprintf("Subagent msg queue: %d\n", m.subagentManager.GetMessageQueueCount()))
		firstMsg := m.subagentManager.GetFirstQueuedMessage()
		if firstMsg != "" {
			sb.WriteString(fmt.Sprintf("Next subagent message: %s\n", truncateStr(firstMsg, 80)))
		}
	} else {
		sb.WriteString("\nSubagents: unavailable\n")
	}

	if m.agentLoop != nil {
		info := m.agentLoop.GetRuntimeInfo()
		if toolsInfo, ok := info["tools"].(map[string]interface{}); ok {
			sb.WriteString(fmt.Sprintf("\nTools: %v\n", toolsInfo["count"]))
		}
		if skillsInfo, ok := info["skills"].(map[string]interface{}); ok {
			sb.WriteString(fmt.Sprintf("Skills: %v/%v\n", skillsInfo["available"], skillsInfo["total"]))
		}
		if agentsInfo, ok := info["agents"].(map[string]interface{}); ok {
			sb.WriteString(fmt.Sprintf("Named agents: %v\n", agentsInfo["count"]))
		}

		stats := m.agentLoop.GetSessionStats(msg.SessionKey)
		memPct := 0
		if stats.ContextWindow > 0 {
			memPct = int(float64(stats.TokenEstimate) / float64(stats.ContextWindow) * 100)
		}
		sb.WriteString("\nMain agent (this session):\n")
		sb.WriteString(fmt.Sprintf("- Model: %s\n", m.config.Agents.Defaults.Model))
		sb.WriteString(fmt.Sprintf("- Messages: %d\n", stats.MessageCount))
		sb.WriteString(fmt.Sprintf("- Context: ~%d/%d tokens (%d%%)\n", stats.TokenEstimate, stats.ContextWindow, memPct))
		if stats.HasSummary {
			sb.WriteString("- Summary: yes\n")
		} else {
			sb.WriteString("- Summary: no\n")
		}
	}

	chStatus := m.GetStatus()
	if len(chStatus) > 0 {
		sb.WriteString("\nChannels:\n")
		for name, raw := range chStatus {
			statusMap, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			running := false
			switch v := statusMap["running"].(type) {
			case bool:
				running = v
			case string:
				b, _ := strconv.ParseBool(v)
				running = b
			}
			if running {
				sb.WriteString(fmt.Sprintf("- %s: running\n", name))
			} else {
				sb.WriteString(fmt.Sprintf("- %s: stopped\n", name))
			}
		}
	}

	m.sendControlReply(msg, strings.TrimSpace(sb.String()))
	return true
}

func (m *Manager) handleModelsControl(msg bus.InboundMessage) bool {
	if m.config == nil {
		m.sendControlReply(msg, "Model info unavailable: config is not initialized.")
		return true
	}
	mainModel := m.config.Agents.Defaults.Model
	mainProvider := resolvedProviderName(m.config, mainModel)
	reply := fmt.Sprintf(
		"Configured models\nMain agent: %s (%s)\nSubagents: %s (%s)",
		mainModel, mainProvider, mainModel, mainProvider,
	)
	m.sendControlReply(msg, reply)
	return true
}

func (m *Manager) handleChannelsControl(msg bus.InboundMessage) bool {
	chStatus := m.GetStatus()
	if len(chStatus) == 0 {
		m.sendControlReply(msg, "No channels enabled.")
		return true
	}
	var sb strings.Builder
	sb.WriteString("Available channels\n")
	for name, raw := range chStatus {
		statusMap, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		running := false
		switch v := statusMap["running"].(type) {
		case bool:
			running = v
		case string:
			b, _ := strconv.ParseBool(v)
			running = b
		}
		if running {
			sb.WriteString(fmt.Sprintf("- %s *\n", name))
		} else {
			sb.WriteString(fmt.Sprintf("- %s\n", name))
		}
	}
	sb.WriteString("* = active")
	m.sendControlReply(msg, strings.TrimSpace(sb.String()))
	return true
}

func (m *Manager) handleInjectControl(msg bus.InboundMessage, body string) bool {
	prefix := m.controlPrefix()
	if strings.TrimSpace(body) == "" {
		m.sendControlReply(msg, fmt.Sprintf("Usage: %sinject MESSAGE", prefix))
		return true
	}

	content := fmt.Sprintf(
		"<urgent_message priority=\"high\" source=\"channel:%sinject\">\n%s\n</urgent_message>\nRespond immediately to this urgent instruction before less urgent tasks.",
		prefix,
		body,
	)

	if m.agentLoop != nil && m.agentLoop.InjectUrgent(msg.SessionKey, content) {
		audit.Record("control_inject_active_run", map[string]interface{}{
			"session_key": msg.SessionKey,
			"channel":     msg.Channel,
			"chat_id":     msg.ChatID,
		})
		m.sendControlReply(msg, "Injected into active run (preempting current cycle).")
		return true
	}

	if m.agentLoop != nil {
		m.sendControlReply(msg, "Inject accepted. Processing now (queue bypass).")
		audit.Record("control_inject_immediate_start", map[string]interface{}{
			"session_key": msg.SessionKey,
			"channel":     msg.Channel,
			"chat_id":     msg.ChatID,
		})

		go func() {
			resp, err := m.agentLoop.ProcessImmediate(context.Background(), msg.Channel, msg.ChatID, msg.SenderID, msg.SessionKey, content)
			if err != nil {
				logger.ErrorCF("channels", "Immediate inject processing failed", map[string]interface{}{
					"error":       err.Error(),
					"session_key": msg.SessionKey,
					"channel":     msg.Channel,
					"chat_id":     msg.ChatID,
				})
				resp = "Inject processing failed. Please retry."
			} else if strings.TrimSpace(resp) == "" {
				resp = "Inject processed."
			}

			m.sendControlReply(msg, resp)
			audit.Record("control_inject_immediate_done", map[string]interface{}{
				"session_key": msg.SessionKey,
				"channel":     msg.Channel,
				"chat_id":     msg.ChatID,
			})
		}()
		return true
	}

	meta := map[string]string{
		"urgent": "true",
		"source": fmt.Sprintf("channel:%sinject", prefix),
	}
	id, ok := m.bus.PublishInboundWithID(bus.InboundMessage{
		Channel:    msg.Channel,
		SenderID:   msg.SenderID,
		ChatID:     msg.ChatID,
		SessionKey: msg.SessionKey,
		Content:    content,
		Media:      msg.Media,
		Metadata:   meta,
	})
	if ok {
		m.sendControlReply(msg, fmt.Sprintf("Inject queued. Queue ID: %s", id))
	} else {
		m.sendControlReply(msg, "Failed to process inject message.")
	}
	return true
}

func (m *Manager) handleFirstControl(msg bus.InboundMessage, body string) bool {
	prefix := m.controlPrefix()
	if strings.TrimSpace(body) == "" {
		m.sendControlReply(msg, fmt.Sprintf("Usage: %sfirst MESSAGE", prefix))
		return true
	}

	id, ok := m.bus.InsertInboundFirst(bus.InboundMessage{
		Channel:    msg.Channel,
		SenderID:   msg.SenderID,
		ChatID:     msg.ChatID,
		SessionKey: msg.SessionKey,
		Content:    body,
		Media:      msg.Media,
		Metadata: map[string]string{
			"source": fmt.Sprintf("channel:%sfirst", prefix),
		},
	})
	if !ok {
		m.sendControlReply(msg, "Failed to enqueue at head.")
		return true
	}

	audit.Record("control_first_enqueue_head", map[string]interface{}{
		"id":          id,
		"session_key": msg.SessionKey,
		"channel":     msg.Channel,
		"chat_id":     msg.ChatID,
	})
	m.sendControlReply(msg, fmt.Sprintf("Enqueued at queue head. Queue ID: %s", id))
	return true
}

func (m *Manager) handleKillControl(msg bus.InboundMessage, body string) bool {
	prefix := m.controlPrefix()
	if m.config == nil || !m.config.Tools.Spawn.Enabled {
		m.sendControlReply(msg, "Kill is unavailable: spawn tool is disabled in config.")
		return true
	}
	if m.subagentManager == nil {
		m.sendControlReply(msg, "Subagent manager is not initialized.")
		return true
	}

	taskID := strings.TrimSpace(body)
	if taskID == "" {
		running := m.subagentManager.GetRunningTasks()
		if len(running) == 0 {
			m.sendControlReply(msg, fmt.Sprintf("Usage: %skill <task_id>\nNo running subagents.", prefix))
			return true
		}
		ids := make([]string, 0, len(running))
		for _, t := range running {
			ids = append(ids, t.ID)
		}
		m.sendControlReply(msg, fmt.Sprintf("Usage: %skill <task_id>\nRunning task IDs: %s", prefix, strings.Join(ids, ", ")))
		return true
	}

	if err := m.subagentManager.Cancel(taskID); err != nil {
		m.sendControlReply(msg, fmt.Sprintf("Failed to cancel task '%s': %v", taskID, err))
		return true
	}

	audit.Record("control_kill_cancel", map[string]interface{}{
		"task_id": taskID,
		"channel": msg.Channel,
		"chat_id": msg.ChatID,
	})
	m.sendControlReply(msg, fmt.Sprintf("Cancelled subagent task '%s'.", taskID))
	return true
}

func (m *Manager) handleDeleteControl(msg bus.InboundMessage) bool {
	items := m.bus.ListInbound()
	if len(items) == 0 {
		m.sendControlReply(msg, "Nothing to delete: inbound queue is empty.")
		return true
	}

	targetID := ""
	targetContent := ""
	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]
		if m.canAppendToTail(item, msg) {
			targetID = item.ID
			targetContent = item.Message.Content
			break
		}
	}
	if targetID == "" {
		m.sendControlReply(msg, "Nothing to delete: no queued message found for this session/sender.")
		return true
	}
	if !m.bus.DeleteInbound(targetID) {
		m.sendControlReply(msg, "Failed to delete queued message.")
		return true
	}

	audit.Record("control_delete_tail", map[string]interface{}{
		"id":          targetID,
		"session_key": msg.SessionKey,
		"channel":     msg.Channel,
		"chat_id":     msg.ChatID,
	})
	m.sendControlReply(msg, fmt.Sprintf("Deleted last queued message. Queue ID: %s\nContent: %q", targetID, truncateStr(targetContent, 80)))
	return true
}

func (m *Manager) GetChannel(name string) (Channel, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	channel, ok := m.channels[name]
	return channel, ok
}

func (m *Manager) GetStatus() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]interface{})
	for name, channel := range m.channels {
		status[name] = map[string]interface{}{
			"enabled": true,
			"running": channel.IsRunning(),
		}
	}
	return status
}

func (m *Manager) GetEnabledChannels() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.channels))
	for name := range m.channels {
		names = append(names, name)
	}
	return names
}

func (m *Manager) RegisterChannel(name string, channel Channel) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.channels[name] = channel
}

func (m *Manager) UnregisterChannel(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.channels, name)
}

func (m *Manager) SendToChannel(ctx context.Context, channelName, chatID, content string) error {
	m.mu.RLock()
	channel, exists := m.channels[channelName]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("channel %s not found", channelName)
	}

	msg := bus.OutboundMessage{
		Channel: channelName,
		ChatID:  chatID,
		Content: content,
	}

	return channel.Send(ctx, msg)
}
