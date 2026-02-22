package adminapi

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/session"
	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/sipeed/picoclaw/pkg/version"
)

type routeRegistrar interface {
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}

type sessionHistoryProvider interface {
	GetSessionHistory(sessionKey string) []providers.Message
}

type runtimeInfoProvider interface {
	GetRuntimeInfo() map[string]interface{}
}

type sessionListProvider interface {
	ListSessions(limit int) []session.SessionSummary
}

type channelStatusProvider interface {
	GetStatus() map[string]interface{}
	GetEnabledChannels() []string
}

type injectUrgentProvider interface {
	InjectUrgent(sessionKey, content string) bool
}

type subagentManagerProvider interface {
	GetSubagentManager() *tools.SubagentManager
}

type inboundAPI struct {
	msgBus         *bus.MessageBus
	hist           sessionHistoryProvider
	injector       injectUrgentProvider
	subagentMgr    *tools.SubagentManager
	cfg            *config.Config
	cfgPath        string
	channelRuntime channelStatusProvider
}

type RuntimeOptions struct {
	Config         *config.Config
	ConfigPath     string
	ChannelRuntime channelStatusProvider
}

func RegisterInboundRoutes(r routeRegistrar, msgBus *bus.MessageBus, runtime sessionHistoryProvider, opts *RuntimeOptions) {
	if r == nil || msgBus == nil {
		return
	}
	api := &inboundAPI{
		msgBus: msgBus,
		hist:   runtime,
	}
	if opts != nil {
		api.cfg = opts.Config
		api.cfgPath = strings.TrimSpace(opts.ConfigPath)
		api.channelRuntime = opts.ChannelRuntime
	}
	if inj, ok := runtime.(injectUrgentProvider); ok {
		api.injector = inj
	}
	if subProvider, ok := runtime.(subagentManagerProvider); ok {
		api.subagentMgr = subProvider.GetSubagentManager()
	}

	r.HandleFunc("/api/v1/inbound", api.handleInboundRoot)
	r.HandleFunc("/api/v1/inbound/", api.handleInboundItem)
	r.HandleFunc("/api/v1/history", api.handleHistory)
	r.HandleFunc("/api/v1/history/agent-log", api.handleAgentLogHistory)
	r.HandleFunc("/api/v1/timeline", api.handleTimeline)
	r.HandleFunc("/api/v1/sessions", api.handleSessions)
	r.HandleFunc("/api/v1/main/message", api.handleMainMessage)
	r.HandleFunc("/api/v1/subagents", api.handleSubagents)
	r.HandleFunc("/api/v1/subagents/", api.handleSubagentItem)
	r.HandleFunc("/api/v1/agents/", api.handleAgentItem)
	r.HandleFunc("/api/v1/events", api.handleEvents)
	r.HandleFunc("/api/v1/runtime", api.handleRuntime)
	RegisterDashboardRoutes(r)
}

type timelineItem struct {
	ID          string         `json:"id"`
	EventID     string         `json:"event_id"`
	TimestampMS int64          `json:"timestamp_ms"`
	Source      string         `json:"source"`
	SessionKey  string         `json:"session_key"`
	AgentID     string         `json:"agent_id"`
	Role        string         `json:"role"`
	Kind        string         `json:"kind"`
	Level       string         `json:"level"`
	Content     string         `json:"content"`
	Payload     map[string]any `json:"payload"`
}

func (a *inboundAPI) handleInboundRoot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": a.msgBus.ListInbound(),
	})
}

