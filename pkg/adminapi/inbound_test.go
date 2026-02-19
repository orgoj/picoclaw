package adminapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
)

type muxRegistrar struct {
	mux *http.ServeMux
}

func (m *muxRegistrar) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	m.mux.HandleFunc(pattern, handler)
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
	RegisterInboundRoutes(&muxRegistrar{mux: mux}, msgBus)

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
