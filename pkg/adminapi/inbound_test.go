package adminapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/session"
)

type muxRegistrar struct {
	mux *http.ServeMux
}

func (m *muxRegistrar) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	m.mux.HandleFunc(pattern, handler)
}

type fakeHistory struct{}

func (f fakeHistory) GetSessionHistory(sessionKey string) []providers.Message {
	return []providers.Message{{Role: "user", Content: "hello " + sessionKey}}
}

func (f fakeHistory) ListSessions(limit int) []session.SessionSummary {
	out := []session.SessionSummary{
		{Key: "telegram:1", Messages: 2, Updated: 200},
		{Key: "telegram:2", Messages: 1, Updated: 100},
	}
	if limit > 0 && len(out) > limit {
		return out[:limit]
	}
	return out
}

func TestInboundRoutes_ListPatchMoveDelete(t *testing.T) {
	msgBus := bus.NewMessageBus()
	id1, ok := msgBus.PublishInboundWithID(bus.InboundMessage{Content: "first"})
	if !ok {
		t.Fatal("failed to enqueue first")
	}
	id2, ok := msgBus.PublishInboundWithID(bus.InboundMessage{Content: "second"})
	if !ok {
		t.Fatal("failed to enqueue second")
	}

	mux := http.NewServeMux()
	RegisterInboundRoutes(&muxRegistrar{mux: mux}, msgBus, nil, nil)

	// PATCH second
	patchBody, _ := json.Marshal(map[string]any{"content": "second-edited"})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/inbound/"+id2, bytes.NewReader(patchBody))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("PATCH status=%d want=200 body=%s", rr.Code, rr.Body.String())
	}

	// MOVE second to index 0
	moveBody, _ := json.Marshal(map[string]any{"index": 0})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/inbound/"+id2+"/move", bytes.NewReader(moveBody))
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("MOVE status=%d want=200 body=%s", rr.Code, rr.Body.String())
	}

	// DELETE first
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/inbound/"+id1, nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("DELETE status=%d want=200 body=%s", rr.Code, rr.Body.String())
	}

	// LIST and verify remaining order/content
	req = httptest.NewRequest(http.MethodGet, "/api/v1/inbound", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET status=%d want=200 body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Items []bus.InboundQueueItem `json:"items"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("items len=%d want=1", len(resp.Items))
	}
	if resp.Items[0].ID != id2 || resp.Items[0].Message.Content != "second-edited" {
		t.Fatalf("remaining item=(%s,%s) want=(%s,second-edited)", resp.Items[0].ID, resp.Items[0].Message.Content, id2)
	}
}

func TestHistoryRoute(t *testing.T) {
	msgBus := bus.NewMessageBus()
	mux := http.NewServeMux()
	RegisterInboundRoutes(&muxRegistrar{mux: mux}, msgBus, fakeHistory{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/history?session_key=telegram:1", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET history status=%d want=200 body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		SessionKey string              `json:"session_key"`
		Items      []providers.Message `json:"items"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal history response: %v", err)
	}
	if resp.SessionKey != "telegram:1" {
		t.Fatalf("session_key=%s want=telegram:1", resp.SessionKey)
	}
	if len(resp.Items) != 1 || resp.Items[0].Content != "hello telegram:1" {
		t.Fatalf("history items mismatch: %+v", resp.Items)
	}
}

