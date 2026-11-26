#include <iostream>
#include <string>
#include <vector>
#include <cassert>
#include <cmath>
#include "audio_packet.hpp"
#include "vad.hpp"
#include "voice_gateway.hpp"
#include "opus_parser.hpp"
#include "jitter_buffer.hpp"
#include "cng.hpp"
#include "transcoder.hpp"

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

    // 5. Test Opus TOC Parser & Demuxer (RFC 6716)
    {
        // Case A: Config 14 (Hybrid, Fullband 48kHz, 10ms, Mono, 1 frame)
        // TOC: (14 << 3) | (0 << 2) | 0 = 0x70
        uint8_t pkt_a[] = {0x70, 0xAA, 0xBB};
        OpusFrameInfo info_a;
        assert(OpusParser::parse_packet(pkt_a, sizeof(pkt_a), info_a));
        assert(info_a.mode == OpusMode::HYBRID);
        assert(info_a.sample_rate_hz == 48000);
        assert(info_a.channels == 1);
        assert(info_a.frame_duration_ms == 10.0);
        assert(info_a.frame_count == 1);
        assert(info_a.total_samples_per_channel == 480);

        // Case B: Config 31 (CELT, Fullband 48kHz, 20ms, Stereo, 2 equal frames)
        // TOC: (31 << 3) | (1 << 2) | 1 = 0xFD
        uint8_t pkt_b[] = {0xFD, 0x11, 0x22};
        OpusFrameInfo info_b;
        assert(OpusParser::parse_packet(pkt_b, sizeof(pkt_b), info_b));
        assert(info_b.mode == OpusMode::CELT);
        assert(info_b.channels == 2);
        assert(info_b.frame_duration_ms == 20.0);
        assert(info_b.frame_count == 2);
        assert(info_b.total_duration_ms == 40.0);
        assert(info_b.total_samples_per_channel == 1920);

        std::cout << "  [PASS] Opus Packet Inspection & RFC 6716 TOC Parsing\n";
    }

    // 6. Test Adaptive Dynamic Jitter Buffer & Packet Loss Concealment (PLC)
    {
        AdaptiveJitterBuffer jbuf(20, 100, 20);

        // Feed packets out-of-order: 100, 102, 101
        std::vector<uint8_t> data = {0x01};
        jbuf.push(AudioPacket(100, 0, 1, data));
        jbuf.push(AudioPacket(102, 960, 1, data));
        jbuf.push(AudioPacket(101, 480, 1, data));

        AudioPacket out_pkt;
        bool is_plc = false;

        // Pop 1: seq 100
        assert(jbuf.pop(out_pkt, is_plc));
        assert(!is_plc && out_pkt.header.sequence_number == 100);

        // Pop 2: seq 101 (reordered correctly!)
        assert(jbuf.pop(out_pkt, is_plc));
        assert(!is_plc && out_pkt.header.sequence_number == 101);

        // Pop 3: seq 102
        assert(jbuf.pop(out_pkt, is_plc));
        assert(!is_plc && out_pkt.header.sequence_number == 102);

        // Test PLC gap detection: Push 104 without pushing 103
        jbuf.push(AudioPacket(104, 1920, 1, data));

        // Next pop should flag PLC for missing seq 103!
        assert(jbuf.pop(out_pkt, is_plc));
        assert(is_plc && out_pkt.header.sequence_number == 103);
        assert(jbuf.get_stats().plc_frames == 1);

        // Subsequent pop delivers 104
        assert(jbuf.pop(out_pkt, is_plc));
        assert(!is_plc && out_pkt.header.sequence_number == 104);

        // Test late packet rejection (pushing seq 100 which was already played)
        assert(!jbuf.push(AudioPacket(100, 0, 1, data)));
        assert(jbuf.get_stats().dropped_late > 0);

        std::cout << "  [PASS] Adaptive Jitter Buffer Re-ordering & PLC Detection\n";
    }

    // 7. Test Silence Suppression & Comfort Noise Generation (CNG)
    {
        ComfortNoiseGenerator cng(-55.0);
        std::vector<int16_t> noise(480);
        cng.generate(noise.data(), noise.size());

        // Calculate RMS of generated comfort noise
        double sum_sq = 0.0;
        for (int16_t s : noise) {
            double v = static_cast<double>(s) / 32767.0;
            sum_sq += v * v;
        }
        double rms = std::sqrt(sum_sq / static_cast<double>(noise.size()));
        double rms_dbfs = 20.0 * std::log10(std::max(rms, 1e-6));

        // Comfort noise at -55 dBFS should be roughly between -65 dBFS and -45 dBFS
        assert(rms_dbfs > -68.0 && rms_dbfs < -42.0);

        // Test DTX Controller
        ComfortNoiseGenerator::DTXController dtx(3, 10);
        assert(dtx.process_frame(true) == ComfortNoiseGenerator::DTXController::Action::TRANSMIT_NORMAL);
        assert(dtx.process_frame(false) == ComfortNoiseGenerator::DTXController::Action::TRANSMIT_HANGOVER);
        assert(dtx.process_frame(false) == ComfortNoiseGenerator::DTXController::Action::TRANSMIT_HANGOVER);
        assert(dtx.process_frame(false) == ComfortNoiseGenerator::DTXController::Action::TRANSMIT_HANGOVER);
        assert(dtx.process_frame(false) == ComfortNoiseGenerator::DTXController::Action::SUPPRESS);
        assert(dtx.is_in_dtx());

        std::cout << "  [PASS] Silence Suppression (DTX) & Comfort Noise Generation (CNG)\n";
    }

    // 8. Test Audio Resampling & Dynamic Limiting
    {
        // Generate 480 samples of 440Hz tone at 48kHz (10 ms)
        std::vector<int16_t> in_48k(480);
        for (size_t i = 0; i < in_48k.size(); ++i) {
            in_48k[i] = static_cast<int16_t>(16000.0 * std::sin(2.0 * 3.14159 * 440.0 * i / 48000.0));
        }

        // Downsample 48kHz -> 16kHz (expected 160 samples for 10ms)
        std::vector<int16_t> out_16k(200);
        size_t n_down = AudioTranscoder::resample_pcm16(in_48k.data(), in_48k.size(), 48000,
                                                        out_16k.data(), out_16k.size(), 16000);
        assert(n_down == 160);

        // Upsample 16kHz -> 48kHz (expected 480 samples for 10ms)
        std::vector<int16_t> out_48k(600);
        size_t n_up = AudioTranscoder::resample_pcm16(out_16k.data(), n_down, 16000,
                                                      out_48k.data(), out_48k.size(), 48000);
        assert(n_up == 480);

        // Test Soft-limiter protection: amplify signal with +12dB boost
        std::vector<int16_t> loud_signal(100, 25000);
        AudioTranscoder::apply_gain_and_limit(loud_signal.data(), loud_signal.size(), 12.0);
        for (int16_t s : loud_signal) {
            assert(s > 0 && s <= 32767); // Strictly bounded without overflow
        }

        std::cout << "  [PASS] Low-Latency Resampling (48kHz <-> 16kHz) & Soft-Limiter\n";
    }

    std::cout << "[yoomates-audio-engine] All C++ audio tests PASSED cleanly (8/8).\n";
}

int main(int argc, char* argv[]) {
    if (argc > 1 && std::string(argv[1]) == "--test") {
        run_tests();
        return 0;
    }

    std::cout << "==========================================================\n";
    std::cout << "  YooMates C++ High-Performance Audio Engine (v1.1)\n";
    std::cout << "  Low-Latency WebRTC Gateway, Jitter Buffer & Transcoder\n";
    std::cout << "==========================================================\n";
    std::cout << "Usage: yoomates-voice [--test]\n";
    return 0;
}
