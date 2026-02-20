package adminapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/session"
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
	r.HandleFunc("/api/v1/sessions", api.handleSessions)
	r.HandleFunc("/api/v1/main/message", api.handleMainMessage)
	r.HandleFunc("/api/v1/subagents", api.handleSubagents)
	r.HandleFunc("/api/v1/subagents/", api.handleSubagentItem)
	r.HandleFunc("/api/v1/events", api.handleEvents)
	r.HandleFunc("/api/v1/runtime", api.handleRuntime)
	RegisterDashboardRoutes(r)
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
	if sessionKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "session_key query param is required"})
		return
	}
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
		content = fmt.Sprintf("<urgent_message priority=\"high\" source=\"api:/api/v1/main/message\">\n%s\n</urgent_message>\nRespond immediately to this urgent instruction before less urgent tasks.", req.Content)
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
	if r.Method != http.MethodGet {
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

	sendSnapshot := func() error {
		payload := map[string]any{
			"session_key": sessionKey,
			"inbound":     a.msgBus.ListInbound(),
			"history":     a.hist.GetSessionHistory(sessionKey),
			"subagents":   a.listSubagentViews(),
			"sessions":    a.listSessions(200),
			"ts":          time.Now().UnixMilli(),
		}
		raw, _ := json.Marshal(payload)
		if _, err := fmt.Fprintf(w, "event: snapshot\n"); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", raw); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}

	_, _ = fmt.Fprintf(w, "retry: 1500\n\n")
	flusher.Flush()

	if err := sendSnapshot(); err != nil {
		return
	}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	keepalive := time.NewTicker(15 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if err := sendSnapshot(); err != nil {
				return
			}
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

	writeJSON(w, http.StatusOK, map[string]any{"runtime": runtime})
}

func runtimeControlCommands(prefix string, includeKill bool) []string {
	cmds := []string{
		prefix + "help",
		prefix + "status",
		prefix + "models",
		prefix + "channels",
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
	return out
}

func writeJSON(w http.ResponseWriter, status int, body map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
