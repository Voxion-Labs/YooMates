#ifndef YOOMATES_VAD_HPP
#define YOOMATES_VAD_HPP

#include <cstdint>
#include <cstddef>
#include <cmath>
#include <vector>

namespace yoomates {

// Energy-based Voice Activity Detector (VAD) with hysteresis and hangover smoothing
class VoiceActivityDetector {
private:
    double threshold_dbfs_;
    double hysteresis_dbfs_;
    size_t hangover_frames_;
    size_t hangover_counter_;
    bool is_speaking_;
    double last_energy_dbfs_;

public:
    VoiceActivityDetector(double threshold_dbfs = -45.0, double hysteresis_dbfs = 3.0, size_t hangover_frames = 10)
        : threshold_dbfs_(threshold_dbfs),
          hysteresis_dbfs_(hysteresis_dbfs),
          hangover_frames_(hangover_frames),
          hangover_counter_(0),
          is_speaking_(false),
          last_energy_dbfs_(-100.0) {}

    // Process a frame of 16-bit linear PCM audio samples
    bool process_pcm16(const int16_t* samples, size_t sample_count) {
        if (!samples || sample_count == 0) {
            return is_speaking_;
        }

        double sum_squares = 0.0;
        for (size_t i = 0; i < sample_count; ++i) {
            double s = static_cast<double>(samples[i]);
            sum_squares += s * s;
        }

        double rms = std::sqrt(sum_squares / static_cast<double>(sample_count));
        
        // Avoid log(0)
        if (rms < 1.0) {
            rms = 1.0;
        }

        // Convert to dBFS (Full scale for 16-bit is 32768.0)
        double dbfs = 20.0 * std::log10(rms / 32768.0);
        last_energy_dbfs_ = dbfs;

        // Threshold check with hysteresis
        double effective_threshold = is_speaking_ ? (threshold_dbfs_ - hysteresis_dbfs_) : threshold_dbfs_;

        if (dbfs >= effective_threshold) {
            is_speaking_ = true;
            hangover_counter_ = hangover_frames_;
        } else {
            if (hangover_counter_ > 0) {
                hangover_counter_--;
                is_speaking_ = true;
            } else {
                is_speaking_ = false;
            }
        }

        return is_speaking_;
    }

    bool is_speaking() const { return is_speaking_; }
    double last_energy_dbfs() const { return last_energy_dbfs_; }

    void set_threshold(double dbfs) { threshold_dbfs_ = dbfs; }
};

} // namespace yoomates

#endif // YOOMATES_VAD_HPP