func (a *inboundAPI) handleInboundItem(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/inbound/")
	path = strings.TrimSpace(path)
	if path == "" {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}

	if strings.HasSuffix(path, "/move") {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
			return
		}
		id := strings.TrimSuffix(path, "/move")
		id = strings.TrimSpace(strings.TrimSuffix(id, "/"))
		if id == "" || strings.Contains(id, "/") {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid inbound message id"})
			return
		}
		var req struct {
			Index int `json:"index"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON body"})
			return
		}
		if !a.msgBus.MoveInbound(id, req.Index) {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "inbound message not found or index out of range"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}

	id := strings.TrimSpace(strings.TrimSuffix(path, "/"))
	if id == "" || strings.Contains(id, "/") {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid inbound message id"})
		return
	}

	switch r.Method {
	case http.MethodPatch:
		var req struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON body"})
			return
		}
		if !a.msgBus.UpdateInboundContent(id, req.Content) {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "inbound message not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	case http.MethodDelete:
		if !a.msgBus.DeleteInbound(id) {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "inbound message not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

func (a *inboundAPI) handleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	if a.hist == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "history provider unavailable"})
		return
	}
	sessionKey := strings.TrimSpace(r.URL.Query().Get("session_key"))
	writeJSON(w, http.StatusOK, map[string]any{
		"session_key": sessionKey,
		"items":       a.hist.GetSessionHistory(sessionKey),
	})
}

func (a *inboundAPI) handleSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	lp, ok := a.hist.(sessionListProvider)
	if !ok || lp == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "sessions provider unavailable"})
		return
	}
	limit := 200
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	items := lp.ListSessions(limit)
	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
	})
}

func (a *inboundAPI) handleTimeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	if a.hist == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "history provider unavailable"})
		return
	}

	sessionFilter := strings.TrimSpace(r.URL.Query().Get("session"))
	agentFilter := strings.TrimSpace(r.URL.Query().Get("agent"))
	kindFilter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("kind")))
	levelFilter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("level")))
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	fromMS := int64(0)
	if raw := strings.TrimSpace(r.URL.Query().Get("from")); raw != "" {
		if n, err := strconv.ParseInt(raw, 10, 64); err == nil && n > 0 {
			fromMS = n
		}
	}
	limit := 3000
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 10000 {
		limit = 10000
	}

	items := make([]timelineItem, 0, 4096)
	sessionKeys := make([]string, 0, 64)
	if sessionFilter != "" {
		sessionKeys = append(sessionKeys, sessionFilter)
	} else if lp, ok := a.hist.(sessionListProvider); ok && lp != nil {
		for _, s := range lp.ListSessions(1000) {
			if strings.TrimSpace(s.Key) != "" {
				sessionKeys = append(sessionKeys, s.Key)
			}
		}
	}

	for _, key := range sessionKeys {
		h := a.hist.GetSessionHistory(key)
		for i, m := range h {
			content := strings.TrimSpace(m.Content)
			role := strings.ToLower(strings.TrimSpace(m.Role))
			ts := m.TimestampMS
			if ts <= 0 {
				ts = int64(i + 1)
			}
			kind := classifyTimelineKind(role, content, a.cfg)
			level := classifyTimelineLevel(content)
			agentID := detectTimelineAgentID(content)
			item := timelineItem{
				ID:          fmt.Sprintf("%s#%d", key, i),
				EventID:     timelineEventID("session", key, strconv.Itoa(i), strconv.FormatInt(ts, 10), role, content),
				TimestampMS: ts,
				Source:      "session",
				SessionKey:  key,
				AgentID:     agentID,
				Role:        role,
				Kind:        kind,
				Level:       level,
				Content:     content,
				Payload: map[string]any{
					"role": role,
				},
			}
			if !timelineMatches(item, sessionFilter, agentFilter, kindFilter, levelFilter, query, fromMS) {
				continue
			}
			items = append(items, item)
		}
	}

	for _, q := range a.msgBus.ListInbound() {
		msg := q.Message
		key := strings.TrimSpace(msg.SessionKey)
		if key == "" {
			if ch := strings.TrimSpace(msg.Channel); ch != "" && strings.TrimSpace(msg.ChatID) != "" {
				key = ch + ":" + strings.TrimSpace(msg.ChatID)
			}
		}
		ts := q.EnqueuedAt
		if ts <= 0 {
			ts = parseTimestampMS(msg.Metadata["enqueued_at"])
		}
		if ts <= 0 {
			ts = time.Now().UnixMilli()
		}
		kind := "queue"
		if raw := strings.ToLower(strings.TrimSpace(msg.Metadata["kind"])); raw != "" {
			kind = raw
		}
		level := "info"
		if raw := strings.ToLower(strings.TrimSpace(msg.Metadata["level"])); raw != "" {
			level = raw
		}
		content := strings.TrimSpace(msg.Content)
		item := timelineItem{
			ID:          "queue:" + q.ID,
			EventID:     timelineEventID("queue", q.ID, strconv.FormatInt(ts, 10), key, content),
			TimestampMS: ts,
			Source:      "queue",
			SessionKey:  key,
			AgentID:     strings.TrimSpace(msg.SenderID),
			Role:        "queue",
			Kind:        kind,
			Level:       level,
			Content:     content,
			Payload: map[string]any{
				"inbound_id":  q.ID,
				"channel":     msg.Channel,
				"chat_id":     msg.ChatID,
				"sender_id":   msg.SenderID,
				"enqueued_at": q.EnqueuedAt,
				"metadata":    mapStringStringAny(msg.Metadata),
			},
		}
		if !timelineMatches(item, sessionFilter, agentFilter, kindFilter, levelFilter, query, fromMS) {
			continue
		}
		items = append(items, item)
	}

	for _, out := range a.msgBus.ListOutboundHistory(1000) {
		msg := out.Message
		key := strings.TrimSpace(msg.Channel) + ":" + strings.TrimSpace(msg.ChatID)
		ts := out.TimestampMS
		if ts <= 0 {
			ts = time.Now().UnixMilli()
		}
		content := strings.TrimSpace(msg.Content)
		kind := classifyTimelineKind("system", content, a.cfg)
		if kind == "sys" {
			// Keep default for explicit SYS/AUTO; otherwise outbound is normal message.
			sysPrefix := "[SYS]"
			autoPrefix := "[AUTO]"
			if a.cfg != nil {
				if p := strings.TrimSpace(a.cfg.Gateway.SysMessagePrefix); p != "" {
					sysPrefix = p
				}
				if p := strings.TrimSpace(a.cfg.Gateway.AutoFinalPrefix); p != "" {
					autoPrefix = p
				}
			}
			if !strings.HasPrefix(content, sysPrefix) && !strings.HasPrefix(content, autoPrefix) {
				kind = "message"
			}
		}
		level := classifyTimelineLevel(content)
		item := timelineItem{
			ID:          "outbound:" + out.ID,
			EventID:     timelineEventID("outbound", out.ID, strconv.FormatInt(ts, 10), key, content),
			TimestampMS: ts,
			Source:      "outbound",
			SessionKey:  key,
			AgentID:     "",
			Role:        "assistant",
			Kind:        kind,
			Level:       level,
			Content:     content,
			Payload: map[string]any{
				"outbound_id": out.ID,
				"channel":     msg.Channel,
				"chat_id":     msg.ChatID,
			},
		}
		if !timelineMatches(item, sessionFilter, agentFilter, kindFilter, levelFilter, query, fromMS) {
			continue
		}
		items = append(items, item)
	}

	if a.cfg != nil && sessionFilter == "" {
		mainLogPath := filepath.Join(a.cfg.LoggingDirPath(), "agent.jsonl")
		lines, _ := readLastLines(mainLogPath, 5000)
		for i, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			entry, ok := parseRuntimeLogEntry(line)
			if !ok {
				// JSONL-only strictness: skip non-structured lines.
				continue
			}
			ts := entry.TimestampMS
			if ts <= 0 {
				ts = int64(i + 1)
			}
			level := strings.ToLower(strings.TrimSpace(entry.Level))
			if level == "" {
				level = "info"
			}
			agentID := detectTimelineAgentID(entry.Message)
			item := timelineItem{
				ID:          fmt.Sprintf("agent.jsonl#%d", i),
				EventID:     timelineEventID("agent_log", strconv.FormatInt(ts, 10), entry.Component, level, entry.Message),
				TimestampMS: ts,
				Source:      "agent_log",
				SessionKey:  "agent.jsonl",
				AgentID:     agentID,
				Role:        "log",
				Kind:        classifyRuntimeLogKind(entry),
				Level:       level,
				Content:     entry.Message,
				Payload: map[string]any{
					"log_path":  mainLogPath,
					"component": entry.Component,
					"fields":    entry.Fields,
					"line":      line,
				},
			}
			if !timelineMatches(item, sessionFilter, agentFilter, kindFilter, levelFilter, query, fromMS) {
				continue
			}
			items = append(items, item)
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].TimestampMS != items[j].TimestampMS {
			return items[i].TimestampMS < items[j].TimestampMS
		}
		return items[i].ID < items[j].ID
	})
	if len(items) > limit {
		items = items[len(items)-limit:]
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"count": len(items),
	})
}

func (a *inboundAPI) handleAgentLogHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	if a.cfg == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "config unavailable"})
		return
	}
	tail := 500
	if raw := strings.TrimSpace(r.URL.Query().Get("tail")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			tail = n
		}
	}
	if tail > 5000 {
		tail = 5000
	}

	logPath := filepath.Join(a.cfg.LoggingDirPath(), "agent.jsonl")
	lines, err := readLastLines(logPath, tail)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"path":  logPath,
			"lines": []string{},
			"error": err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"path":  logPath,
		"lines": lines,
	})
}

func (a *inboundAPI) handleMainMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req struct {
		SessionKey string `json:"session_key"`
		Channel    string `json:"channel"`
		ChatID     string `json:"chat_id"`
		SenderID   string `json:"sender_id"`
		Content    string `json:"content"`
		Urgent     bool   `json:"urgent"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON body"})
		return
	}
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "content is required"})
		return
	}
	if strings.TrimSpace(req.Channel) == "" {
		req.Channel = "web"
	}
	if strings.TrimSpace(req.ChatID) == "" {
		req.ChatID = "dashboard"
	}
	if strings.TrimSpace(req.SenderID) == "" {
		req.SenderID = "web"
	}
	if strings.TrimSpace(req.SessionKey) == "" {
		req.SessionKey = fmt.Sprintf("%s:%s", req.Channel, req.ChatID)
	}

	content := req.Content
	if req.Urgent {
		content = fmt.Sprintf("<inject_message source=\"api:/api/v1/main/message\">\n%s\n</inject_message>", req.Content)
		if a.injector != nil && a.injector.InjectUrgent(req.SessionKey, content) {
			writeJSON(w, http.StatusOK, map[string]any{
				"ok":          true,
				"injected":    true,
				"session_key": req.SessionKey,
			})
			return
		}
	}

	meta := map[string]string{"source": "api:/api/v1/main/message"}
	if req.Urgent {
		meta["urgent"] = "true"
	}

	id, ok := a.msgBus.PublishInboundWithID(bus.InboundMessage{
		Channel:    req.Channel,
		SenderID:   req.SenderID,
		ChatID:     req.ChatID,
		Content:    content,
		SessionKey: req.SessionKey,
		Metadata:   meta,
	})
	if !ok {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "inbound queue timeout/full"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":          true,
		"queued":      true,
		"id":          id,
		"session_key": req.SessionKey,
	})
}

