package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"yoomates/internal/config"
)

func TestHealthHandler(t *testing.T) {
	cfg := config.Load()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	handler := SetupRoutes(cfg, nil, nil, nil, nil)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", rr.Code)
	}

	var res HealthResponse
	if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	if res.Status != "ok" {
		t.Errorf("expected status 'ok', got %s", res.Status)
	}

	if res.Engine != "yoomates-go" {
		t.Errorf("expected engine 'yoomates-go', got %s", res.Engine)
	}
}

func TestRootRedirect(t *testing.T) {
	cfg := config.Load()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler := SetupRoutes(cfg, nil, nil, nil, nil)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected redirect 307, got %d", rr.Code)
	}

	location := rr.Header().Get("Location")
	if location != "/hubs" {
		t.Errorf("expected redirect location /hubs, got %s", location)
	}
}

func TestServiceWorkerEndpoint(t *testing.T) {
	cfg := config.Load()
	req := httptest.NewRequest(http.MethodGet, "/sw.js", nil)
	rr := httptest.NewRecorder()

	handler := SetupRoutes(cfg, nil, nil, nil, nil)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK for /sw.js, got %d", rr.Code)
	}

	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "javascript") {
		t.Errorf("expected Content-Type application/javascript, got %s", ct)
	}

	allowed := rr.Header().Get("Service-Worker-Allowed")
	if allowed != "/" {
		t.Errorf("expected Service-Worker-Allowed header '/', got %s", allowed)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "yoomates-v1") {
		t.Errorf("expected body to contain cache name 'yoomates-v1', got %s", body)
	}
}
