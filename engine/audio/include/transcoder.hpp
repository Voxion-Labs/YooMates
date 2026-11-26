#ifndef YOOMATES_TRANSCODER_HPP
#define YOOMATES_TRANSCODER_HPP

#include <cstdint>
#include <vector>
#include <cmath>
#include <algorithm>

namespace yoomates {

// Fast, low-latency audio resampler and dynamic soft-limiting processor
class AudioTranscoder {
public:
    // Linear interpolated sample rate conversion
    // Returns number of samples written to output buffer
    static size_t resample_pcm16(const int16_t* input, size_t in_samples, int in_rate,
                                 int16_t* output, size_t out_capacity, int out_rate) {
        if (!input || !output || in_samples == 0 || in_rate <= 0 || out_rate <= 0) {
            return 0;
        }

        if (in_rate == out_rate) {
            size_t count = std::min(in_samples, out_capacity);
            std::copy(input, input + count, output);
            return count;
        }

        double ratio = static_cast<double>(in_rate) / static_cast<double>(out_rate);
        size_t expected_out = static_cast<size_t>(in_samples / ratio);
        size_t out_count = std::min(expected_out, out_capacity);

        for (size_t out_idx = 0; out_idx < out_count; ++out_idx) {
            double src_pos = out_idx * ratio;
            size_t in_idx0 = static_cast<size_t>(src_pos);
            size_t in_idx1 = std::min(in_idx0 + 1, in_samples - 1);
            double frac = src_pos - in_idx0;

            double s0 = static_cast<double>(input[in_idx0]);
            double s1 = static_cast<double>(input[in_idx1]);
            double interpolated = s0 + frac * (s1 - s0);

            // Clamp to int16_t
            if (interpolated > 32767.0) interpolated = 32767.0;
            if (interpolated < -32768.0) interpolated = -32768.0;

            output[out_idx] = static_cast<int16_t>(interpolated);
        }

        return out_count;
    }

    // Apply digital gain (dB) and anti-clipping soft-limiter
    static void apply_gain_and_limit(int16_t* samples, size_t count, double gain_db) {
        if (!samples || count == 0) return;

        double gain_linear = std::pow(10.0, gain_db / 20.0);
        const double threshold = 0.70; // 70% of full scale (-3 dBFS)

        for (size_t i = 0; i < count; ++i) {
            double norm = (static_cast<double>(samples[i]) / 32767.0) * gain_linear;
            double abs_val = std::abs(norm);

            double out_norm = norm;
            if (abs_val > threshold) {
                // Soft compression curve above -3 dBFS to prevent harsh square-wave clipping
                double excess = abs_val - threshold;
                double compressed = threshold + (1.0 - threshold) * std::tanh(excess / (1.0 - threshold));
                out_norm = (norm < 0.0) ? -compressed : compressed;
            }

            double clamped = std::max(-1.0, std::min(1.0, out_norm)) * 32767.0;
            samples[i] = static_cast<int16_t>(clamped);
        }
    }
};

} // namespace yoomates

#endif // YOOMATES_TRANSCODER_HPP