func (a *inboundAPI) handleSubagents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	if a.subagentMgr == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "subagent manager unavailable"})
		return
	}

	out := a.listSubagentViews()
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (a *inboundAPI) handleSubagentItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	if a.subagentMgr == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "subagent manager unavailable"})
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/subagents/")
	id = strings.TrimSpace(strings.TrimSuffix(id, "/"))
	if id == "" || strings.Contains(id, "/") {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid subagent id"})
		return
	}
	task, ok := a.subagentMgr.GetTask(id)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "subagent task not found"})
		return
	}
	if r.Method == http.MethodDelete {
		if err := a.subagentMgr.Cancel(id); err != nil {
			writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":     true,
			"id":     id,
			"status": "cancelled",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":          task.ID,
		"label":       task.Label,
		"name":        task.Name,
		"directory":   task.Directory,
		"status":      task.Status,
		"task":        task.Task,
		"result":      task.Result,
		"created":     task.Created,
		"started":     task.Started,
		"ended":       task.Ended,
		"pending":     task.PendingMsgs,
		"origin_chat": task.OriginChatID,
	})
}

func (a *inboundAPI) handleAgentItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	if a.cfg == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "config unavailable"})
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/")
	path = strings.TrimSpace(path)
	if !strings.HasSuffix(path, "/log") {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
	agentID := strings.TrimSpace(strings.TrimSuffix(path, "/log"))
	agentID = strings.TrimSuffix(agentID, "/")
	if agentID == "" || strings.Contains(agentID, "/") || strings.Contains(agentID, "..") {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid agent id"})
		return
	}
	tail := 500
	if raw := strings.TrimSpace(r.URL.Query().Get("tail")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			tail = n
		}
	}
	if tail > 5000 {
		tail = 5000
	}

	logPath := filepath.Join(a.cfg.LoggingDirPath(), "agents", agentID+".jsonl")
	lines, err := readLastLines(logPath, tail)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"agent_id": agentID,
			"path":     logPath,
			"lines":    []string{},
			"error":    err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"agent_id": agentID,
		"path":     logPath,
		"lines":    lines,
	})
}

