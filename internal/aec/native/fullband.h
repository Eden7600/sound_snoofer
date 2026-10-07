#pragma once
#include <array>
#include <cmath>
#include <cstdint>
#include <cstring>
#include <stdexcept>
#include "api/audio/audio_processing.h"

namespace snoofer {
inline rtc::scoped_refptr<webrtc::AudioProcessing> HighBandAEC() {
    webrtc::AudioProcessing::Config config;
    config.pipeline.maximum_internal_processing_rate = 48000;
    config.echo_canceller.enabled = true;
    config.echo_canceller.enforce_high_pass_filtering = false;
    config.high_pass_filter.enabled = false;
    config.noise_suppression.enabled = false;
    config.gain_controller1.enabled = false;
    config.gain_controller2.enabled = false;
    return webrtc::AudioProcessingBuilder().SetConfig(config).Create();
}

// Complementary linear-phase crossover. Identical branches reconstruct exactly
// as a 64-sample delay; the upper branch is always echo-processed audio.
class Crossover {
    std::array<float,129> taps{}, low{}, high{};
    int position = 0;
public:
    Crossover() {
        constexpr double pi = 3.14159265358979323846;
        double sum = 0;
        for (int i = 0; i < 129; ++i) {
            int n = i - 64;
            double sinc = n ? std::sin(2*pi*6000/48000*n)/(pi*n) : 2.0*6000/48000;
            double window = .42 - .5*std::cos(2*pi*i/128) + .08*std::cos(4*pi*i/128);
            taps[i] = static_cast<float>(sinc * window);
            sum += taps[i];
        }
        for (auto& t : taps) t = static_cast<float>(t / sum);
    }
    float Process(float neural, float classic) {
        low[position] = neural; high[position] = classic;
        float value = high[(position + 129 - 64) % 129];
        for (int k = 0; k < 129; ++k) {
            int i = (position + 129 - k) % 129;
            value += taps[k] * (low[i] - high[i]);
        }
        position = (position + 1) % 129;
        return value;
    }
};

class Fullband {
    rtc::scoped_refptr<webrtc::AudioProcessing> apm = HighBandAEC();
    Crossover crossover;
    std::array<float,4096> history{};
    uint64_t written = 0;
public:
    // Neural: one 256-sample hop plus sinc transport = 834 samples (measured).
    // Bundled AEC3 delay is pinned by the native probe before mixing branches.
    static constexpr int kAECDelay = 430;
    static constexpr int kAlignment = 834 - kAECDelay;
    void Push(const float* mic, const float* ref) {
        float capture[480], render[480];
        std::memcpy(capture,mic,sizeof(capture));
        std::memcpy(render,ref,sizeof(render));
        float* capturePtr[] = {capture};
        float* renderPtr[] = {render};
        const webrtc::StreamConfig mono(48000,1);
        if (apm->ProcessReverseStream(renderPtr,mono,mono,renderPtr) ||
            apm->ProcessStream(capturePtr,mono,mono,capturePtr))
            throw std::runtime_error("high-band AEC failed");
        for (float sample : capture) history[written++ % history.size()] = sample;
    }
    void Mix(uint64_t start, float* neural, int count) {
        for (int i = 0; i < count; ++i) {
            int64_t source = static_cast<int64_t>(start + i) - kAlignment;
            float high = 0;
            if (source >= 0) {
                auto index = static_cast<uint64_t>(source);
                if (index >= written || written-index > history.size())
                    throw std::runtime_error("high-band framing mismatch");
                high = history[index % history.size()];
            }
            neural[i] = crossover.Process(neural[i],high);
        }
    }
};
}

