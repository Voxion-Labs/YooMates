package voice

import (
	"errors"
	"fmt"
	"html"
	"strings"
	"sync"
	"time"
)

// VoicePeer represents an active participant in a voice channel
type VoicePeer struct {
	UserID   string    `json:"user_id"`
	Username string    `json:"username"`
	Initials string    `json:"initials"`
	Muted    bool      `json:"muted"`
	Deafened bool      `json:"deafened"`
	Speaking bool      `json:"speaking"`
	JoinedAt time.Time `json:"joined_at"`
}

// VoiceRoom maintains active participants in a voice channel
type VoiceRoom struct {
	ID       string                        `json:"id"`
	ServerID string                        `json:"server_id"`
	Name     string                        `json:"name"`
	Peers    map[string]*VoicePeer         `json:"peers"`
	Trackers map[string]*PeerBufferTracker `json:"-"`
	mu       sync.RWMutex
}

// Gateway orchestrates voice rooms, routing, and WebRTC/Audio peer states
type Gateway struct {
	rooms    map[string]*VoiceRoom
	userRoom map[string]string // user_id -> channel_id
	mu       sync.RWMutex
}

// NewGateway creates a thread-safe Voice Gateway instance
func NewGateway() *Gateway {
	return &Gateway{
		rooms:    make(map[string]*VoiceRoom),
		userRoom: make(map[string]string),
	}
}

