package adminapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/providers"
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
	RegisterInboundRoutes(&muxRegistrar{mux: mux}, msgBus, nil)

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
	RegisterInboundRoutes(&muxRegistrar{mux: mux}, msgBus, fakeHistory{})

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
	RegisterInboundRoutes(&muxRegistrar{mux: mux}, msgBus, fakeHistory{})

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

func TestDashboardRoute(t *testing.T) {
	msgBus := bus.NewMessageBus()
	mux := http.NewServeMux()
	RegisterInboundRoutes(&muxRegistrar{mux: mux}, msgBus, fakeHistory{})

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
