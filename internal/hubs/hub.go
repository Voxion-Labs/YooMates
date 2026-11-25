package hubs

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"
)

// Hub maintains active clients, routes real-time broadcasts, and handles message lifecycle
type Hub struct {
	clients        map[*Client]bool
	channelClients map[string]map[*Client]bool
	serverClients  map[string]map[*Client]bool
	presence       map[string]int
	broadcast      chan *WSMessage
	register       chan *Client
	unregister     chan *Client
	repo           *Repository
	mu             sync.RWMutex
}

// NewHub creates a new Hub instance with an optional database repository
func NewHub(repo *Repository) *Hub {
	return &Hub{
		clients:        make(map[*Client]bool),
		channelClients: make(map[string]map[*Client]bool),
		serverClients:  make(map[string]map[*Client]bool),
		presence:       make(map[string]int),
		broadcast:      make(chan *WSMessage, 256),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		repo:           repo,
	}
}

// Run begins listening for client registration, unregistration, and broadcast events
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true

			if client.ServerID != "" {
				if _, ok := h.serverClients[client.ServerID]; !ok {
					h.serverClients[client.ServerID] = make(map[*Client]bool)
				}
				h.serverClients[client.ServerID][client] = true
			}

			if client.ChannelID != "" {
				if _, ok := h.channelClients[client.ChannelID]; !ok {
					h.channelClients[client.ChannelID] = make(map[*Client]bool)
				}
				h.channelClients[client.ChannelID][client] = true
			}

			if client.User != nil {
				h.presence[client.User.ID]++
				if h.presence[client.User.ID] == 1 {
					go h.BroadcastPresence(client.ServerID, client.User.ID, client.User.Username, "online")
				}
			}
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)

				if client.ServerID != "" && h.serverClients[client.ServerID] != nil {
					delete(h.serverClients[client.ServerID], client)
					if len(h.serverClients[client.ServerID]) == 0 {
						delete(h.serverClients, client.ServerID)
					}
				}

				if client.ChannelID != "" && h.channelClients[client.ChannelID] != nil {
					delete(h.channelClients[client.ChannelID], client)
					if len(h.channelClients[client.ChannelID]) == 0 {
						delete(h.channelClients, client.ChannelID)
					}
				}

				if client.User != nil {
					h.presence[client.User.ID]--
					if h.presence[client.User.ID] <= 0 {
						delete(h.presence, client.User.ID)
						go h.BroadcastPresence(client.ServerID, client.User.ID, client.User.Username, "offline")
					}
				}
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.processAndRoute(msg)
		}
	}
}

// processAndRoute handles business logic (persistence & HTML generation) before distributing frames
func (h *Hub) processAndRoute(msg *WSMessage) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch msg.Type {
	case TypeChat:
		authorName := msg.Username
		if authorName == "" {
			authorName = "Team Member"
		}
		authorID := msg.UserID
		if authorID == "" {
			authorID = "guest"
		}

		if h.repo != nil && msg.Content != "" {
			item, err := h.repo.SaveMessage(ctx, authorID, authorName, msg.ChannelID, msg.ServerID, msg.Content)
			if err == nil && item != nil {
				msg.MessageID = item.ID
				msg.HTML = RenderMessageOOB(item)
			}
		} else if msg.Content != "" && msg.HTML == "" {
			item := &MessageItem{
				ID:            "msg_" + time.Now().Format("150405"),
				AuthorName:    authorName,
				Initials:      "U",
				Content:       msg.Content,
				FormattedTime: "Today at " + time.Now().Format("15:04"),
			}
			msg.HTML = RenderMessageOOB(item)
		}

	case TypeTyping:
		msg.HTML = RenderTypingOOB(msg.Username, msg.IsTyping)

	case TypeEdit:
		if h.repo != nil && msg.MessageID != "" && msg.Content != "" {
			item, err := h.repo.EditMessage(ctx, msg.MessageID, msg.UserID, msg.Content)
			if err == nil && item != nil {
				item.AuthorName = msg.Username
				msg.HTML = RenderMessageEditOOB(item)
			}
		}

	case TypeDelete:
		if h.repo != nil && msg.MessageID != "" {
			_ = h.repo.DeleteMessage(ctx, msg.MessageID, msg.UserID)
		}
		msg.HTML = RenderMessageDeleteOOB(msg.MessageID)

	case TypeReaction:
		if h.repo != nil && msg.MessageID != "" && msg.Emoji != "" {
			reactions, err := h.repo.ToggleReaction(ctx, msg.MessageID, msg.UserID, msg.ServerID, msg.Emoji)
			if err == nil {
				msg.HTML = RenderReactionsOOB(msg.MessageID, reactions)
			}
		}
	}

	h.dispatch(msg)
}

// dispatch delivers the prepared frame to clients
func (h *Hub) dispatch(msg *WSMessage) {
	bytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[hub-error] Failed to marshal broadcast message: %v\n", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	// 1. Channel-scoped message
	if msg.ChannelID != "" {
		if clients, ok := h.channelClients[msg.ChannelID]; ok {
			for client := range clients {
				select {
				case client.Send <- bytes:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
		}
		return
	}

	// 2. Server-scoped message
	if msg.ServerID != "" {
		if clients, ok := h.serverClients[msg.ServerID]; ok {
			for client := range clients {
				select {
				case client.Send <- bytes:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
		}
		return
	}

	// 3. Global broadcast
	for client := range h.clients {
		select {
		case client.Send <- bytes:
		default:
			close(client.Send)
			delete(h.clients, client)
		}
	}
}

// SubscribeChannel moves or adds client to a specific channel
func (h *Hub) SubscribeChannel(client *Client, channelID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client.ChannelID != "" && client.ChannelID != channelID {
		if clients, ok := h.channelClients[client.ChannelID]; ok {
			delete(clients, client)
		}
	}

	client.ChannelID = channelID
	if _, ok := h.channelClients[channelID]; !ok {
		h.channelClients[channelID] = make(map[*Client]bool)
	}
	h.channelClients[channelID][client] = true
}

// UnsubscribeChannel removes client from a specific channel
func (h *Hub) UnsubscribeChannel(client *Client, channelID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.channelClients[channelID]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.channelClients, channelID)
		}
	}
	if client.ChannelID == channelID {
		client.ChannelID = ""
	}
}

// BroadcastMessage sends a message to the hub's broadcast channel
func (h *Hub) BroadcastMessage(msg *WSMessage) {
	h.broadcast <- msg
}

// BroadcastPresence sends an online/offline status update
func (h *Hub) BroadcastPresence(serverID, userID, username, status string) {
	msg, err := NewWSMessage(
		TypePresence,
		serverID,
		"",
		userID,
		username,
		PresencePayload{Status: status},
		"",
	)
	if err == nil {
		h.BroadcastMessage(msg)
	}
}

// IsUserOnline returns whether the given user is currently connected
func (h *Hub) IsUserOnline(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.presence[userID] > 0
}

// ActiveConnections returns the total number of connected clients
func (h *Hub) ActiveConnections() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