func (a *inboundAPI) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	if a.hist == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "history provider unavailable"})
		return
	}
	sessionKey := strings.TrimSpace(r.URL.Query().Get("session_key"))
	if sessionKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "session_key query param is required"})
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "streaming unsupported"})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	type eventSnapshot struct {
		Inbound   []bus.InboundQueueItem
		History   []providers.Message
		Subagents []taskView
		Sessions  []session.SessionSummary
		Runtime   map[string]any
	}
	capture := func() eventSnapshot {
		history := []providers.Message{}
		if sessionKey != "" {
			history = a.hist.GetSessionHistory(sessionKey)
		}
		return eventSnapshot{
			Inbound:   a.msgBus.ListInbound(),
			History:   history,
			Subagents: a.listSubagentViews(),
			Sessions:  a.listSessions(200),
			Runtime:   a.buildRuntimeData(),
		}
	}
	sendEvent := func(eventName string, payload map[string]any) error {
		raw, _ := json.Marshal(payload)
		if _, err := fmt.Fprintf(w, "event: %s\n", eventName); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", raw); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}
	sendSnapshot := func(s eventSnapshot) error {
		payload := map[string]any{
			"session_key": sessionKey,
			"inbound":     s.Inbound,
			"history":     s.History,
			"subagents":   s.Subagents,
			"sessions":    s.Sessions,
			"runtime":     s.Runtime,
			"ts":          time.Now().UnixMilli(),
		}
		return sendEvent("snapshot", payload)
	}
	sendPatch := func(prev, next eventSnapshot) error {
		payload := map[string]any{
			"session_key": sessionKey,
			"ts":          time.Now().UnixMilli(),
		}
		changed := false
		if !jsonEqual(prev.Inbound, next.Inbound) {
			payload["inbound"] = next.Inbound
			changed = true
		}
		if !jsonEqual(prev.History, next.History) {
			payload["history"] = next.History
			changed = true
		}
		if !jsonEqual(prev.Subagents, next.Subagents) {
			payload["subagents"] = next.Subagents
			changed = true
		}
		if !jsonEqual(prev.Sessions, next.Sessions) {
			payload["sessions"] = next.Sessions
			changed = true
		}
		if !jsonEqual(prev.Runtime, next.Runtime) {
			payload["runtime"] = next.Runtime
			changed = true
		}
		if !changed {
			return nil
		}
		return sendEvent("patch", payload)
	}

	_, _ = fmt.Fprintf(w, "retry: 1500\n\n")
	flusher.Flush()

	prev := capture()
	if err := sendSnapshot(prev); err != nil {
		return
	}
	ticker := time.NewTicker(750 * time.Millisecond)
	defer ticker.Stop()
	keepalive := time.NewTicker(15 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			next := capture()
			if err := sendPatch(prev, next); err != nil {
				return
			}
			prev = next
		case <-keepalive.C:
			if _, err := fmt.Fprintf(w, ": keepalive %d\n\n", time.Now().Unix()); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (a *inboundAPI) listSessions(limit int) []session.SessionSummary {
	lp, ok := a.hist.(sessionListProvider)
	if !ok || lp == nil {
		return nil
	}
	return lp.ListSessions(limit)
}

func (a *inboundAPI) handleRuntime(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"runtime": a.buildRuntimeData()})
}

