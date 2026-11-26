#ifndef YOOMATES_OPUS_PARSER_HPP
#define YOOMATES_OPUS_PARSER_HPP

#include <cstdint>
#include <cstddef>
#include <string>
#include <vector>

namespace yoomates {

enum class OpusMode {
    SILK,
    HYBRID,
    CELT
};

enum class OpusBandwidth {
    NARROWBAND = 8000,
    MEDIUMBAND = 12000,
    WIDEBAND = 16000,
    SUPERWIDEBAND = 24000,
    FULLBAND = 48000
};

struct OpusFrameInfo {
    OpusMode mode;
    OpusBandwidth bandwidth;
    int sample_rate_hz;
    int channels; // 1 for mono, 2 for stereo
    double frame_duration_ms;
    int frame_count;
    int samples_per_frame;
    int total_samples_per_channel;
    double total_duration_ms;
    bool is_vbr;
};

// RFC 6716 Opus Packet Inspector and Demuxer
class OpusParser {
public:
    static bool parse_packet(const uint8_t* data, size_t length, OpusFrameInfo& out_info) {
        if (!data || length < 1) {
            return false;
        }

        uint8_t toc = data[0];
        uint8_t config = (toc >> 3) & 0x1F;
        uint8_t s = (toc >> 2) & 0x01;
        uint8_t c = toc & 0x03;

        out_info.channels = (s == 1) ? 2 : 1;
        out_info.is_vbr = false;

        // Decode Config table (RFC 6716 Table 2)
        if (config <= 11) {
            // SILK-only
            out_info.mode = OpusMode::SILK;
            if (config <= 3) {
                out_info.bandwidth = OpusBandwidth::NARROWBAND;
                out_info.sample_rate_hz = 8000;
                static const double durations[] = {10.0, 20.0, 40.0, 60.0};
                out_info.frame_duration_ms = durations[config];
            } else if (config <= 7) {
                out_info.bandwidth = OpusBandwidth::MEDIUMBAND;
                out_info.sample_rate_hz = 12000;
                static const double durations[] = {10.0, 20.0, 40.0, 60.0};
                out_info.frame_duration_ms = durations[config - 4];
            } else {
                out_info.bandwidth = OpusBandwidth::WIDEBAND;
                out_info.sample_rate_hz = 16000;
                static const double durations[] = {10.0, 20.0, 40.0, 60.0};
                out_info.frame_duration_ms = durations[config - 8];
            }
        } else if (config <= 15) {
            // Hybrid
            out_info.mode = OpusMode::HYBRID;
            if (config <= 13) {
                out_info.bandwidth = OpusBandwidth::SUPERWIDEBAND;
                out_info.sample_rate_hz = 24000;
                out_info.frame_duration_ms = (config == 12) ? 10.0 : 20.0;
            } else {
                out_info.bandwidth = OpusBandwidth::FULLBAND;
                out_info.sample_rate_hz = 48000;
                out_info.frame_duration_ms = (config == 14) ? 10.0 : 20.0;
            }
        } else {
            // CELT-only
            out_info.mode = OpusMode::CELT;
            static const double celt_durations[] = {2.5, 5.0, 10.0, 20.0};
            int sub = (config - 16) % 4;
            int bw_group = (config - 16) / 4;
            out_info.frame_duration_ms = celt_durations[sub];

            switch (bw_group) {
                case 0:
                    out_info.bandwidth = OpusBandwidth::NARROWBAND;
                    out_info.sample_rate_hz = 8000;
                    break;
                case 1:
                    out_info.bandwidth = OpusBandwidth::WIDEBAND;
                    out_info.sample_rate_hz = 16000;
                    break;
                case 2:
                    out_info.bandwidth = OpusBandwidth::SUPERWIDEBAND;
                    out_info.sample_rate_hz = 24000;
                    break;
                default:
                    out_info.bandwidth = OpusBandwidth::FULLBAND;
                    out_info.sample_rate_hz = 48000;
                    break;
            }
        }

        // Decode Frame count code (RFC 6716 Section 3.2)
        switch (c) {
            case 0:
                out_info.frame_count = 1;
                break;
            case 1:
            case 2:
                out_info.frame_count = 2;
                break;
            case 3:
                if (length < 2) {
                    return false;
                }
                out_info.is_vbr = (data[1] & 0x80) != 0;
                out_info.frame_count = data[1] & 0x3F;
                if (out_info.frame_count < 1 || out_info.frame_count > 48) {
                    return false;
                }
                break;
        }

        // Standard 48kHz audio reference clock (WebRTC default)
        out_info.samples_per_frame = static_cast<int>(out_info.frame_duration_ms * 48.0);
        out_info.total_samples_per_channel = out_info.samples_per_frame * out_info.frame_count;
        out_info.total_duration_ms = out_info.frame_duration_ms * out_info.frame_count;

        return true;
    }

    static std::string mode_to_string(OpusMode mode) {
        switch (mode) {
            case OpusMode::SILK: return "SILK";
            case OpusMode::HYBRID: return "HYBRID";
            case OpusMode::CELT: return "CELT";
            default: return "UNKNOWN";
        }
    }
};

} // namespace yoomates

#endif // YOOMATES_OPUS_PARSER_HPP
