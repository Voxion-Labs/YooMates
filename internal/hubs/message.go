package hubs

import (
	"encoding/json"
	"time"
)

// Message types for real-time Hubs WebSocket communication
const (
	TypeChat          = "chat"
	TypeTyping        = "typing"
	TypePresence      = "presence"
	TypeReaction      = "reaction"
	TypeEdit          = "edit"
	TypeDelete        = "delete"
	TypeSubscribe     = "subscribe"
	TypeUnsubscribe   = "unsubscribe"
	TypeVoiceJoin     = "voice_join"
	TypeVoiceLeave    = "voice_leave"
	TypeVoiceMute     = "voice_mute"
	TypeVoiceDeafen   = "voice_deafen"
	TypeVoiceSpeaking = "voice_speaking"
	TypeVoiceState    = "voice_state"
	TypeError         = "error"
)

// WSMessage represents a real-time event frame.
// Supports dual-mode: HTML for HTMX (hx-ext="ws") swaps and structured JSON for WASM/dumb-pipe.
type WSMessage struct {
	Type       string          `json:"type"`
	ServerID   string          `json:"server_id,omitempty"`
	ChannelID  string          `json:"channel_id,omitempty"`
	UserID     string          `json:"user_id,omitempty"`
	Username   string          `json:"username,omitempty"`
	Content    string          `json:"content,omitempty"`
	FileURL    string          `json:"file_url,omitempty"`
	FileName   string          `json:"file_name,omitempty"`
	FileType   string          `json:"file_type,omitempty"`
	FileSize   int             `json:"file_size,omitempty"`
	MessageID  string          `json:"message_id,omitempty"`
	Emoji      string          `json:"emoji,omitempty"`
	IsTyping   bool            `json:"is_typing,omitempty"`
	Muted      bool            `json:"muted,omitempty"`
	Deafened   bool            `json:"deafened,omitempty"`
	IsSpeaking bool            `json:"is_speaking,omitempty"`
	Payload    json.RawMessage `json:"payload,omitempty"`
	HTML       string          `json:"html,omitempty"`
	Timestamp  string          `json:"timestamp"`
}

// ChatPayload represents the structured data for a chat message
type ChatPayload struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	FileURL   string `json:"file_url,omitempty"`
	FileName  string `json:"file_name,omitempty"`
	FileType  string `json:"file_type,omitempty"`
	FileSize  int    `json:"file_size,omitempty"`
	IsEdited  bool   `json:"is_edited,omitempty"`
	CreatedAt string `json:"created_at"`
}

// TypingPayload represents an ephemeral typing event
type TypingPayload struct {
	IsTyping bool `json:"is_typing"`
}

// PresencePayload represents online/offline status
type PresencePayload struct {
	Status string `json:"status"` // "online", "idle", "offline"
}

// ReactionPayload represents an emoji reaction event
type ReactionPayload struct {
	MessageID string `json:"message_id"`
	Emoji     string `json:"emoji"`
}

// EditPayload represents an edit message request
type EditPayload struct {
	MessageID string `json:"message_id"`
	Content   string `json:"content"`
}

// DeletePayload represents a delete message request
type DeletePayload struct {
	MessageID string `json:"message_id"`
}

// NewWSMessage creates a timestamped WSMessage
func NewWSMessage(msgType, serverID, channelID, userID, username string, payload interface{}, html string) (*WSMessage, error) {
	var rawPayload json.RawMessage
	if payload != nil {
		bytes, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		rawPayload = bytes
	}

	return &WSMessage{
		Type:      msgType,
		ServerID:  serverID,
		ChannelID: channelID,
		UserID:    userID,
		Username:  username,
		Payload:   rawPayload,
		HTML:      html,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}, nil
}
