package hubs

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestViewEngineRenderHubs(t *testing.T) {
	templatesDir := filepath.Join("..", "..", "web", "templates")
	engine, err := NewViewEngine(templatesDir, nil)
	if err != nil {
		t.Fatalf("failed to create ViewEngine: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/hubs", nil)
	rr := httptest.NewRecorder()

	engine.HubsHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "hubs-shell") {
		t.Error("expected body to contain hubs-shell class")
	}
	if !strings.Contains(body, "server-rail") {
		t.Error("expected body to contain server-rail")
	}
	if !strings.Contains(body, "channel-sidebar") {
		t.Error("expected body to contain channel-sidebar")
	}
	if !strings.Contains(body, "chat-window") {
		t.Error("expected body to contain chat-window")
	}
	if !strings.Contains(body, "member-sidebar") {
		t.Error("expected body to contain member-sidebar")
	}
	if !strings.Contains(body, "ws-connect=\"/ws?") {
		t.Error("expected body to contain WebSocket connection attribute")
	}
}
