#ifndef YOOMATES_VOICE_GATEWAY_HPP
#define YOOMATES_VOICE_GATEWAY_HPP

#include <string>
#include <unordered_map>
#include <vector>
#include <memory>
#include "lock.hpp"
#include "audio_packet.hpp"
#include "vad.hpp"

namespace yoomates {

struct VoicePeerState {
    std::string peer_id;
    uint32_t ssrc;
    bool is_muted;
    bool is_deafened;
    bool is_speaking;
    uint64_t packets_received;
    uint64_t packets_routed;
    JitterEstimator jitter;
    VoiceActivityDetector vad;

    VoicePeerState()
        : ssrc(0), is_muted(false), is_deafened(false), is_speaking(false),
          packets_received(0), packets_routed(0) {}

    VoicePeerState(const std::string& id, uint32_t s)
        : peer_id(id), ssrc(s), is_muted(false), is_deafened(false),
          is_speaking(false), packets_received(0), packets_routed(0) {}
};

class VoiceRoom {
private:
    std::string room_id_;
    std::unordered_map<std::string, VoicePeerState> peers_;
    mutable FastMutex mutex_;

public:
    explicit VoiceRoom(const std::string& room_id) : room_id_(room_id) {}

    bool add_peer(const std::string& peer_id, uint32_t ssrc);
    bool remove_peer(const std::string& peer_id);
    bool set_peer_mute(const std::string& peer_id, bool muted);
    bool set_peer_deafen(const std::string& peer_id, bool deafened);
    bool set_peer_speaking(const std::string& peer_id, bool speaking);

    // Routes an incoming packet from a sender to all listening peers in the room
    std::vector<std::string> route_packet(const std::string& sender_id, const AudioPacket& packet, uint32_t arrival_time_ms);

    size_t peer_count() const;
    std::vector<std::string> get_peer_ids() const;
    bool get_peer_state(const std::string& peer_id, VoicePeerState& state_out) const;
    std::string get_room_id() const { return room_id_; }
};

class VoiceGateway {
private:
    std::unordered_map<std::string, std::shared_ptr<VoiceRoom>> rooms_;
    mutable FastMutex mutex_;

public:
    VoiceGateway() = default;

    std::shared_ptr<VoiceRoom> get_or_create_room(const std::string& room_id);
    bool remove_room(const std::string& room_id);
    size_t active_room_count() const;
    std::vector<std::string> get_active_room_ids() const;
};

} // namespace yoomates

#endif // YOOMATES_VOICE_GATEWAY_HPP
