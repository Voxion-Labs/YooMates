#ifndef YOOMATES_JITTER_BUFFER_HPP
#define YOOMATES_JITTER_BUFFER_HPP

#include <cstdint>
#include <deque>
#include <algorithm>
#include "audio_packet.hpp"

namespace yoomates {

// Adaptive Jitter Buffer with Sequence Re-ordering and Loss Detection
class AdaptiveJitterBuffer {
public:
    struct Stats {
        uint32_t received = 0;
        uint32_t played = 0;
        uint32_t dropped_late = 0;
        uint32_t plc_frames = 0;
        uint32_t target_delay_ms = 40;
        uint32_t current_depth_ms = 0;
    };

private:
    std::deque<AudioPacket> queue_;
    uint16_t next_seq_ = 0;
    bool initialized_ = false;

    uint32_t min_delay_ms_ = 20;
    uint32_t max_delay_ms_ = 120;
    uint32_t target_delay_ms_ = 40;
    uint32_t frame_duration_ms_ = 20;

    Stats stats_;

    // Handle 16-bit sequence number comparison with wrap-around
    static bool seq_less(uint16_t a, uint16_t b) {
        return static_cast<int16_t>(a - b) < 0;
    }

    static int16_t seq_diff(uint16_t a, uint16_t b) {
        return static_cast<int16_t>(a - b);
    }

public:
    AdaptiveJitterBuffer(uint32_t min_delay_ms = 20, uint32_t max_delay_ms = 120, uint32_t frame_duration_ms = 20)
        : min_delay_ms_(min_delay_ms), max_delay_ms_(max_delay_ms),
          target_delay_ms_(min_delay_ms + 20), frame_duration_ms_(frame_duration_ms) {}

    // Update target playout delay dynamically based on observed RFC 3550 jitter
    void update_target_delay(double jitter_ms) {
        uint32_t computed = static_cast<uint32_t>(jitter_ms * 2.5 + frame_duration_ms_);
        target_delay_ms_ = std::max(min_delay_ms_, std::min(max_delay_ms_, computed));
        stats_.target_delay_ms = target_delay_ms_;
    }

    // Insert packet into sorted position
    bool push(const AudioPacket& pkt) {
        stats_.received++;

        if (!initialized_) {
            next_seq_ = pkt.header.sequence_number;
            queue_.push_back(pkt);
            initialized_ = true;
            stats_.current_depth_ms = static_cast<uint32_t>(queue_.size()) * frame_duration_ms_;
            return true;
        }

        // Check if packet is late (already passed next_seq_)
        if (seq_less(pkt.header.sequence_number, next_seq_)) {
            stats_.dropped_late++;
            return false;
        }

        // Check for duplicate
        for (const auto& item : queue_) {
            if (item.header.sequence_number == pkt.header.sequence_number) {
                return false; // Duplicate dropped
            }
        }

        // Insertion sort by sequence number
        auto it = std::upper_bound(queue_.begin(), queue_.end(), pkt,
            [](const AudioPacket& a, const AudioPacket& b) {
                return seq_less(a.header.sequence_number, b.header.sequence_number);
            });
        queue_.insert(it, pkt);

        // Bounded queue enforcement (drop oldest if exceeding max delay depth)
        uint32_t max_packets = (max_delay_ms_ / frame_duration_ms_) + 2;
        while (queue_.size() > max_packets) {
            queue_.pop_front();
            stats_.dropped_late++;
            if (!queue_.empty()) {
                next_seq_ = queue_.front().header.sequence_number;
            }
        }

        stats_.current_depth_ms = static_cast<uint32_t>(queue_.size()) * frame_duration_ms_;
        return true;
    }

    // Retrieve next audio packet for playout; flags PLC if packet was lost
    bool pop(AudioPacket& out_pkt, bool& is_plc) {
        if (!initialized_ || queue_.empty()) {
            is_plc = false;
            stats_.current_depth_ms = 0;
            return false;
        }

        const auto& front = queue_.front();
        int16_t diff = seq_diff(front.header.sequence_number, next_seq_);

        if (diff == 0) {
            // Expected packet is available
            out_pkt = front;
            queue_.pop_front();
            next_seq_++;
            is_plc = false;
            stats_.played++;
            stats_.current_depth_ms = static_cast<uint32_t>(queue_.size()) * frame_duration_ms_;
            return true;
        } else if (diff > 0) {
            // Gap detected: next_seq_ packet is missing!
            // Trigger Packet Loss Concealment (PLC) and advance sequence
            out_pkt.header.sequence_number = next_seq_;
            out_pkt.header.timestamp = front.header.timestamp - (diff * frame_duration_ms_ * 48);
            out_pkt.payload.clear();
            next_seq_++;
            is_plc = true;
            stats_.plc_frames++;
            stats_.current_depth_ms = static_cast<uint32_t>(queue_.size()) * frame_duration_ms_;
            return true;
        } else {
            // Front packet has older sequence number than next_seq_ (stale/late)
            queue_.pop_front();
            stats_.dropped_late++;
            return pop(out_pkt, is_plc);
        }
    }

    size_t size() const { return queue_.size(); }
    bool is_empty() const { return queue_.empty(); }
    uint32_t get_target_delay_ms() const { return target_delay_ms_; }
    const Stats& get_stats() const { return stats_; }

    void reset() {
        queue_.clear();
        initialized_ = false;
        next_seq_ = 0;
        stats_ = Stats();
    }
};

} // namespace yoomates

#endif // YOOMATES_JITTER_BUFFER_HPP
