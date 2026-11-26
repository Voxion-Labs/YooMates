#ifndef YOOMATES_CNG_HPP
#define YOOMATES_CNG_HPP

#include <cstdint>
#include <vector>
#include <cmath>
#include <algorithm>

namespace yoomates {

// Fast XorShift32 pseudo-random number generator for zero-overhead comfort noise
class FastPRNG {
private:
    uint32_t state_;

public:
    explicit FastPRNG(uint32_t seed = 0x12345678) : state_(seed ? seed : 0x12345678) {}

    uint32_t next() {
        uint32_t x = state_;
        x ^= x << 13;
        x ^= x >> 17;
        x ^= x << 5;
        state_ = x;
        return x;
    }

    // Returns float between -1.0 and 1.0
    float next_float() {
        return (static_cast<float>(next()) / 2147483648.0f) - 1.0f;
    }
};

// Comfort Noise Generator (CNG) & Discontinuous Transmission (DTX) Controller
class ComfortNoiseGenerator {
private:
    FastPRNG prng_;
    float filter_state_ = 0.0f;
    double target_dbfs_ = -55.0; // Subtle low-level acoustic presence

public:
    explicit ComfortNoiseGenerator(double target_dbfs = -55.0, uint32_t seed = 0x9E3779B9)
        : prng_(seed), target_dbfs_(target_dbfs) {}

    void set_target_dbfs(double dbfs) {
        target_dbfs_ = std::min(-20.0, std::max(-90.0, dbfs));
    }

    double get_target_dbfs() const { return target_dbfs_; }

    // Generates spectrally shaped comfort noise into a 16-bit PCM buffer
    void generate(int16_t* buffer, size_t num_samples) {
        if (!buffer || num_samples == 0) return;

        // Amplitude scalar for target dBFS
        float peak_amplitude = 32767.0f * static_cast<float>(std::pow(10.0, target_dbfs_ / 20.0));
        
        // Single-pole low-pass filter (approximate pink/brown noise characteristic for natural acoustic presence)
        const float alpha = 0.75f;

        for (size_t i = 0; i < num_samples; ++i) {
            float white = prng_.next_float();
            filter_state_ = (alpha * filter_state_) + ((1.0f - alpha) * white);
            float sample = filter_state_ * peak_amplitude * 1.4f; // Compensate for filter attenuation

            // Clamp to int16_t range
            if (sample > 32767.0f) sample = 32767.0f;
            if (sample < -32768.0f) sample = -32768.0f;

            buffer[i] = static_cast<int16_t>(sample);
        }
    }

    // Silence Suppression / DTX Decision Manager
    class DTXController {
    private:
        int silence_frames_ = 0;
        int hangover_frames_ = 4;
        bool in_dtx_ = false;
        int dtx_counter_ = 0;
        int sid_interval_ = 20; // Send SID (Silence Insertion Descriptor) every 20 frames (~400ms)

    public:
        explicit DTXController(int hangover_frames = 4, int sid_interval = 20)
            : hangover_frames_(hangover_frames), sid_interval_(sid_interval) {}

        enum class Action {
            TRANSMIT_NORMAL,
            TRANSMIT_HANGOVER,
            TRANSMIT_SID, // Low-bandwidth comfort noise descriptor
            SUPPRESS       // Drop packet (bandwidth saved)
        };

        Action process_frame(bool is_speech) {
            if (is_speech) {
                silence_frames_ = 0;
                in_dtx_ = false;
                dtx_counter_ = 0;
                return Action::TRANSMIT_NORMAL;
            }

            silence_frames_++;
            if (silence_frames_ <= hangover_frames_) {
                return Action::TRANSMIT_HANGOVER;
            }

            in_dtx_ = true;
            dtx_counter_++;
            if (dtx_counter_ >= sid_interval_) {
                dtx_counter_ = 0;
                return Action::TRANSMIT_SID;
            }

            return Action::SUPPRESS;
        }

        bool is_in_dtx() const { return in_dtx_; }
        int get_silence_frame_count() const { return silence_frames_; }
    };
};

} // namespace yoomates

#endif // YOOMATES_CNG_HPP
