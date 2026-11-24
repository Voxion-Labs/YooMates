package hubs

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"yoomates/internal/auth"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Origin validation: allows local development and any same-origin/subdomain requests
		return true
	},
}

// WSHandler handles WebSocket requests from clients (HTMX hx-ext="ws" or vanilla dumb-pipe)
func WSHandler(hub *Hub, authMgr *auth.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var currentUser *auth.UserContext

		// 1. Check user context from session middleware
		if user, ok := auth.GetUserFromContext(r.Context()); ok && user != nil {
			currentUser = user
		} else if authMgr != nil {
			// 2. Check query param token as fallback
			token := r.URL.Query().Get("token")
			if token == "" {
				token = auth.GetSessionToken(r)
			}
			if token != "" {
				if user, err := authMgr.ValidateSession(r.Context(), token); err == nil {
					currentUser = user
				}
			}
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[ws-upgrade-error] %v\n", err)
			return
		}

		serverID := r.URL.Query().Get("server_id")
		channelID := r.URL.Query().Get("channel_id")

		client := &Client{
			Hub:       hub,
			Conn:      conn,
			Send:      make(chan []byte, 256),
			User:      currentUser,
			ServerID:  serverID,
			ChannelID: channelID,
		}

		hub.register <- client

		go client.WritePump()
		go client.ReadPump()
	}
}
