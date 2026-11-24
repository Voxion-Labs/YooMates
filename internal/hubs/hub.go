package hubs

import (
	"encoding/json"
	"log"
	"sync"
)

// Hub maintains the set of active clients and broadcasts messages to the appropriate rooms
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Channel subscriptions: channelID -> map[*Client]bool
	channelClients map[string]map[*Client]bool

	// Server subscriptions: serverID -> map[*Client]bool
	serverClients map[string]map[*Client]bool

	// Presence tracking: userID -> active connection count
	presence map[string]int

	// Inbound messages to broadcast
	broadcast chan *WSMessage

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Mutex for subscription maps
	mu sync.RWMutex
}

// NewHub creates a new Hub instance
func NewHub() *Hub {
	return &Hub{
		clients:        make(map[*Client]bool),
		channelClients: make(map[string]map[*Client]bool),
		serverClients:  make(map[string]map[*Client]bool),
		presence:       make(map[string]int),
		broadcast:      make(chan *WSMessage, 256),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
	}
}

// Run begins listening for client registration, unregistration, and broadcast events
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true

			// Track server subscription
			if client.ServerID != "" {
				if _, ok := h.serverClients[client.ServerID]; !ok {
					h.serverClients[client.ServerID] = make(map[*Client]bool)
				}
				h.serverClients[client.ServerID][client] = true
			}

			// Track channel subscription
			if client.ChannelID != "" {
				if _, ok := h.channelClients[client.ChannelID]; !ok {
					h.channelClients[client.ChannelID] = make(map[*Client]bool)
				}
				h.channelClients[client.ChannelID][client] = true
			}

			// Update presence if user is authenticated
			if client.User != nil {
				h.presence[client.User.ID]++
				if h.presence[client.User.ID] == 1 {
					// User just came online
					go h.BroadcastPresence(client.ServerID, client.User.ID, client.User.Username, "online")
				}
			}
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)

				// Remove from server subscriptions
				if client.ServerID != "" && h.serverClients[client.ServerID] != nil {
					delete(h.serverClients[client.ServerID], client)
					if len(h.serverClients[client.ServerID]) == 0 {
						delete(h.serverClients, client.ServerID)
					}
				}

				// Remove from channel subscriptions
				if client.ChannelID != "" && h.channelClients[client.ChannelID] != nil {
					delete(h.channelClients[client.ChannelID], client)
					if len(h.channelClients[client.ChannelID]) == 0 {
						delete(h.channelClients, client.ChannelID)
					}
				}

				// Update presence
				if client.User != nil {
					h.presence[client.User.ID]--
					if h.presence[client.User.ID] <= 0 {
						delete(h.presence, client.User.ID)
						// User went offline
						go h.BroadcastPresence(client.ServerID, client.User.ID, client.User.Username, "offline")
					}
				}
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.routeBroadcast(msg)
		}
	}
}

// routeBroadcast distributes the message to the intended targets (channel, server, or global)
func (h *Hub) routeBroadcast(msg *WSMessage) {
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

	// Unsubscribe from previous channel if different
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
