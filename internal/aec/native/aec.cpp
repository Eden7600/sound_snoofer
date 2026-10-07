// Echo cancellation for Snoofer: WebRTC AEC3 behind a small C ABI. The
// control thread creates, configures, reads and destroys an engine; the
// insert stages run on Voicemeeter's audio thread from the monitor's callback.
// Configuration crosses threads through atomics only, and every exception is
// caught at the boundary, after which the engine passes audio through.
#include <windows.h>

#include <atomic>
#include <cmath>
#include <cstring>
#include <memory>
#include <optional>
#include <stdexcept>
#include <vector>

#include "api/audio/audio_processing.h"
#include "api/audio/echo_canceller3_config.h"
#include "api/audio/echo_control.h"
#include "modules/audio_processing/aec3/echo_canceller3.h"

extern "C" {
#include "audio_buffer.h"
}

namespace {

constexpr int kNone = -1;
constexpr int kMaxFrame = 480;  // 10 ms at 48 kHz, the largest supported rate.

// Strength presets relax AEC3's suppressor: higher masking thresholds keep
// more near-end signal. Strong is AEC3's default.
webrtc::EchoCanceller3Config ConfigFor(int strength) {
    auto c = webrtc::EchoCanceller3::CreateDefaultMultichannelConfig();
    // The bundled detector compares only L/R; center/surround-only audio must
    // retain its own filters even when the front pair is identical or silent.
    c.multi_channel.detect_stereo_content = false;
    // One of six independent speakers cannot explain 80% of capture energy.
    // Keep lag aggregation, but allow a 10% contribution to establish delay.
    c.delay.delay_candidate_detection_threshold = 0.9f;
    c.delay.render_alignment_mixing.prefer_first_two_channels = false;
    using M = webrtc::EchoCanceller3Config::Suppressor::MaskingThresholds;
    if (strength == 1) {
        c.suppressor.normal_tuning.mask_lf = M(.5f, .6f, .3f);
        c.suppressor.normal_tuning.mask_hf = M(.15f, .2f, .3f);
    } else if (strength == 2) {
        c.suppressor.normal_tuning.mask_lf = M(.8f, .9f, .3f);
        c.suppressor.normal_tuning.mask_hf = M(.3f, .4f, .3f);
    }
    return c;
}

class Factory : public webrtc::EchoControlFactory {
   public:
    explicit Factory(int strength) : strength_(strength) {}
    std::unique_ptr<webrtc::EchoControl> Create(int sample_rate_hz, int num_render_channels, int num_capture_channels) override {
        return std::make_unique<webrtc::EchoCanceller3>(ConfigFor(strength_), std::nullopt, sample_rate_hz,
                                                        static_cast<size_t>(num_render_channels),
                                                        static_cast<size_t>(num_capture_channels));
    }

   private:
    int strength_;
};

// Fifo is a fixed-capacity sample queue; it never allocates after creation.
class Fifo {
   public:
    void Reset(size_t capacity) {
        data_.assign(capacity, 0.0f);
        head_ = size_ = 0;
    }
    size_t Size() const { return size_; }
    void Push(float v) {
        if (size_ == data_.size()) throw std::runtime_error("capture output overflow");
        data_[(head_ + size_) % data_.size()] = v;
        ++size_;
    }
    float Pop() {
        float v = data_[head_];
        head_ = (head_ + 1) % data_.size();
        --size_;
        return v;
    }

   private:
    std::vector<float> data_;
    size_t head_ = 0, size_ = 0;
};

}  // namespace

// AECConfig is the control thread's request; -1 marks an unused channel.
struct AECConfig {
    int mic[2];        // Input-insert channels of the managed mic strip.
    int reference[8];  // Output-insert channels of the speaker bus.
    int strength;      // 0 strong, 1 balanced, 2 gentle.
    int bypass;        // Non-zero: pass the mic through untouched.
};

// AECStats is read by the control thread at any time.
struct AECStats {
    int active;         // Processing with a supported rate and targets.
    int sample_rate;    // As last seen from Voicemeeter.
    int erle_centi_db;  // Echo return loss enhancement x100; -1 if unknown.
    int delay_ms;       // Estimated echo delay; -1 if unknown.
    unsigned frames;    // 10 ms frames processed.
    int failed;         // An engine error switched to pass-through.
};

struct AEC {
    // Requested configuration, written by the control thread.
    std::atomic<int> mic0{kNone}, mic1{kNone}, strength{0}, bypass{1};
    std::atomic<int> refs[8] = {-1, -1, -1, -1, -1, -1, -1, -1};
    std::atomic<unsigned> generation{0};
    // Reported state, written by the audio thread.
    std::atomic<int> active{0}, rate{0}, erle{-1}, delay{-1}, failed{0};
    std::atomic<unsigned> frames{0};

