#include <iostream>
#include <string>
#include <vector>
#include <cassert>
#include <cmath>
#include "audio_packet.hpp"
#include "vad.hpp"
#include "voice_gateway.hpp"

using namespace yoomates;

void run_tests() {
    std::cout << "[yoomates-audio-engine] Running C++ audio verification suite...\n";

    // 1. Test RTP Packet Serialization & Deserialization
    {
        std::vector<uint8_t> dummy_opus = {0xF8, 0xFF, 0xFE, 0x01, 0x02, 0x03};
        AudioPacket pkt(101, 48000, 0x12345678, dummy_opus);
        auto serialized = pkt.serialize();

        assert(serialized.size() == 12 + dummy_opus.size());
        auto deserialized = AudioPacket::deserialize(serialized.data(), serialized.size());

        assert(deserialized.header.sequence_number == 101);
        assert(deserialized.header.timestamp == 48000);
        assert(deserialized.header.ssrc == 0x12345678);
        assert(deserialized.payload == dummy_opus);
        std::cout << "  [PASS] RTP Packet Serialization & RFC 3550 Deserialization\n";
    }

    // 2. Test Jitter & Packet Loss Estimator
    {
        JitterEstimator jitter;
        jitter.update(1, 20, 0);
        jitter.update(2, 40, 20);
        jitter.update(4, 85, 60); // Simulated packet 3 lost

        assert(jitter.get_received_count() == 3);
        assert(jitter.get_packet_loss_rate() > 0.0);
        assert(jitter.get_jitter_ms() >= 0.0);
        std::cout << "  [PASS] Interarrival Jitter & Packet Loss Estimation\n";
    }

    // 3. Test Voice Activity Detection (VAD)
    {
        VoiceActivityDetector vad(-45.0, 3.0, 5);

        // Frame of silence (amplitude = 0)
        std::vector<int16_t> silence(480, 0);
        bool speaking_silence = vad.process_pcm16(silence.data(), silence.size());
        assert(!speaking_silence);

        // Frame of active speech (amplitude = 8000, ~ -12 dBFS)
        std::vector<int16_t> speech(480);
        for (size_t i = 0; i < speech.size(); ++i) {
            speech[i] = static_cast<int16_t>(8000.0 * std::sin(2.0 * 3.14159 * 440.0 * i / 48000.0));
        }
        bool speaking_voice = vad.process_pcm16(speech.data(), speech.size());
        assert(speaking_voice);
        assert(vad.last_energy_dbfs() > -20.0);
        std::cout << "  [PASS] Real-time VAD & RMS Energy Metering\n";
    }

    // 4. Test Voice Room Audio Routing & Peer Isolation
    {
        VoiceGateway gateway;
        auto room = gateway.get_or_create_room("chan_voice_lounge");
        assert(room != nullptr);

        assert(room->add_peer("user_alice", 1001));
        assert(room->add_peer("user_bob", 1002));
        assert(room->add_peer("user_charlie", 1003));
        assert(room->peer_count() == 3);

        // Alice sends a packet -> should route to Bob and Charlie
        std::vector<uint8_t> payload = {0x01, 0x02};
        AudioPacket pkt(1, 1000, 1001, payload);
        auto targets = room->route_packet("user_alice", pkt, 20);
        assert(targets.size() == 2);

        // Charlie deafens -> should not receive packets
        room->set_peer_deafen("user_charlie", true);
        auto targets_after_deafen = room->route_packet("user_alice", pkt, 40);
        assert(targets_after_deafen.size() == 1);
        assert(targets_after_deafen[0] == "user_bob");

        // Alice mutes -> her packets should be dropped
        room->set_peer_mute("user_alice", true);
        auto targets_muted = room->route_packet("user_alice", pkt, 60);
        assert(targets_muted.empty());

        // Remove user
        room->remove_peer("user_bob");
        assert(room->peer_count() == 2);
        std::cout << "  [PASS] Multi-peer Audio Packet Multiplexing & Mute/Deafen Routing\n";
    }

    std::cout << "[yoomates-audio-engine] All C++ audio tests PASSED cleanly (4/4).\n";
}

int main(int argc, char* argv[]) {
    if (argc > 1 && std::string(argv[1]) == "--test") {
        run_tests();
        return 0;
    }

    std::cout << "==========================================================\n";
    std::cout << "  YooMates C++ High-Performance Audio Engine (v1.0)\n";
    std::cout << "  Low-Latency WebRTC Gateway & RTP Packet Processor\n";
    std::cout << "==========================================================\n";
    std::cout << "Usage: yoomates-voice [--test]\n";
    return 0;
}
