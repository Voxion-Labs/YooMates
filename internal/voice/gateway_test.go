package voice

import (
	"strings"
	"testing"
)

func TestGateway_JoinAndLeave(t *testing.T) {
	gw := NewGateway()

	// 1. Join room
	peer, err := gw.Join("srv_1", "chan_voice_1", "user_1", "Alice")
	if err != nil {
		t.Fatalf("failed to join room: %v", err)
	}

	if peer.UserID != "user_1" || peer.Username != "Alice" || peer.Initials != "A" {
		t.Errorf("unexpected peer attributes: %+v", peer)
	}

	peers := gw.GetRoomPeers("chan_voice_1")
	if len(peers) != 1 {
		t.Fatalf("expected 1 peer, got %d", len(peers))
	}

	// 2. Leave room
	leftPeer, err := gw.Leave("chan_voice_1", "user_1")
	if err != nil {
		t.Fatalf("failed to leave room: %v", err)
	}
	if leftPeer.UserID != "user_1" {
		t.Errorf("expected left peer user_1, got %s", leftPeer.UserID)
	}

	peersAfter := gw.GetRoomPeers("chan_voice_1")
	if len(peersAfter) != 0 {
		t.Errorf("expected 0 peers after leaving, got %d", len(peersAfter))
	}
}

func TestGateway_SwitchRoom(t *testing.T) {
	gw := NewGateway()

	_, _ = gw.Join("srv_1", "chan_room_a", "user_1", "Alice")
	if gw.GetUserRoom("user_1") != "chan_room_a" {
		t.Errorf("expected user in chan_room_a")
	}

	// Joining chan_room_b should remove user from chan_room_a
	_, _ = gw.Join("srv_1", "chan_room_b", "user_1", "Alice")
	if gw.GetUserRoom("user_1") != "chan_room_b" {
		t.Errorf("expected user in chan_room_b")
	}

	if len(gw.GetRoomPeers("chan_room_a")) != 0 {
		t.Errorf("expected chan_room_a to be empty after room switch")
	}
	if len(gw.GetRoomPeers("chan_room_b")) != 1 {
		t.Errorf("expected chan_room_b to have 1 peer")
	}
}

func TestGateway_MuteAndDeafen(t *testing.T) {
	gw := NewGateway()
	_, _ = gw.Join("srv_1", "chan_v1", "user_bob", "Bob")

	// 1. Mute
	p, err := gw.SetMute("chan_v1", "user_bob", true)
	if err != nil || !p.Muted {
		t.Fatalf("expected peer to be muted, got %v", err)
	}

	// Speaking while muted must stay false
	p, _ = gw.SetSpeaking("chan_v1", "user_bob", true)
	if p.Speaking {
		t.Errorf("peer should not be speaking while muted")
	}

	// 2. Deafen
	p, err = gw.SetDeafen("chan_v1", "user_bob", true)
	if err != nil || !p.Deafened || !p.Muted {
		t.Fatalf("expected peer to be deafened and muted, got %v", err)
	}
}

func TestGateway_RenderOOB(t *testing.T) {
	peer := &VoicePeer{
		UserID:   "u1",
		Username: "Alice",
		Initials: "A",
		Muted:    false,
		Deafened: false,
		Speaking: true,
	}

	statusHTML := RenderVoiceStatusOOB("chan_v1", "Lounge", peer)
	if !strings.Contains(statusHTML, "Voice Connected") || !strings.Contains(statusHTML, "Lounge") {
		t.Errorf("expected status HTML to contain 'Voice Connected' and 'Lounge', got: %s", statusHTML)
	}

	disconnectHTML := RenderVoiceDisconnectOOB()
	if !strings.Contains(disconnectHTML, "display: none") {
		t.Errorf("expected disconnect HTML to hide panel, got: %s", disconnectHTML)
	}

	participantsHTML := RenderVoiceParticipantsOOB("chan_v1", []*VoicePeer{peer})
	if !strings.Contains(participantsHTML, "speaking") || !strings.Contains(participantsHTML, "Alice") {
		t.Errorf("expected participants HTML to contain 'speaking' and 'Alice', got: %s", participantsHTML)
	}
}