    // Audio-thread state.
    unsigned applied = ~0u;
    int sample_rate = 0;
    int micChannels = 0;
    int mic[2] = {kNone, kNone}, ref[8] = {};
    int captureFill = 0, renderFill = 0;
    long expectedReferenceSamples = 0;
    bool processing = false;
    rtc::scoped_refptr<webrtc::AudioProcessing> apm;
    Fifo micOut[2];
    float capture[2][kMaxFrame], render[8][kMaxFrame];
};

namespace {

bool Supported(int rate) { return rate == 48000 || rate == 32000 || rate == 16000; }

// Configure (re)creates the engine for the current request and rate; it runs
// on the audio thread only when the request or the rate changes.
void Configure(AEC* a, int rate) {
    a->applied = a->generation.load(std::memory_order_acquire);
    a->mic[0] = a->mic0.load();
    a->mic[1] = a->mic1.load();
    for (int c = 0; c < 8; ++c) a->ref[c] = a->refs[c].load();
    a->captureFill = a->renderFill = 0;
    a->expectedReferenceSamples = 0;
    a->sample_rate = rate;
    a->micChannels = (a->mic[0] != kNone) + (a->mic[1] != kNone && a->mic[1] != a->mic[0]);
    a->processing = !a->bypass.load() && Supported(rate) && a->micChannels > 0 && a->ref[0] != kNone;
    a->apm = nullptr;
    a->active.store(a->processing ? 1 : 0);
    a->rate.store(rate);
    a->erle.store(-1);
    a->delay.store(-1);
    if (!a->processing) return;
    webrtc::AudioProcessing::Config config;
    config.pipeline.maximum_internal_processing_rate = 48000;
    config.pipeline.multi_channel_render = true;
    config.pipeline.multi_channel_capture = true;
    config.high_pass_filter.enabled = true;
    config.echo_canceller.enabled = true;
    config.noise_suppression.enabled = false;
    config.gain_controller1.enabled = false;
    config.gain_controller2.enabled = false;
    a->apm = webrtc::AudioProcessingBuilder()
                 .SetConfig(config)
                 .SetEchoControlFactory(std::make_unique<Factory>(a->strength.load()))
                 .Create();
    const size_t frame = static_cast<size_t>(rate / 100);
    for (auto& f : a->micOut) {
        f.Reset(frame + 1);
        for (size_t i = 0; i < frame; ++i) f.Push(0.0f);  // One frame of latency.
    }
}

void PassThrough(AudioBuffer* b) {
    for (long c = 0; c < b->outputs && c < 128; ++c) {
        if (b->read[c] && b->write[c] && b->read[c] != b->write[c]) {
            std::memmove(b->write[c], b->read[c], static_cast<size_t>(b->samples) * sizeof(float));
        }
    }
}

bool Valid(AudioBuffer* b) {
    return b && b->samples > 0 && b->samples <= 65536 && b->inputs > 0 && b->inputs <= 128 && b->outputs == b->inputs;
}

bool Has(AudioBuffer* b, int channel) {
    return channel >= 0 && channel < b->inputs && b->read[channel] && b->write[channel];
}

void Process(AEC* a, AudioBuffer* b) {
    if (a->applied != a->generation.load(std::memory_order_acquire) || a->sample_rate != b->sr) {
        Configure(a, b->sr);
    }
    if (!a->processing) return;
    const int frame = a->sample_rate / 100;
    if (a->expectedReferenceSamples != 0) throw std::runtime_error("missing output callback");
    for (int ch = 0; ch < a->micChannels; ++ch) {
        if (!Has(b, a->mic[ch])) throw std::runtime_error("missing microphone channel");
    }
    a->expectedReferenceSamples = b->samples;
    const webrtc::StreamConfig captureConfig(a->sample_rate, static_cast<size_t>(a->micChannels));
    float* capturePtr[2] = {a->capture[0], a->capture[1]};
    // Complete and consume frames incrementally: even large callbacks cannot
    // overflow a four-frame FIFO or introduce buffer-size-dependent latency.
    for (long i = 0; i < b->samples; ++i) {
        for (int ch = 0; ch < a->micChannels; ++ch) {
            a->capture[ch][a->captureFill] = b->read[a->mic[ch]][i];
        }
        if (++a->captureFill == frame) {
            if (a->apm->ProcessStream(capturePtr, captureConfig, captureConfig, capturePtr) != 0) {
                throw std::runtime_error("ProcessStream failed");
            }
            for (int ch = 0; ch < a->micChannels; ++ch) {
                for (int j = 0; j < frame; ++j) a->micOut[ch].Push(a->capture[ch][j]);
            }
            a->captureFill = 0;
            unsigned n = a->frames.fetch_add(1) + 1;
            if (n % 50 == 0) {
                webrtc::AudioProcessingStats s = a->apm->GetStatistics();
                a->erle.store(s.echo_return_loss_enhancement ? static_cast<int>(std::lround(*s.echo_return_loss_enhancement * 100)) : -1);
                a->delay.store(s.delay_ms ? static_cast<int>(*s.delay_ms) : -1);
            }
        }
        for (int ch = 0; ch < a->micChannels; ++ch) {
            if (!a->micOut[ch].Size()) throw std::runtime_error("capture output underflow");
            b->write[a->mic[ch]][i] = a->micOut[ch].Pop();
        }
    }
}

}  // namespace