func (a *inboundAPI) buildRuntimeData() map[string]any {
	runtime := map[string]any{
		"version": map[string]any{
			"app": version.Format(),
			"go":  version.GetGoVersion(),
		},
		"inbound_queue": map[string]any{
			"count": len(a.msgBus.ListInbound()),
		},
		"subagents": map[string]any{
			"running_count": 0,
			"recent_count":  0,
			"queue_count":   0,
		},
	}

	if rp, ok := a.hist.(runtimeInfoProvider); ok {
		info := rp.GetRuntimeInfo()
		runtime["agent"] = info
	}

	if a.subagentMgr != nil {
		runtime["subagents"] = map[string]any{
			"running_count": len(a.subagentMgr.GetRunningTasks()),
			"recent_count":  len(a.subagentMgr.GetRecentTasks(20)),
			"queue_count":   a.subagentMgr.GetMessageQueueCount(),
		}
	}

	if a.channelRuntime != nil {
		enabled := a.channelRuntime.GetEnabledChannels()
		sort.Strings(enabled)
		runtime["channels"] = map[string]any{
			"enabled_count": len(enabled),
			"enabled":       enabled,
			"status":        a.channelRuntime.GetStatus(),
		}
	}

	if a.cfg != nil {
		prefix := "+"
		if p := strings.TrimSpace(a.cfg.Ingress.ConcatPrefix); p != "" {
			prefix = string([]rune(p)[0])
		}
		commands := runtimeControlCommands(prefix, a.cfg.Tools.Spawn.Enabled)
		runtime["controls"] = map[string]any{
			"prefix":   prefix,
			"count":    len(commands),
			"commands": commands,
		}
		runtime["catalog"] = map[string]any{
			"defined_agents": a.listDefinedAgentsCatalog(),
			"skills":         a.listSkillsCatalog(),
		}
		runtime["model"] = map[string]any{
			"provider": a.cfg.Agents.Defaults.Provider,
			"model":    a.cfg.Agents.Defaults.Model,
		}
		cfgData, cfgErr := a.loadSanitizedConfig()
		runtime["config"] = map[string]any{
			"path":      a.cfgPath,
			"available": cfgErr == "",
			"error":     cfgErr,
			"sanitized": cfgData,
		}
	}
	return runtime
}

