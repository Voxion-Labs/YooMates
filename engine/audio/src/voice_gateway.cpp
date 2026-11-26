#include "voice_gateway.hpp"

namespace yoomates {

bool VoiceRoom::add_peer(const std::string& peer_id, uint32_t ssrc) {
    FastLockGuard lock(mutex_);
    if (peers_.find(peer_id) != peers_.end()) {
        return false;
    }
    peers_[peer_id] = VoicePeerState(peer_id, ssrc);
    return true;
}

bool VoiceRoom::remove_peer(const std::string& peer_id) {
    FastLockGuard lock(mutex_);
    return peers_.erase(peer_id) > 0;
}

bool VoiceRoom::set_peer_mute(const std::string& peer_id, bool muted) {
    FastLockGuard lock(mutex_);
    auto it = peers_.find(peer_id);
    if (it == peers_.end()) return false;
    it->second.is_muted = muted;
    return true;
}

bool VoiceRoom::set_peer_deafen(const std::string& peer_id, bool deafened) {
    FastLockGuard lock(mutex_);
    auto it = peers_.find(peer_id);
    if (it == peers_.end()) return false;
    it->second.is_deafened = deafened;
    return true;
}

bool VoiceRoom::set_peer_speaking(const std::string& peer_id, bool speaking) {
    FastLockGuard lock(mutex_);
    auto it = peers_.find(peer_id);
    if (it == peers_.end()) return false;
    it->second.is_speaking = speaking;
    return true;
}

std::vector<std::string> VoiceRoom::route_packet(const std::string& sender_id, const AudioPacket& packet, uint32_t arrival_time_ms) {
    FastLockGuard lock(mutex_);
    std::vector<std::string> destination_peer_ids;

    auto sender_it = peers_.find(sender_id);
    if (sender_it == peers_.end()) {
        return destination_peer_ids;
    }

    // If sender is muted, drop packet immediately
    if (sender_it->second.is_muted) {
        return destination_peer_ids;
    }

    sender_it->second.packets_received++;
    sender_it->second.jitter.update(packet.header.sequence_number, arrival_time_ms, packet.header.timestamp);

    for (auto& pair : peers_) {
        const std::string& target_id = pair.first;
        auto& target_state = pair.second;

        // Don't loop back to sender, and don't route to deafened users
        if (target_id != sender_id && !target_state.is_deafened) {
            destination_peer_ids.push_back(target_id);
            sender_it->second.packets_routed++;
        }
    }

    return destination_peer_ids;
}

size_t VoiceRoom::peer_count() const {
    FastLockGuard lock(mutex_);
    return peers_.size();
}

std::vector<std::string> VoiceRoom::get_peer_ids() const {
    FastLockGuard lock(mutex_);
    std::vector<std::string> ids;
    ids.reserve(peers_.size());
    for (const auto& pair : peers_) {
        ids.push_back(pair.first);
    }
    return ids;
}

bool VoiceRoom::get_peer_state(const std::string& peer_id, VoicePeerState& state_out) const {
    FastLockGuard lock(mutex_);
    auto it = peers_.find(peer_id);
    if (it == peers_.end()) return false;
    state_out = it->second;
    return true;
}

std::shared_ptr<VoiceRoom> VoiceGateway::get_or_create_room(const std::string& room_id) {
    FastLockGuard lock(mutex_);
    auto it = rooms_.find(room_id);
    if (it != rooms_.end()) {
        return it->second;
    }
    auto room = std::make_shared<VoiceRoom>(room_id);
    rooms_[room_id] = room;
    return room;
}

bool VoiceGateway::remove_room(const std::string& room_id) {
    FastLockGuard lock(mutex_);
    return rooms_.erase(room_id) > 0;
}

size_t VoiceGateway::active_room_count() const {
    FastLockGuard lock(mutex_);
    return rooms_.size();
}

std::vector<std::string> VoiceGateway::get_active_room_ids() const {
    FastLockGuard lock(mutex_);
    std::vector<std::string> ids;
    ids.reserve(rooms_.size());
    for (const auto& pair : rooms_) {
        ids.push_back(pair.first);
    }
    return ids;
}

} // namespace yoomates