// Join connects a user to a voice channel, removing them from any previous room
func (g *Gateway) Join(serverID, channelID, userID, username string) (*VoicePeer, error) {
	if channelID == "" || userID == "" {
		return nil, errors.New("channelID and userID must not be empty")
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	// 1. If user was in another room, remove them first
	if prevRoomID, exists := g.userRoom[userID]; exists && prevRoomID != channelID {
		if prevRoom, ok := g.rooms[prevRoomID]; ok {
			prevRoom.mu.Lock()
			delete(prevRoom.Peers, userID)
			if prevRoom.Trackers != nil {
				delete(prevRoom.Trackers, userID)
			}
			if len(prevRoom.Peers) == 0 {
				delete(g.rooms, prevRoomID)
			}
			prevRoom.mu.Unlock()
		}
		delete(g.userRoom, userID)
	}

	// 2. Get or create room
	room, exists := g.rooms[channelID]
	if !exists {
		room = &VoiceRoom{
			ID:       channelID,
			ServerID: serverID,
			Name:     "Lounge",
			Peers:    make(map[string]*VoicePeer),
			Trackers: make(map[string]*PeerBufferTracker),
		}
		g.rooms[channelID] = room
	}

	initials := "U"
	if len(username) > 0 {
		initials = strings.ToUpper(string(username[0]))
	}

	peer := &VoicePeer{
		UserID:   userID,
		Username: username,
		Initials: initials,
		Muted:    false,
		Deafened: false,
		Speaking: false,
		JoinedAt: time.Now().UTC(),
	}

	room.mu.Lock()
	room.Peers[userID] = peer
	if room.Trackers == nil {
		room.Trackers = make(map[string]*PeerBufferTracker)
	}
	room.Trackers[userID] = NewPeerBufferTracker()
	room.mu.Unlock()

	g.userRoom[userID] = channelID

	return peer, nil
}

// Leave removes a user from a voice channel
func (g *Gateway) Leave(channelID, userID string) (*VoicePeer, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	room, exists := g.rooms[channelID]
	if !exists {
		return nil, errors.New("voice room not found")
	}

	room.mu.Lock()
	peer, peerExists := room.Peers[userID]
	if peerExists {
		delete(room.Peers, userID)
		if room.Trackers != nil {
			delete(room.Trackers, userID)
		}
	}
	remaining := len(room.Peers)
	room.mu.Unlock()

	if remaining == 0 {
		delete(g.rooms, channelID)
	}
	delete(g.userRoom, userID)

	if !peerExists {
		return nil, errors.New("peer not in voice room")
	}

	return peer, nil
}

// SetMute toggles mute state for a user
func (g *Gateway) SetMute(channelID, userID string, muted bool) (*VoicePeer, error) {
	g.mu.RLock()
	room, exists := g.rooms[channelID]
	g.mu.RUnlock()

	if !exists {
		return nil, errors.New("voice room not found")
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	peer, ok := room.Peers[userID]
	if !ok {
		return nil, errors.New("peer not found in room")
	}
	peer.Muted = muted
	if muted {
		peer.Speaking = false
	}
	return peer, nil
}

// SetDeafen toggles deafen state for a user
func (g *Gateway) SetDeafen(channelID, userID string, deafened bool) (*VoicePeer, error) {
	g.mu.RLock()
	room, exists := g.rooms[channelID]
	g.mu.RUnlock()

	if !exists {
		return nil, errors.New("voice room not found")
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	peer, ok := room.Peers[userID]
	if !ok {
		return nil, errors.New("peer not found in room")
	}
	peer.Deafened = deafened
	if deafened {
		peer.Muted = true
		peer.Speaking = false
	}
	return peer, nil
}

// SetSpeaking updates real-time speaking indicator
func (g *Gateway) SetSpeaking(channelID, userID string, speaking bool) (*VoicePeer, error) {
	g.mu.RLock()
	room, exists := g.rooms[channelID]
	g.mu.RUnlock()

	if !exists {
		return nil, errors.New("voice room not found")
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	peer, ok := room.Peers[userID]
	if !ok {
		return nil, errors.New("peer not found in room")
	}
	if peer.Muted {
		peer.Speaking = false
	} else {
		peer.Speaking = speaking
	}
	return peer, nil
}

// GetRoomPeers returns all active peers in a voice room
func (g *Gateway) GetRoomPeers(channelID string) []*VoicePeer {
	g.mu.RLock()
	room, exists := g.rooms[channelID]
	g.mu.RUnlock()

	if !exists {
		return []*VoicePeer{}
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	peers := make([]*VoicePeer, 0, len(room.Peers))
	for _, p := range room.Peers {
		peers = append(peers, p)
	}
	return peers
}

// GetUserRoom returns the channel ID the user is currently connected to
func (g *Gateway) GetUserRoom(userID string) string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.userRoom[userID]
}

// RecordPeerPacket updates audio metrics for a peer in a voice room
func (g *Gateway) RecordPeerPacket(channelID, userID string, seq uint16, payloadBytes int, arrivalTimeMS, rtpTimestampMS int64) {
	g.mu.RLock()
	room, exists := g.rooms[channelID]
	g.mu.RUnlock()
	if !exists {
		return
	}

	room.mu.RLock()
	tracker, ok := room.Trackers[userID]
	room.mu.RUnlock()
	if ok && tracker != nil {
		tracker.RecordPacket(seq, payloadBytes, arrivalTimeMS, rtpTimestampMS)
	}
}

// GetPeerMetrics retrieves audio metrics for a peer
func (g *Gateway) GetPeerMetrics(channelID, userID string) PeerMetrics {
	g.mu.RLock()
	room, exists := g.rooms[channelID]
	g.mu.RUnlock()
	if !exists {
		return DefaultMetrics()
	}

	room.mu.RLock()
	tracker, ok := room.Trackers[userID]
	room.mu.RUnlock()
	if ok && tracker != nil {
		return tracker.GetMetrics()
	}
	return DefaultMetrics()
}

// RenderVoiceStatusOOB renders bottom-left Voice Connected panel
func RenderVoiceStatusOOB(channelID, channelName string, peer *VoicePeer) string {
	return RenderVoiceStatusWithMetricsOOB(channelID, channelName, peer, DefaultMetrics())
}

// RenderVoiceStatusWithMetricsOOB renders bottom-left Voice Connected panel with real-time stats
func RenderVoiceStatusWithMetricsOOB(channelID, channelName string, peer *VoicePeer, metrics PeerMetrics) string {
	micIcon := "🎤"
	if peer.Muted {
		micIcon = "🔇"
	}
	deafenIcon := "🎧"
	if peer.Deafened {
		deafenIcon = "🔕"
	}

	return fmt.Sprintf(`
<div id="voice-status-panel" class="voice-status-panel" hx-swap-oob="outerHTML">
  <div class="voice-status-header">
    <div class="voice-status-indicator">
      <span class="voice-ping-dot"></span>
      <span class="voice-status-text">Voice Connected</span>
    </div>
    <span class="voice-channel-name">%s</span>
  </div>
  <div class="voice-stats-bar">
    <span class="voice-stat-pill" title="Jitter buffer delay">⏱️ %dms</span>
    <span class="voice-stat-pill" title="Packet loss">🛡️ %.1f%%%% loss</span>
    <span class="voice-stat-pill" title="Opus fullband audio">🔊 48kHz</span>
  </div>
  <div class="voice-actions">
    <button class="voice-ctrl-btn %s" title="Toggle Mute" ws-send hx-vals='{"type":"voice_mute","channel_id":"%s","muted":%t}'>
      %s
    </button>
    <button class="voice-ctrl-btn %s" title="Toggle Deafen" ws-send hx-vals='{"type":"voice_deafen","channel_id":"%s","deafened":%t}'>
      %s
    </button>
    <button class="voice-ctrl-btn voice-disconnect-btn" title="Disconnect" ws-send hx-vals='{"type":"voice_leave","channel_id":"%s"}'>
      📞
    </button>
  </div>
</div>
`, html.EscapeString(channelName), metrics.TargetDelayMS, metrics.PacketLossRate*100.0,
		map[bool]string{true: "active", false: ""}[peer.Muted], channelID, !peer.Muted, micIcon,
		map[bool]string{true: "active", false: ""}[peer.Deafened], channelID, !peer.Deafened, deafenIcon,
		channelID)
}

// RenderVoiceDisconnectOOB clears the voice panel on disconnection
func RenderVoiceDisconnectOOB() string {
	return `<div id="voice-status-panel" class="voice-status-panel" style="display: none;" hx-swap-oob="outerHTML"></div>`
}

// RenderVoiceParticipantsOOB renders speaking grid/list of voice participants in the channel
func RenderVoiceParticipantsOOB(channelID string, peers []*VoicePeer) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<div id="voice-participants-%s" class="voice-participants-overlay" hx-swap-oob="innerHTML">`, channelID))
	for _, p := range peers {
		speakingClass := ""
		if p.Speaking {
			speakingClass = "speaking"
		}
		statusBadge := ""
		if p.Deafened {
			statusBadge = `<span class="peer-badge">🔕</span>`
		} else if p.Muted {
			statusBadge = `<span class="peer-badge">🔇</span>`
		}

		sb.WriteString(fmt.Sprintf(`
    <div class="voice-peer-card %s" id="voice-peer-%s">
      <div class="peer-avatar %s">%s%s</div>
      <span class="peer-name">%s</span>
    </div>
`, speakingClass, p.UserID, speakingClass, p.Initials, statusBadge, html.EscapeString(p.Username)))
	}
	sb.WriteString(`</div>`)
	return sb.String()
}