func TestMainMessageRoute_QueuesMessage(t *testing.T) {
	msgBus := bus.NewMessageBus()
	mux := http.NewServeMux()
	RegisterInboundRoutes(&muxRegistrar{mux: mux}, msgBus, fakeHistory{}, nil)

	body, _ := json.Marshal(map[string]any{
		"session_key": "telegram:1",
		"channel":     "telegram",
		"chat_id":     "1",
		"sender_id":   "tester",
		"content":     "hello",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/main/message", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("POST main message status=%d want=200 body=%s", rr.Code, rr.Body.String())
	}

	items := msgBus.ListInbound()
	if len(items) != 1 {
		t.Fatalf("inbound items len=%d want=1", len(items))
	}
	if items[0].Message.Content != "hello" {
		t.Fatalf("queued content=%q want=hello", items[0].Message.Content)
	}
}

func TestSessionsRoute_ReturnsSummaries(t *testing.T) {
	msgBus := bus.NewMessageBus()
	mux := http.NewServeMux()
	RegisterInboundRoutes(&muxRegistrar{mux: mux}, msgBus, fakeHistory{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions?limit=1", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET sessions status=%d want=200 body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Items []session.SessionSummary `json:"items"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal sessions response: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("sessions len=%d want=1", len(resp.Items))
	}
	if resp.Items[0].Key != "telegram:1" {
		t.Fatalf("sessions[0].key=%q want telegram:1", resp.Items[0].Key)
	}
}

func TestDashboardRoute(t *testing.T) {
	msgBus := bus.NewMessageBus()
	mux := http.NewServeMux()
	RegisterInboundRoutes(&muxRegistrar{mux: mux}, msgBus, fakeHistory{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /dashboard status=%d want=200", rr.Code)
	}
	if rr.Body.Len() == 0 {
		t.Fatal("dashboard response is empty")
	}
}

func TestEventsRoute_StreamsSnapshot(t *testing.T) {
	msgBus := bus.NewMessageBus()
	_, _ = msgBus.PublishInboundWithID(bus.InboundMessage{Content: "x"})
	mux := http.NewServeMux()
	RegisterInboundRoutes(&muxRegistrar{mux: mux}, msgBus, fakeHistory{}, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Millisecond)
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events?session_key=telegram:1", nil).WithContext(ctx)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	got := rr.Body.String()
	if !strings.Contains(got, "event: snapshot") {
		t.Fatalf("expected SSE snapshot event, got: %q", got)
	}
	if !strings.Contains(got, "\"session_key\":\"telegram:1\"") {
		t.Fatalf("expected session key in snapshot, got: %q", got)
	}
}

type fakeRuntimeHistory struct {
	fakeHistory
}

func (f fakeRuntimeHistory) GetRuntimeInfo() map[string]interface{} {
	return map[string]interface{}{
		"tools": map[string]interface{}{
			"count": 3,
			"names": []string{"exec", "read_file", "spawn"},
		},
		"skills": map[string]interface{}{
			"available": 4,
			"total":     4,
		},
		"agents": map[string]interface{}{
			"count": 2,
		},
	}
}

type fakeChannels struct{}

func (f fakeChannels) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"telegram": map[string]interface{}{"enabled": true, "running": true},
	}
}

func (f fakeChannels) GetEnabledChannels() []string {
	return []string{"telegram"}
}

func TestRuntimeRoute_ProvidesSummaryAndSanitizedConfig(t *testing.T) {
	msgBus := bus.NewMessageBus()
	_, _ = msgBus.PublishInboundWithID(bus.InboundMessage{Content: "x"})

	cfgPath := filepath.Join(t.TempDir(), "config.json")
	rawCfg := `{
  "agents": { "defaults": { "model": "glm-4.7", "provider": "zhipu" } },
  "ingress": { "concat_prefix": "+" },
  "tools": { "spawn": { "enabled": true } },
  "channels": { "telegram": { "enabled": true, "token": "1234567890TOKEN" } },
  "providers": { "zhipu": { "api_key": "SECRET_API_KEY_12345" } }
}`
	if err := os.WriteFile(cfgPath, []byte(rawCfg), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Model = "glm-4.7"
	cfg.Agents.Defaults.Provider = "zhipu"
	cfg.Ingress.ConcatPrefix = "+"
	cfg.Tools.Spawn.Enabled = true

	mux := http.NewServeMux()
	RegisterInboundRoutes(&muxRegistrar{mux: mux}, msgBus, fakeRuntimeHistory{}, &RuntimeOptions{
		Config:         cfg,
		ConfigPath:     cfgPath,
		ChannelRuntime: fakeChannels{},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runtime", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET runtime status=%d want=200 body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Runtime map[string]interface{} `json:"runtime"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal runtime response: %v", err)
	}
	if resp.Runtime["agent"] == nil {
		t.Fatalf("runtime.agent missing: %+v", resp.Runtime)
	}
	controls, ok := resp.Runtime["controls"].(map[string]interface{})
	if !ok {
		t.Fatalf("runtime.controls missing: %+v", resp.Runtime)
	}
	if controls["count"] == nil {
		t.Fatalf("runtime.controls.count missing: %+v", controls)
	}
	cfgNode, ok := resp.Runtime["config"].(map[string]interface{})
	if !ok {
		t.Fatalf("runtime.config missing: %+v", resp.Runtime)
	}
	if cfgNode["available"] != true {
		t.Fatalf("runtime.config.available=%v want=true", cfgNode["available"])
	}
	sanitizedRaw, err := json.Marshal(cfgNode["sanitized"])
	if err != nil {
		t.Fatalf("marshal sanitized config: %v", err)
	}
	sanitized := string(sanitizedRaw)
	if strings.Contains(sanitized, "SECRET_API_KEY_12345") || strings.Contains(sanitized, "1234567890TOKEN") {
		t.Fatalf("sanitized config leaked secrets: %s", sanitized)
	}
}

func TestAgentLogHistoryRoute_ReturnsTailLines(t *testing.T) {
	msgBus := bus.NewMessageBus()
	mux := http.NewServeMux()

	logDir := t.TempDir()
	logPath := filepath.Join(logDir, "agent.log")
	raw := strings.Join([]string{
		"line-1",
		"line-2",
		"line-3",
		"line-4",
	}, "\n") + "\n"
	if err := os.WriteFile(logPath, []byte(raw), 0644); err != nil {
		t.Fatalf("write agent log: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Logging.Dir = logDir

	RegisterInboundRoutes(&muxRegistrar{mux: mux}, msgBus, fakeHistory{}, &RuntimeOptions{
		Config: cfg,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/history/agent-log?tail=2", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET agent-log history status=%d want=200 body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Path  string   `json:"path"`
		Lines []string `json:"lines"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal agent-log response: %v", err)
	}
	if resp.Path != logPath {
		t.Fatalf("path=%q want=%q", resp.Path, logPath)
	}
	if len(resp.Lines) != 2 {
		t.Fatalf("lines len=%d want=2", len(resp.Lines))
	}
	if resp.Lines[0] != "line-3" || resp.Lines[1] != "line-4" {
		t.Fatalf("tail lines mismatch: %+v", resp.Lines)
	}
}
