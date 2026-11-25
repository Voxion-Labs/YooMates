package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
