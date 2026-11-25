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

func TestHubLifecycleAndRealtimeInteractions(t *testing.T) {
	repo := NewRepository(nil)
	hub := NewHub(repo)
	go hub.Run()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	// Connect Client 2
	conn2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial client 2: %v", err)
	}
	defer conn2.Close()

	time.Sleep(50 * time.Millisecond)
	if hub.ActiveConnections() != 2 {
		t.Fatalf("expected 2 active connections, got %d", hub.ActiveConnections())
	}

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

	// 1. Test Live Chat Message & OOB Render
	chatMsg := &WSMessage{
		Type:      TypeChat,
		ServerID:  "srv_1",
		ChannelID: "chan_general",
		UserID:    "usr_test_1",
		Username:  "tester",
		Content:   "Testing live HTMX WS swap",
	}
	hub.BroadcastMessage(chatMsg)

	receivedChat, err := readUntil(conn2, TypeChat)
	if err != nil {
		t.Fatalf("failed to receive chat: %v", err)
	}
	if !strings.Contains(receivedChat.HTML, `hx-swap-oob="beforeend"`) {
		t.Errorf("expected HTML to contain hx-swap-oob beforeend, got %s", receivedChat.HTML)
	}
	if !strings.Contains(receivedChat.HTML, "Testing live HTMX WS swap") {
		t.Errorf("expected HTML to contain message content, got %s", receivedChat.HTML)
	}

	// 2. Test Typing Indicator
	typingMsg := &WSMessage{
		Type:      TypeTyping,
		ChannelID: "chan_general",
		Username:  "tester",
		IsTyping:  true,
	}
	hub.BroadcastMessage(typingMsg)

	receivedTyping, err := readUntil(conn2, TypeTyping)
	if err != nil {
		t.Fatalf("failed to receive typing: %v", err)
	}
	if !strings.Contains(receivedTyping.HTML, "tester is typing...") {
		t.Errorf("expected typing HTML to contain username, got %s", receivedTyping.HTML)
	}

	// 3. Test Emoji Reaction
	reactMsg := &WSMessage{
		Type:      TypeReaction,
		ChannelID: "chan_general",
		ServerID:  "srv_1",
		UserID:    "usr_test_1",
		MessageID: "msg_123",
		Emoji:     "🔥",
	}
	hub.BroadcastMessage(reactMsg)

	receivedReact, err := readUntil(conn2, TypeReaction)
	if err != nil {
		t.Fatalf("failed to receive reaction: %v", err)
	}
	if !strings.Contains(receivedReact.HTML, "🔥") {
		t.Errorf("expected reaction HTML to contain emoji, got %s", receivedReact.HTML)
	}

	// 4. Test Soft-Delete Message
	deleteMsg := &WSMessage{
		Type:      TypeDelete,
		ChannelID: "chan_general",
		MessageID: "msg_123",
		UserID:    "usr_test_1",
	}
	hub.BroadcastMessage(deleteMsg)

	receivedDelete, err := readUntil(conn2, TypeDelete)
	if err != nil {
		t.Fatalf("failed to receive delete: %v", err)
	}
	if !strings.Contains(receivedDelete.HTML, "[This message was deleted]") {
		t.Errorf("expected delete HTML to indicate deletion, got %s", receivedDelete.HTML)
	}
}

func TestChannelIsolation(t *testing.T) {
	hub := NewHub(nil)
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
	msgAlpha := &WSMessage{
		Type:      TypeChat,
		ChannelID: "chan_alpha",
		UserID:    "u1",
		Username:  "alice",
		Content:   "alpha-only-secret",
	}
	hub.BroadcastMessage(msgAlpha)

	_ = connAlpha.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, pAlpha, err := connAlpha.ReadMessage()
	if err != nil {
		t.Fatalf("connAlpha expected to receive message: %v", err)
	}
	if !strings.Contains(string(pAlpha), "alpha-only-secret") {
		t.Errorf("unexpected message on connAlpha: %s", string(pAlpha))
	}

	// connBeta should NOT receive and should timeout
	_ = connBeta.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	_, _, err = connBeta.ReadMessage()
	if err == nil {
		t.Error("connBeta should not have received a message targeted to chan_alpha")
	}
}
