package voice

import (
	"strings"
	"testing"
)

func TestPeerBufferTracker_JitterAndLoss(t *testing.T) {
	tracker := NewPeerBufferTracker()

	// Packet 1: arrival 20ms, timestamp 0
	tracker.RecordPacket(100, 160, 20, 0)
	m := tracker.GetMetrics()
	if m.PacketsReceived != 1 {
		t.Fatalf("expected 1 packet received, got %d", m.PacketsReceived)
	}

	// Packet 2: arrival 40ms, timestamp 20 (transit = 20, identical -> jitter remains low)
	tracker.RecordPacket(101, 160, 40, 20)
	m = tracker.GetMetrics()
	if m.PacketsReceived != 2 {
		t.Fatalf("expected 2 packets received, got %d", m.PacketsReceived)
	}

	// Packet 4: simulated gap (packet 102 lost), arrival 90ms, timestamp 60
	tracker.RecordPacket(103, 160, 90, 60)
	m = tracker.GetMetrics()
	if m.PacketsReceived != 3 {
		t.Fatalf("expected 3 packets received, got %d", m.PacketsReceived)
	}
	if m.PacketLossRate <= 0.0 {
		t.Errorf("expected packet loss > 0, got %f", m.PacketLossRate)
	}
	if m.JitterMS <= 0.0 {
		t.Errorf("expected jitter > 0, got %f", m.JitterMS)
	}

	// Record PLC and Silence
	tracker.RecordPLC()
	tracker.SetSilenceSuppressed(true)
	m = tracker.GetMetrics()
	if m.PLCFrames != 1 {
		t.Errorf("expected 1 PLC frame, got %d", m.PLCFrames)
	}
	if !m.SilenceSuppressed {
		t.Errorf("expected silence suppressed to be true")
	}
}

func TestGateway_PeerMetricsIntegration(t *testing.T) {
	gw := NewGateway()
	peer, err := gw.Join("srv_main", "chan_voice_lounge", "user_1", "Alice")
	if err != nil {
		t.Fatalf("unexpected error joining: %v", err)
	}
	if peer == nil {
		t.Fatalf("expected peer to be non-nil")
	}

	// Record packets through Gateway
	gw.RecordPeerPacket("chan_voice_lounge", "user_1", 1, 160, 20, 0)
	gw.RecordPeerPacket("chan_voice_lounge", "user_1", 2, 160, 40, 20)

	metrics := gw.GetPeerMetrics("chan_voice_lounge", "user_1")
	if metrics.PacketsReceived != 2 {
		t.Errorf("expected 2 packets received in metrics, got %d", metrics.PacketsReceived)
	}

	// Render status with metrics
	html := RenderVoiceStatusWithMetricsOOB("chan_voice_lounge", "Lounge", peer, metrics)
	if !strings.Contains(html, "voice-stats-bar") {
		t.Errorf("expected HTML to contain voice-stats-bar")
	}
	if !strings.Contains(html, "48kHz") {
		t.Errorf("expected HTML to contain 48kHz audio badge")
	}
}
