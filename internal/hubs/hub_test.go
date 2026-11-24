package hubs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"yoomates/internal/auth"
)

func TestHubLifecycleAndBroadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock authenticated context
		ctx := auth.WithUserContext(r.Context(), &auth.UserContext{
			ID:       "usr_test_1",
			Username: "tester",
			Name:     "Test User",
		})
		r = r.WithContext(ctx)
		WSHandler(hub, nil)(w, r)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "?server_id=srv_1&channel_id=chan_general"

	// Connect Client 1
	conn1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial websocket: %v", err)
	}
	defer conn1.Close()

	// Wait for registration
	time.Sleep(50 * time.Millisecond)
	if hub.ActiveConnections() != 1 {
		t.Fatalf("expected 1 active connection, got %d", hub.ActiveConnections())
	}
	if !hub.IsUserOnline("usr_test_1") {
		t.Error("expected user usr_test_1 to be marked online")
	}

	// Connect Client 2 to same channel
	conn2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial client 2: %v", err)
	}
	defer conn2.Close()

	time.Sleep(50 * time.Millisecond)
	if hub.ActiveConnections() != 2 {
		t.Fatalf("expected 2 active connections, got %d", hub.ActiveConnections())
	}

	// Send test broadcast message to chan_general
	testMsg, err := NewWSMessage(
		TypeChat,
		"srv_1",
		"chan_general",
		"usr_test_1",
		"tester",
		ChatPayload{
			ID:      "msg_1",
			Content: "Hello YooMates Go Engine!",
		},
		"<div id='msg-1'>Hello YooMates Go Engine!</div>",
	)
	if err != nil {
		t.Fatalf("failed to create WS message: %v", err)
	}

	hub.BroadcastMessage(testMsg)

	// Helper to read until a specific message type is received
	readUntil := func(conn *websocket.Conn, targetType string) (*WSMessage, error) {
		for {
			_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			_, p, err := conn.ReadMessage()
			if err != nil {
				return nil, err
			}
			var msg WSMessage
			if err := json.Unmarshal(p, &msg); err != nil {
				return nil, err
			}
			if msg.Type == targetType {
				return &msg, nil
			}
		}
	}

	// Read chat message on Client 1
	received1, err := readUntil(conn1, TypeChat)
	if err != nil {
		t.Fatalf("client 1 failed to receive chat broadcast: %v", err)
	}
	if received1.ChannelID != "chan_general" {
		t.Errorf("expected channel chan_general, got %s", received1.ChannelID)
	}

	// Read chat message on Client 2
	received2, err := readUntil(conn2, TypeChat)
	if err != nil {
		t.Fatalf("client 2 failed to receive chat broadcast: %v", err)
	}
	if received2.HTML != "<div id='msg-1'>Hello YooMates Go Engine!</div>" {
		t.Errorf("unexpected HTML received: %s", received2.HTML)
	}

	// Close Client 1 and Client 2 and verify unregistration
	conn1.Close()
	conn2.Close()
	time.Sleep(100 * time.Millisecond)

	if hub.ActiveConnections() != 0 {
		t.Errorf("expected 0 active connections after disconnect, got %d", hub.ActiveConnections())
	}
}

func TestChannelIsolation(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WSHandler(hub, nil)(w, r)
	}))
	defer server.Close()

	baseURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Client in chan_alpha
	connAlpha, _, err := websocket.DefaultDialer.Dial(baseURL+"?channel_id=chan_alpha", nil)
	if err != nil {
		t.Fatalf("failed to connect alpha: %v", err)
	}
	defer connAlpha.Close()

	// Client in chan_beta
	connBeta, _, err := websocket.DefaultDialer.Dial(baseURL+"?channel_id=chan_beta", nil)
	if err != nil {
		t.Fatalf("failed to connect beta: %v", err)
	}
	defer connBeta.Close()

	time.Sleep(50 * time.Millisecond)

	// Broadcast exclusively to chan_alpha
	msgAlpha, _ := NewWSMessage(TypeChat, "", "chan_alpha", "u1", "alice", nil, "alpha-only")
	hub.BroadcastMessage(msgAlpha)

	// connAlpha should receive
	_ = connAlpha.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, pAlpha, err := connAlpha.ReadMessage()
	if err != nil {
		t.Fatalf("connAlpha expected to receive message: %v", err)
	}
	if !strings.Contains(string(pAlpha), "alpha-only") {
		t.Errorf("unexpected message on connAlpha: %s", string(pAlpha))
	}

	// connBeta should NOT receive and should timeout
	_ = connBeta.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	_, _, err = connBeta.ReadMessage()
	if err == nil {
		t.Error("connBeta should not have received a message targeted to chan_alpha")
	}
}
