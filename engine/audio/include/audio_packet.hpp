#ifndef YOOMATES_AUDIO_PACKET_HPP
#define YOOMATES_AUDIO_PACKET_HPP

#include <cstdint>
#include <vector>
#include <cstring>
#include <stdexcept>
#include <cmath>

namespace yoomates {

// RTP-compatible audio packet header (RFC 3550)
struct AudioHeader {
    uint8_t version : 2;
    uint8_t padding : 1;
    uint8_t extension : 1;
    uint8_t csrc_count : 4;
    uint8_t marker : 1;
    uint8_t payload_type : 7;
    uint16_t sequence_number;
    uint32_t timestamp;
    uint32_t ssrc;

    AudioHeader()
        : version(2), padding(0), extension(0), csrc_count(0),
          marker(0), payload_type(111), sequence_number(0),
          timestamp(0), ssrc(0) {}
};

// Represents an audio packet flowing through the low-latency media gateway
class AudioPacket {
public:
    AudioHeader header;
    std::vector<uint8_t> payload;

    AudioPacket() = default;

    AudioPacket(uint16_t seq, uint32_t ts, uint32_t ssrc, const std::vector<uint8_t>& data) {
        header.sequence_number = seq;
        header.timestamp = ts;
        header.ssrc = ssrc;
        payload = data;
    }

    // Serialize packet to wire format
    std::vector<uint8_t> serialize() const {
        std::vector<uint8_t> buffer(12 + payload.size());
        
        buffer[0] = (header.version << 6) | (header.padding << 5) | (header.extension << 4) | (header.csrc_count & 0x0F);
        buffer[1] = (header.marker << 7) | (header.payload_type & 0x7F);
        
        buffer[2] = (header.sequence_number >> 8) & 0xFF;
        buffer[3] = header.sequence_number & 0xFF;

        buffer[4] = (header.timestamp >> 24) & 0xFF;
        buffer[5] = (header.timestamp >> 16) & 0xFF;
        buffer[6] = (header.timestamp >> 8) & 0xFF;
        buffer[7] = header.timestamp & 0xFF;

        buffer[8] = (header.ssrc >> 24) & 0xFF;
        buffer[9] = (header.ssrc >> 16) & 0xFF;
        buffer[10] = (header.ssrc >> 8) & 0xFF;
        buffer[11] = header.ssrc & 0xFF;

        if (!payload.empty()) {
            std::memcpy(buffer.data() + 12, payload.data(), payload.size());
        }

        return buffer;
    }

    // Deserialize packet from raw wire bytes
    static AudioPacket deserialize(const uint8_t* data, size_t length) {
        if (length < 12) {
            throw std::runtime_error("Invalid packet: buffer smaller than RTP header");
        }

        AudioPacket pkt;
        pkt.header.version = (data[0] >> 6) & 0x03;
        pkt.header.padding = (data[0] >> 5) & 0x01;
        pkt.header.extension = (data[0] >> 4) & 0x01;
        pkt.header.csrc_count = data[0] & 0x0F;

        pkt.header.marker = (data[1] >> 7) & 0x01;
        pkt.header.payload_type = data[1] & 0x7F;

        pkt.header.sequence_number = (static_cast<uint16_t>(data[2]) << 8) | data[3];
        pkt.header.timestamp = (static_cast<uint32_t>(data[4]) << 24) |
                               (static_cast<uint32_t>(data[5]) << 16) |
                               (static_cast<uint32_t>(data[6]) << 8) |
                               data[7];
        pkt.header.ssrc = (static_cast<uint32_t>(data[8]) << 24) |
                          (static_cast<uint32_t>(data[9]) << 16) |
                          (static_cast<uint32_t>(data[10]) << 8) |
                          data[11];

        size_t payload_size = length - 12;
        if (payload_size > 0) {
            pkt.payload.resize(payload_size);
            std::memcpy(pkt.payload.data(), data + 12, payload_size);
        }

        return pkt;
    }
};

// Lightweight jitter & packet loss estimator (RFC 3550)
class JitterEstimator {
private:
    double jitter_ = 0.0;
    uint32_t last_transit_ = 0;
    uint32_t expected_packets_ = 0;
    uint32_t received_packets_ = 0;
    uint16_t last_seq_ = 0;
    bool initialized_ = false;

public:
    void update(uint16_t seq, uint32_t arrival_time_ms, uint32_t rtp_timestamp_ms) {
        received_packets_++;

        if (!initialized_) {
            last_seq_ = seq;
            last_transit_ = arrival_time_ms - rtp_timestamp_ms;
            expected_packets_ = 1;
            initialized_ = true;
            return;
        }

        // Sequence number gap calculation
        int16_t diff = seq - last_seq_;
        if (diff > 0) {
            expected_packets_ += diff;
            last_seq_ = seq;
        }

        // RFC 3550 Interarrival Jitter Calculation
        uint32_t transit = arrival_time_ms - rtp_timestamp_ms;
        double d = std::abs(static_cast<double>(transit) - static_cast<double>(last_transit_));
        last_transit_ = transit;
        jitter_ += (d - jitter_) / 16.0;
    }

    double get_jitter_ms() const { return jitter_; }

    double get_packet_loss_rate() const {
        if (expected_packets_ == 0) return 0.0;
        int32_t lost = static_cast<int32_t>(expected_packets_) - static_cast<int32_t>(received_packets_);
        if (lost <= 0) return 0.0;
        return static_cast<double>(lost) / static_cast<double>(expected_packets_);
    }

    uint32_t get_received_count() const { return received_packets_; }
};

} // namespace yoomates

#endif // YOOMATES_AUDIO_PACKET_HPP