func (a *inboundAPI) listDefinedAgentsCatalog() []map[string]any {
	out := make([]map[string]any, 0)
	if a.cfg == nil {
		return out
	}
	workspace := strings.TrimSpace(a.cfg.WorkspacePath())
	workspaceAgents := tools.LoadAvailableAgents(workspace)
	seen := make(map[string]bool)

	for _, item := range workspaceAgents {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		seen[name] = true
		out = append(out, map[string]any{
			"name":        name,
			"description": item.Description,
			"source":      "workspace",
		})
	}

	cfgNames := make([]string, 0, len(a.cfg.Agents.NamedAgents))
	for name := range a.cfg.Agents.NamedAgents {
		cfgNames = append(cfgNames, name)
	}
	sort.Strings(cfgNames)
	for _, name := range cfgNames {
		if seen[name] {
			continue
		}
		out = append(out, map[string]any{
			"name":        name,
			"description": "Configured named agent",
			"source":      "config",
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		li := strings.TrimSpace(fmt.Sprint(out[i]["name"]))
		rj := strings.TrimSpace(fmt.Sprint(out[j]["name"]))
		return li < rj
	})
	return out
}

func (a *inboundAPI) listSkillsCatalog() []map[string]any {
	out := make([]map[string]any, 0)
	if a.cfg == nil {
		return out
	}

	workspace := strings.TrimSpace(a.cfg.WorkspacePath())
	homeDir, _ := os.UserHomeDir()
	globalSkills := ""
	if homeDir != "" {
		globalSkills = filepath.Join(homeDir, ".picoclaw", "skills")
	}
	wd, _ := os.Getwd()
	builtinSkills := ""
	if wd != "" {
		builtinSkills = filepath.Join(wd, "skills")
	}

	loader := skills.NewSkillsLoader(workspace, globalSkills, builtinSkills)
	all := loader.ListSkills()
	sort.SliceStable(all, func(i, j int) bool {
		return all[i].Name < all[j].Name
	})

	for _, s := range all {
		out = append(out, map[string]any{
			"name":        s.Name,
			"description": s.Description,
			"path":        s.Path,
			"source":      s.Source,
		})
	}
	return out
}

func runtimeControlCommands(prefix string, includeKill bool) []string {
	cmds := []string{
		prefix + "help",
		prefix + "status",
		prefix + "models",
		prefix + "channels",
		prefix + "stop",
		prefix + "continue",
		prefix + "inject MESSAGE",
		prefix + "first MESSAGE",
		prefix + "delete",
		prefix + prefix + "MESSAGE",
	}
	if includeKill {
		cmds = append(cmds, prefix+"kill TASK_ID")
	}
	return cmds
}

func (a *inboundAPI) loadSanitizedConfig() (any, string) {
	// Prefer raw config file for exact user-edited structure (includes named agents).
	if a.cfgPath != "" {
		raw, err := os.ReadFile(a.cfgPath)
		if err == nil && len(raw) > 0 {
			var doc any
			if err := json.Unmarshal(raw, &doc); err == nil {
				return sanitizeConfigValue(doc, ""), ""
			}
			return nil, "failed to parse config JSON"
		}
	}
	// Fallback to loaded config object.
	if a.cfg != nil {
		raw, err := json.Marshal(a.cfg)
		if err != nil {
			return nil, "failed to marshal config"
		}
		var doc any
		if err := json.Unmarshal(raw, &doc); err != nil {
			return nil, "failed to decode config"
		}
		return sanitizeConfigValue(doc, ""), ""
	}
	return nil, "config unavailable"
}

func sanitizeConfigValue(v any, key string) any {
	switch vv := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(vv))
		for k, val := range vv {
			out[k] = sanitizeConfigValue(val, k)
		}
		return out
	case []any:
		out := make([]any, 0, len(vv))
		for _, item := range vv {
			out = append(out, sanitizeConfigValue(item, key))
		}
		return out
	case string:
		if isSensitiveConfigKey(key) {
			return maskSecret(vv)
		}
		return vv
	default:
		if isSensitiveConfigKey(key) {
			return "***"
		}
		return v
	}
}

