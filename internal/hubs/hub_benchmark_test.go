package hubs

import (
	"fmt"
	"testing"
	"time"

	"yoomates/internal/auth"
)

func BenchmarkHubBroadcast_100Clients(b *testing.B) {
	benchmarkHubBroadcast(b, 100)
}

func BenchmarkHubBroadcast_500Clients(b *testing.B) {
	benchmarkHubBroadcast(b, 500)
}

func BenchmarkHubBroadcast_1000Clients(b *testing.B) {
	benchmarkHubBroadcast(b, 1000)
}

func BenchmarkHubChannelRouting_1000Clients(b *testing.B) {
	hub := NewHub(nil)
	go hub.Run()

	totalClients := 1000
	targetChannel := "chan_targeted"
	clients := make([]*Client, totalClients)

	for i := 0; i < totalClients; i++ {
		ch := "chan_other"
		if i%10 == 0 {
			ch = targetChannel
		}
		c := &Client{
			Hub:       hub,
			Send:      make(chan []byte, 1000),
			ServerID:  "srv_bench",
			ChannelID: ch,
			User: &auth.UserContext{
				ID:       fmt.Sprintf("usr_%d", i),
				Username: fmt.Sprintf("bench_%d", i),
			},
		}
		hub.register <- c
		clients[i] = c
	}
	time.Sleep(20 * time.Millisecond)

	msg := &WSMessage{
		Type:      TypeChat,
		ServerID:  "srv_bench",
		ChannelID: targetChannel,
		Content:   "Targeted Channel Broadcast Test",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		hub.dispatch(msg)
		for j := 0; j < totalClients; j += 10 {
			select {
			case <-clients[j].Send:
			default:
			}
		}
	}

	b.StopTimer()
	for _, c := range clients {
		hub.unregister <- c
	}
}

func BenchmarkHubRegistration_1000Clients(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		hub := NewHub(nil)
		go hub.Run()

		clients := make([]*Client, 1000)
		for j := 0; j < 1000; j++ {
			c := &Client{
				Hub:       hub,
				Send:      make(chan []byte, 10),
				ServerID:  "srv_bench",
				ChannelID: "chan_bench",
				User: &auth.UserContext{
					ID:       fmt.Sprintf("usr_%d", j),
					Username: fmt.Sprintf("bench_%d", j),
				},
			}
			hub.register <- c
			clients[j] = c
		}
		for _, c := range clients {
			hub.unregister <- c
		}
	}
}

func benchmarkHubBroadcast(b *testing.B, clientCount int) {
	hub := NewHub(nil)
	go hub.Run()

	clients := make([]*Client, clientCount)
	for i := 0; i < clientCount; i++ {
		c := &Client{
			Hub:       hub,
			Send:      make(chan []byte, 1000),
			ServerID:  "srv_bench",
			ChannelID: "chan_bench",
			User: &auth.UserContext{
				ID:       fmt.Sprintf("usr_%d", i),
				Username: fmt.Sprintf("bench_%d", i),
			},
		}
		hub.register <- c
		clients[i] = c
	}
	time.Sleep(20 * time.Millisecond)

	msg := &WSMessage{
		Type:      TypeChat,
		ServerID:  "srv_bench",
		ChannelID: "chan_bench",
		Content:   "High-Throughput Go Broadcast Test",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		hub.dispatch(msg)
		for _, c := range clients {
			select {
			case <-c.Send:
			default:
			}
		}
	}

	b.StopTimer()
	for _, c := range clients {
		hub.unregister <- c
	}
}
