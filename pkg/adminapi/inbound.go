package adminapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/providers"
)

type routeRegistrar interface {
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}

type inboundAPI struct {
	msgBus *bus.MessageBus
	hist   sessionHistoryProvider
}

type sessionHistoryProvider interface {
	GetSessionHistory(sessionKey string) []providers.Message
}

func RegisterInboundRoutes(r routeRegistrar, msgBus *bus.MessageBus, history sessionHistoryProvider) {
	if r == nil || msgBus == nil {
		return
	}
	api := &inboundAPI{msgBus: msgBus, hist: history}
	r.HandleFunc("/api/v1/inbound", api.handleInboundRoot)
	r.HandleFunc("/api/v1/inbound/", api.handleInboundItem)
	r.HandleFunc("/api/v1/history", api.handleHistory)
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

func writeJSON(w http.ResponseWriter, status int, body map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
