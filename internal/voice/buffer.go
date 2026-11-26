package voice

import (
	"math"
	"sync"
	"time"
)

// PeerMetrics holds network and buffer performance metrics for an active voice participant
type PeerMetrics struct {
	JitterMS          float64   `json:"jitter_ms"`
	PacketLossRate    float64   `json:"packet_loss_rate"`
	PacketsReceived   uint32    `json:"packets_received"`
	PacketsPlayed     uint32    `json:"packets_played"`
	PLCFrames         uint32    `json:"plc_frames"`
	BitrateKbps       float64   `json:"bitrate_kbps"`
	TargetDelayMS     uint32    `json:"target_delay_ms"`
	CurrentDepthMS    uint32    `json:"current_depth_ms"`
	SilenceSuppressed bool      `json:"silence_suppressed"`
	LastUpdate        time.Time `json:"last_update"`
}

// DefaultMetrics returns baseline metrics for a newly connected voice peer
func DefaultMetrics() PeerMetrics {
	return PeerMetrics{
		JitterMS:          1.2,
		PacketLossRate:    0.0,
		PacketsReceived:   0,
		PacketsPlayed:     0,
		PLCFrames:         0,
		BitrateKbps:       64.0,
		TargetDelayMS:     40,
		CurrentDepthMS:    20,
		SilenceSuppressed: false,
		LastUpdate:        time.Now().UTC(),
	}
}

// PeerBufferTracker tracks audio packet flow and computes buffer statistics for a voice peer
type PeerBufferTracker struct {
	mu           sync.RWMutex
	metrics      PeerMetrics
	lastTransit  int64
	lastSeq      uint16
	expectedPkts uint32
	initialized  bool
	bytesWindow  uint64
	windowStart  time.Time
}

// NewPeerBufferTracker creates an audio buffer and jitter tracker
func NewPeerBufferTracker() *PeerBufferTracker {
	return &PeerBufferTracker{
		metrics:     DefaultMetrics(),
		windowStart: time.Now(),
	}
}

// RecordPacket updates interarrival jitter and packet loss calculations per RFC 3550
func (t *PeerBufferTracker) RecordPacket(seq uint16, payloadBytes int, arrivalTimeMS int64, rtpTimestampMS int64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.metrics.PacketsReceived++
	t.metrics.PacketsPlayed++
	t.bytesWindow += uint64(payloadBytes)

	if !t.initialized {
		t.lastSeq = seq
		t.lastTransit = arrivalTimeMS - rtpTimestampMS
		t.expectedPkts = 1
		t.initialized = true
		t.metrics.LastUpdate = time.Now().UTC()
		return
	}

	diff := int16(seq - t.lastSeq)
	if diff > 0 {
		t.expectedPkts += uint32(diff)
		t.lastSeq = seq
	}

	// RFC 3550 Interarrival Jitter calculation: J(i) = J(i-1) + (|D(i-1,i)| - J(i-1))/16
	transit := arrivalTimeMS - rtpTimestampMS
	d := math.Abs(float64(transit - t.lastTransit))
	t.lastTransit = transit
	t.metrics.JitterMS += (d - t.metrics.JitterMS) / 16.0

	// Adaptive target delay computation: clamp(Jitter * 2.5 + 20, 20ms, 120ms)
	target := uint32(t.metrics.JitterMS*2.5 + 20.0)
	if target < 20 {
		target = 20
	} else if target > 120 {
		target = 120
	}
	t.metrics.TargetDelayMS = target

	// Packet loss rate calculation
	if t.expectedPkts > 0 {
		lost := int32(t.expectedPkts) - int32(t.metrics.PacketsReceived)
		if lost < 0 {
			t.metrics.PacketLossRate = 0.0
		} else {
			t.metrics.PacketLossRate = float64(lost) / float64(t.expectedPkts)
		}
	}

	// Rolling bitrate estimation (reset every 1 second)
	now := time.Now()
	elapsed := now.Sub(t.windowStart).Seconds()
	if elapsed >= 1.0 {
		t.metrics.BitrateKbps = (float64(t.bytesWindow) * 8.0) / (elapsed * 1000.0)
		t.bytesWindow = 0
		t.windowStart = now
	}

	t.metrics.LastUpdate = now.UTC()
}

// RecordPLC increments the packet loss concealment counter
func (t *PeerBufferTracker) RecordPLC() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.metrics.PLCFrames++
}

// SetSilenceSuppressed updates discontinuous transmission / silence suppression status
func (t *PeerBufferTracker) SetSilenceSuppressed(suppressed bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.metrics.SilenceSuppressed = suppressed
}

// GetMetrics returns a snapshot of peer audio metrics
func (t *PeerBufferTracker) GetMetrics() PeerMetrics {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.metrics
}
