package adminapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/tools"
)

type routeRegistrar interface {
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}

type sessionHistoryProvider interface {
	GetSessionHistory(sessionKey string) []providers.Message
}

type injectUrgentProvider interface {
	InjectUrgent(sessionKey, content string) bool
}

type subagentManagerProvider interface {
	GetSubagentManager() *tools.SubagentManager
}

type inboundAPI struct {
	msgBus      *bus.MessageBus
	hist        sessionHistoryProvider
	injector    injectUrgentProvider
	subagentMgr *tools.SubagentManager
}

func RegisterInboundRoutes(r routeRegistrar, msgBus *bus.MessageBus, runtime sessionHistoryProvider) {
	if r == nil || msgBus == nil {
		return
	}
	api := &inboundAPI{
		msgBus: msgBus,
		hist:   runtime,
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
	r.HandleFunc("/api/v1/main/message", api.handleMainMessage)
	r.HandleFunc("/api/v1/subagents", api.handleSubagents)
	r.HandleFunc("/api/v1/subagents/", api.handleSubagentItem)
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

	tasks := a.subagentMgr.ListTasks()
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

func writeJSON(w http.ResponseWriter, status int, body map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