func isSensitiveConfigKey(key string) bool {
	k := strings.ToLower(strings.TrimSpace(key))
	if k == "" {
		return false
	}
	switch {
	case strings.Contains(k, "token"):
		return true
	case strings.Contains(k, "secret"):
		return true
	case strings.Contains(k, "api_key"):
		return true
	case strings.Contains(k, "apikey"):
		return true
	case strings.Contains(k, "password"):
		return true
	default:
		return false
	}
}

func maskSecret(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 4 {
		return strings.Repeat("*", len(s))
	}
	return s[:2] + strings.Repeat("*", len(s)-4) + s[len(s)-2:]
}

type taskView struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Name      string `json:"name"`
	Directory string `json:"directory"`
	Status    string `json:"status"`
	Created   int64  `json:"created"`
	Started   int64  `json:"started"`
	Ended     int64  `json:"ended"`
	Pending   int    `json:"pending"`
}

func (a *inboundAPI) listSubagentViews() []taskView {
	if a.subagentMgr == nil {
		return nil
	}
	tasks := a.subagentMgr.ListTasks()
	out := make([]taskView, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, taskView{
			ID:        t.ID,
			Label:     t.Label,
			Name:      t.Name,
			Directory: t.Directory,
			Status:    t.Status,
			Created:   t.Created,
			Started:   t.Started,
			Ended:     t.Ended,
			Pending:   len(t.PendingMsgs),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		li := out[i]
		rj := out[j]
		leftTS := li.Started
		if leftTS == 0 {
			leftTS = li.Created
		}
		rightTS := rj.Started
		if rightTS == 0 {
			rightTS = rj.Created
		}
		if leftTS != rightTS {
			return leftTS > rightTS
		}
		if li.Created != rj.Created {
			return li.Created > rj.Created
		}
		return li.ID < rj.ID
	})
	return out
}

func readLastLines(path string, limit int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if limit <= 0 {
		limit = 1
	}
	ring := make([]string, 0, limit)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if len(ring) < limit {
			ring = append(ring, line)
			continue
		}
		copy(ring, ring[1:])
		ring[len(ring)-1] = line
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	out := make([]string, len(ring))
	copy(out, ring)
	return out, nil
}

