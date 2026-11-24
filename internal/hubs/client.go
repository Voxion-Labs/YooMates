package hubs

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"yoomates/internal/auth"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 64 * 1024 // 64 KB
)

// Client represents a connected WebSocket client session
type Client struct {
	Hub       *Hub
	Conn      *websocket.Conn
	Send      chan []byte
	User      *auth.UserContext
	ServerID  string
	ChannelID string
}

// ReadPump pumps messages from the websocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		_ = c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[ws-client] Unexpected close: %v\n", err)
			}
			break
		}

		var wsMsg WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			// If not JSON, it could be raw text from an HTMX form submission
			wsMsg = WSMessage{
				Type:      TypeChat,
				ServerID:  c.ServerID,
				ChannelID: c.ChannelID,
				HTML:      string(message),
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			}
		}

		// Ensure user attribution is authentic
		if c.User != nil {
			wsMsg.UserID = c.User.ID
			wsMsg.Username = c.User.Username
		}

		if wsMsg.ServerID == "" {
			wsMsg.ServerID = c.ServerID
		}
		if wsMsg.ChannelID == "" {
			wsMsg.ChannelID = c.ChannelID
		}

		// Process message routing
		switch wsMsg.Type {
		case TypeSubscribe:
			if wsMsg.ChannelID != "" {
				c.Hub.SubscribeChannel(c, wsMsg.ChannelID)
			}
		case TypeUnsubscribe:
			if wsMsg.ChannelID != "" {
				c.Hub.UnsubscribeChannel(c, wsMsg.ChannelID)
			}
		default:
			c.Hub.broadcast <- &wsMsg
		}
	}
}

// WritePump pumps messages from the hub to the websocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

			// Add queued chat messages to the current websocket frame
			n := len(c.Send)
			for i := 0; i < n; i++ {
				_, _ = w.Write([]byte{'\n'})
				_, _ = w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