extern "C" {

__declspec(dllexport) HRESULT __cdecl AECCreate(AEC** out) {
    if (!out) return E_INVALIDARG;
    try {
        *out = new AEC();
        return S_OK;
    } catch (...) {
        *out = nullptr;
        return E_OUTOFMEMORY;
    }
}

__declspec(dllexport) HRESULT __cdecl AECDestroy(AEC* a) {
    delete a;  // Only after the callback no longer calls the insert stages.
    return S_OK;
}

__declspec(dllexport) HRESULT __cdecl AECConfigureV2(AEC* a, const AECConfig* c) {
    if (!a || !c || c->mic[0] < 0 || c->mic[0] >= 128 || c->mic[1] < -1 || c->mic[1] >= 128 ||
        c->strength < 0 || c->strength > 2 || (c->bypass != 0 && c->bypass != 1)) return E_INVALIDARG;
    for (int i = 0; i < 8; ++i) {
        if (c->reference[i] < 0 || c->reference[i] >= 128) return E_INVALIDARG;
        for (int j = 0; j < i; ++j) if (c->reference[i] == c->reference[j]) return E_INVALIDARG;
    }
    a->mic0.store(c->mic[0]);
    a->mic1.store(c->mic[1]);
    for (int i = 0; i < 8; ++i) a->refs[i].store(c->reference[i]);
    a->strength.store(c->strength);
    a->bypass.store(c->bypass);
    a->generation.fetch_add(1, std::memory_order_release);
    return S_OK;
}

__declspec(dllexport) HRESULT __cdecl AECReadStats(AEC* a, AECStats* s) {
    if (!a || !s) return E_INVALIDARG;
    s->active = a->active.load();
    s->sample_rate = a->rate.load();
    s->erle_centi_db = a->erle.load();
    s->delay_ms = a->delay.load();
    s->frames = a->frames.load();
    s->failed = a->failed.load();
    return S_OK;
}

// AECInputInsert processes the mic channels and passes every other channel through.
__declspec(dllexport) void __stdcall AECInputInsert(void* context, AudioBuffer* b) {
    AEC* a = static_cast<AEC*>(context);
    if (!Valid(b)) return;
    PassThrough(b);
    if (!a || a->failed.load()) return;
    try {
        Process(a, b);
    } catch (...) {
        a->failed.store(1);
        a->active.store(0);
        PassThrough(b);
    }
}

// AECOutputInsert records the speaker reference; it never changes the output.
__declspec(dllexport) void __stdcall AECOutputInsert(void* context, AudioBuffer* b) {
    AEC* a = static_cast<AEC*>(context);
    if (!a || !Valid(b) || !a->processing || a->failed.load()) return;
    try {
        if (b->sr != a->sample_rate || b->samples != a->expectedReferenceSamples) {
            throw std::runtime_error("unpaired output callback");
        }
        for (int c = 0; c < 8; ++c) {
            if (!Has(b, a->ref[c])) throw std::runtime_error("missing reference channel");
        }
        const int frame = a->sample_rate / 100;
        const webrtc::StreamConfig renderConfig(a->sample_rate, 8);
        float* renderPtr[8];
        for (int c = 0; c < 8; ++c) renderPtr[c] = a->render[c];
        for (long i = 0; i < b->samples; ++i) {
            for (int c = 0; c < 8; ++c) a->render[c][a->renderFill] = b->read[a->ref[c]][i];
            if (++a->renderFill == frame) {
                if (a->apm->ProcessReverseStream(renderPtr, renderConfig, renderConfig, renderPtr) != 0) {
                    throw std::runtime_error("ProcessReverseStream failed");
                }
                a->renderFill = 0;
            }
        }
        a->expectedReferenceSamples = 0;
    } catch (...) {
        a->failed.store(1);
        a->active.store(0);
    }
}

}  // extern "C"