func writeJSON(w http.ResponseWriter, status int, body map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func jsonEqual(left, right any) bool {
	lb, err := json.Marshal(left)
	if err != nil {
		return false
	}
	rb, err := json.Marshal(right)
	if err != nil {
		return false
	}
	return string(lb) == string(rb)
}

func detectTimelineAgentID(content string) string {
	lc := strings.ToLower(content)
	if idx := strings.Index(lc, "subagent-"); idx >= 0 {
		rest := content[idx:]
		for i := 0; i < len(rest); i++ {
			ch := rest[i]
			if ch == '\n' || ch == ' ' || ch == '"' || ch == '\'' || ch == ')' || ch == ']' {
				return rest[:i]
			}
		}
		return rest
	}
	return ""
}

func classifyTimelineKind(role, content string, cfg *config.Config) string {
	role = strings.ToLower(strings.TrimSpace(role))
	content = strings.TrimSpace(content)
	lc := strings.ToLower(content)
	switch role {
	case "tool":
		return "tool"
	case "assistant", "user":
		return "message"
	case "log":
		switch {
		case strings.Contains(lc, "llm "):
			return "llm"
		case strings.Contains(lc, "queue"), strings.Contains(lc, "inbound"), strings.Contains(lc, "outbound"):
			return "queue"
		case strings.Contains(lc, "error"):
			return "error"
		case strings.Contains(lc, "warn"):
			return "warn"
		case strings.Contains(lc, "debug"):
			return "debug"
		default:
			return "info"
		}
	case "system":
		sysPrefix := "[SYS]"
		autoPrefix := "[AUTO]"
		if cfg != nil {
			if p := strings.TrimSpace(cfg.Gateway.SysMessagePrefix); p != "" {
				sysPrefix = p
			}
			if p := strings.TrimSpace(cfg.Gateway.AutoFinalPrefix); p != "" {
				autoPrefix = p
			}
		}
		switch {
		case strings.HasPrefix(content, sysPrefix):
			return "sys"
		case strings.HasPrefix(content, autoPrefix):
			return "auto"
		default:
			return "sys"
		}
	default:
		return "info"
	}
}

func classifyTimelineLevel(content string) string {
	lc := strings.ToLower(content)
	switch {
	case strings.Contains(lc, "error"), strings.Contains(lc, "failed"), strings.Contains(lc, "panic"):
		return "error"
	case strings.Contains(lc, "warn"), strings.Contains(lc, "timeout"):
		return "warn"
	case strings.Contains(lc, "debug"):
		return "debug"
	default:
		return "info"
	}
}

func timelineMatches(item timelineItem, session, agent, kind, level, q string, fromMS int64) bool {
	if session != "" && item.SessionKey != session {
		return false
	}
	if agent != "" {
		agent = strings.ToLower(agent)
		if !strings.Contains(strings.ToLower(item.AgentID), agent) &&
			!strings.Contains(strings.ToLower(item.Content), agent) &&
			!strings.Contains(strings.ToLower(item.SessionKey), agent) {
			return false
		}
	}
	if kind != "" && kind != "all" && strings.ToLower(item.Kind) != kind {
		return false
	}
	if level != "" && level != "all" && strings.ToLower(item.Level) != level {
		return false
	}
	if fromMS > 0 && item.TimestampMS < fromMS {
		return false
	}
	if q != "" {
		stack := strings.ToLower(item.Content + "\n" + item.SessionKey + "\n" + item.AgentID + "\n" + item.Kind + "\n" + item.Level + "\n" + item.Source)
		if !strings.Contains(stack, q) {
			return false
		}
	}
	return true
}

type runtimeLogEntry struct {
	Level       string         `json:"level"`
	Timestamp   string         `json:"timestamp"`
	Component   string         `json:"component"`
	Message     string         `json:"message"`
	Fields      map[string]any `json:"fields"`
	TimestampMS int64          `json:"-"`
}

func parseRuntimeLogEntry(line string) (runtimeLogEntry, bool) {
	var entry runtimeLogEntry
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		return runtimeLogEntry{}, false
	}
	entry.Message = strings.TrimSpace(entry.Message)
	if entry.Message == "" {
		return runtimeLogEntry{}, false
	}
	if t, err := time.Parse(time.RFC3339, strings.TrimSpace(entry.Timestamp)); err == nil {
		entry.TimestampMS = t.UnixMilli()
	}
	if entry.Fields == nil {
		entry.Fields = map[string]any{}
	}
	return entry, true
}

func classifyRuntimeLogKind(entry runtimeLogEntry) string {
	comp := strings.ToLower(strings.TrimSpace(entry.Component))
	msg := strings.ToLower(strings.TrimSpace(entry.Message))
	switch comp {
	case "tool":
		return "tool"
	case "agent":
		if strings.Contains(msg, "llm ") {
			return "llm"
		}
		return "message"
	case "bus":
		return "queue"
	case "channels", "gateway", "system":
		return "sys"
	}
	level := strings.ToLower(strings.TrimSpace(entry.Level))
	switch level {
	case "error":
		return "error"
	case "warn":
		return "warn"
	case "debug":
		return "debug"
	default:
		return "info"
	}
}

func mapStringStringAny(in map[string]string) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func parseTimestampMS(raw string) int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

func timelineEventID(source string, parts ...string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(source))
	_, _ = h.Write([]byte{0})
	for _, p := range parts {
		_, _ = h.Write([]byte(p))
		_, _ = h.Write([]byte{0})
	}
	sum := h.Sum64()
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], sum)
	return fmt.Sprintf("%s:%x", source, buf[:])
}
