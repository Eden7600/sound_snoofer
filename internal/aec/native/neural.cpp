// LocalVQE insert: callback-owned framing, bounded SPSC queues, worker-only inference.
#include <windows.h>
#include <atomic>
#include <array>
#include <algorithm>
#include <cmath>
#include <cstring>
#include <memory>
#include <thread>
#include "common_audio/resampler/push_sinc_resampler.h"
#include "localvqe_api.h"
#include "fullband.h"
extern "C" {
#include "audio_buffer.h"
}

namespace {
constexpr int kHop = 256, kMaxBlock = 2048, kQueue = 32;
struct Config { int mic[2], reference[8], strength, bypass; };
struct Stats { int active, sample_rate, erle, delay; unsigned frames; int failed; };
struct Input { unsigned generation; int rate; float mic[480], ref[480]; };
struct Output { unsigned generation; int count; uint64_t start; float data[768]; };

// Single producer and single consumer; indices never reset across generations.
template<class T> struct Queue {
    std::array<T, kQueue> slots{};
    std::atomic<unsigned> head{0}, tail{0};
    bool Push(const T& value) {
        unsigned h = head.load(std::memory_order_relaxed);
        if (h - tail.load(std::memory_order_acquire) == kQueue) return false;
        slots[h % kQueue] = value;
        head.store(h + 1, std::memory_order_release);
        return true;
    }
    // Consumer only: release published slots without touching the producer index.
    void Discard() {
        tail.store(head.load(std::memory_order_acquire), std::memory_order_release);
    }
    bool Pop(T& value) {
        unsigned t = tail.load(std::memory_order_relaxed);
        if (t == head.load(std::memory_order_acquire)) return false;
        value = slots[t % kQueue];
        tail.store(t + 1, std::memory_order_release);
        return true;
    }
};
struct Engine {
    localvqe_ctx_t model = 0;
    bool fullband = false; // Immutable before the worker starts.
    std::atomic<bool> stop{false}, bypass{true};
    std::atomic<unsigned> generation{1}, revision{0}, frames{0};
    std::atomic<int> mic[2], refs[8], rate{0}, active{0}, latency{0};
    std::atomic<uint64_t> failure{0};
    Queue<Input> input;
    Queue<Output> output;
    std::thread worker;
    // Audio callback thread only.
    unsigned applied = 0, recoveryGeneration = 0;
    bool recovering = false;
    int recoverySamples = 0, stableSamples = 0;
    int blockSize = 0;
    int m[2]{}, r[8]{}, sampleRate = 0, pending = 0, fill = 0, startup = 0;
    bool referenceReady = false, haveOutput = false;
    uint64_t cursor = 0;
    float capture[kMaxBlock]{}, processed[kMaxBlock]{};
    Input packet{};
    Output current{};
    ~Engine() {
        stop.store(true);
        if (worker.joinable()) worker.join();
        if (model) localvqe_free(model);
    }
    int Failed() const {
        uint64_t f = failure.load();
        return unsigned(f >> 32) == generation.load() ? int(f & 0xffffffffu) : 0;
    }
    void Fail(unsigned gen, int reason) {
        uint64_t previous = failure.load();
        // An old worker result must never erase a newer stream's failure.
        while (unsigned(previous >> 32) <= gen && gen == generation.load()) {
            if (failure.compare_exchange_weak(previous, (uint64_t(gen) << 32) | unsigned(reason))) return;
        }
    }
    unsigned Reset() { active.store(0); return generation.fetch_add(1) + 1; }
    // Callback thread only: discard both timelines, never pair an old mic with
    // a later reference. Further gaps retain the same bounded recovery budget.
    void Discontinuity() {
        if (!recovering) recoverySamples = 0;
        recovering = true;
        stableSamples = 0;
        recoveryGeneration = Reset();
    }
    void Run() {
        unsigned gen = 0;
        int accumulated = 0, factor = 1;
        uint64_t emitted = 0;
        float mic16[160]{}, ref16[160]{}, micHop[256]{}, refHop[256]{}, result[256]{};
        std::unique_ptr<webrtc::PushSincResampler> downMic, downRef, up;
        std::unique_ptr<snoofer::Fullband> upper;
        Input p{};
        while (!stop.load()) {
            if (!input.Pop(p)) { Sleep(1); continue; }
            if (p.generation != generation.load() || Failed()) continue;
            try {
                if (gen != p.generation) {
                    gen = p.generation;
                    localvqe_reset(model);
                    factor = p.rate / 16000;
                    accumulated = 0;
                    emitted = 0;
                    if (fullband) upper = std::make_unique<snoofer::Fullband>();
                    if (factor > 1) {
                        downMic = std::make_unique<webrtc::PushSincResampler>(160 * factor, 160);
                        downRef = std::make_unique<webrtc::PushSincResampler>(160 * factor, 160);
                        up = std::make_unique<webrtc::PushSincResampler>(256, 256 * factor);
                    }
                }
                if (upper) upper->Push(p.mic,p.ref);
                if (factor > 1) {
                    downMic->Resample(p.mic, 160 * factor, mic16, 160);
                    downRef->Resample(p.ref, 160 * factor, ref16, 160);
                } else {
                    std::memcpy(mic16, p.mic, sizeof(mic16));
                    std::memcpy(ref16, p.ref, sizeof(ref16));
                }
                for (int i = 0; i < 160; ++i) {
                    micHop[accumulated] = mic16[i];
                    refHop[accumulated++] = ref16[i];
                    if (accumulated != kHop) continue;
                    accumulated = 0;
                    if (localvqe_process_frame_f32(model, micHop, refHop, kHop, result)) {
                        Fail(gen, 3); break;
                    }
                    Output out{};
                    out.generation = gen;
                    out.count = kHop * factor;
                    out.start = emitted;
                    emitted += out.count;
                    if (factor > 1) up->Resample(result, kHop, out.data, out.count);
                    else std::memcpy(out.data, result, sizeof(result));
                    if (upper) upper->Mix(out.start,out.data,out.count);
                    bool finite = true;
                    for (int j = 0; j < out.count; ++j) finite &= std::isfinite(out.data[j]);
                    if (!finite) { Fail(gen, 10); break; }
                    if (gen != generation.load()) break;
                    if (!output.Push(out)) { Fail(gen, 9); break; }
                    frames.fetch_add(1);
                }
            } catch (...) { Fail(p.generation, 8); }
        }
    }
};
bool Valid(AudioBuffer* b) {
    return b && b->samples > 0 && b->samples <= 65536 && b->inputs > 0 && b->inputs <= 128 && b->outputs == b->inputs;
}
bool Has(AudioBuffer* b, int c) { return c >= 0 && c < b->inputs && b->read[c] && b->write[c]; }
void Pass(AudioBuffer* b) {
    for (int c = 0; c < b->outputs; ++c)
        if (b->read[c] && b->write[c] && b->read[c] != b->write[c])
            std::memmove(b->write[c], b->read[c], b->samples * sizeof(float));
}
}
static HRESULT CreateNeural(Engine** result, const char* path, bool fullband) {
    if (!result || !path) return E_INVALIDARG;
    *result = nullptr;
    try {
        auto e = std::make_unique<Engine>();
        e->fullband = fullband;
        auto options = localvqe_options_new();
        if (!options) return E_OUTOFMEMORY;
        localvqe_options_set_model_path(options, path);
        localvqe_options_set_backend(options, "CPU");
        localvqe_options_set_threads(options, 4);
        e->model = localvqe_new_with_options(options);
        localvqe_options_free(options);
        if (!e->model || localvqe_sample_rate(e->model) != 16000 || localvqe_hop_length(e->model) != 256) return E_FAIL;
        e->worker = std::thread([p = e.get()] { p->Run(); });
        *result = e.release();
        return S_OK;
    } catch (...) { return E_FAIL; }
}
extern "C" {
__declspec(dllexport) HRESULT __cdecl AECNeuralCreate(Engine** result, const char* path) {
    return CreateNeural(result,path,false);
}
__declspec(dllexport) HRESULT __cdecl AECNeuralFullbandCreate(Engine** result, const char* path) {
    return CreateNeural(result,path,true);
}
__declspec(dllexport) HRESULT __cdecl AECDestroy(Engine* e) { delete e; return S_OK; }
__declspec(dllexport) HRESULT __cdecl AECConfigureV2(Engine* e, const Config* c) {
    if (!e || !c) return E_INVALIDARG;
    for (int v : c->mic) if (v < 0 || v >= 128) return E_INVALIDARG;
    for (int v : c->reference) if (v < 0 || v >= 128) return E_INVALIDARG;
    // Seqlock: a callback encountering an in-progress update simply passes through.
    e->revision.fetch_add(1);
    e->bypass.store(c->bypass != 0);
    for (int i = 0; i < 2; ++i) e->mic[i].store(c->mic[i]);
    for (int i = 0; i < 8; ++i) e->refs[i].store(c->reference[i]);
    e->Reset();
    e->revision.fetch_add(1);
    return S_OK;
}
__declspec(dllexport) HRESULT __cdecl AECReadStats(Engine* e, Stats* s) {
    if (!e || !s) return E_INVALIDARG;
    int failed = e->Failed();
    *s = {e->active.load() && !failed && !e->bypass.load(), e->rate.load(), -1, -1, e->frames.load(), failed != 0};
    return S_OK;
}
__declspec(dllexport) HRESULT __cdecl AECReadLatency(Engine* e, int* milliseconds) {
    if (!e || !milliseconds) return E_INVALIDARG;
    *milliseconds = e->latency.load(); return S_OK;
}
__declspec(dllexport) HRESULT __cdecl AECReadFailure(Engine* e, int* reason) {
    if (!e || !reason) return E_INVALIDARG;
    *reason = e->Failed(); return S_OK;
}
__declspec(dllexport) HRESULT __cdecl AECResetFailure(Engine* e) {
    if (!e) return E_INVALIDARG;
    e->Reset(); return S_OK;
}
__declspec(dllexport) void __stdcall AECInputInsert(void* context, AudioBuffer* b) {
    auto e = static_cast<Engine*>(context);
    if (e && !b) { e->Reset(); e->rate.store(0); return; }
    if (!Valid(b)) return;
    Pass(b);
    if (!e) return;
    e->rate.store(b->sr);
    unsigned rev = e->revision.load();
    if (rev % 2 || e->bypass.load()) { e->active.store(0); return; }
    if (e->sampleRate != b->sr || b->samples > e->blockSize) e->Reset();
    unsigned gen = e->generation.load();
    if (!e->Failed() && e->applied == gen && e->pending && e->referenceReady) {
        e->Discontinuity();
        gen = e->generation.load();
    }
    if (e->applied != gen) {
        // A control/lifecycle reset starts fresh; our own resync keeps its budget.
        if (gen != e->recoveryGeneration) e->recovering = false;
        for (int i = 0; i < 2; ++i) e->m[i] = e->mic[i].load();
        for (int i = 0; i < 8; ++i) e->r[i] = e->refs[i].load();
        if (rev != e->revision.load()) return;
        // Pre-roll does not consume output. Release old results now so repeated
        // resets cannot fill the queue before consumption starts. A concurrently
        // published old result is still rejected by its generation below.
        e->output.Discard();
        e->applied = gen; e->sampleRate = b->sr; e->blockSize = b->samples;
        e->latency.store(b->sr > 0 ? 80 + (1000 * e->blockSize + b->sr - 1) / b->sr + (b->sr > 16000 ? 2 : 0) + (e->fullband ? 2 : 0) : 0);
        e->pending = e->fill = e->startup = 0;
        e->cursor = 0; e->referenceReady = e->haveOutput = false;
        e->active.store(0);
    }
    if (e->fullband && b->sr != 48000) return;
    if (e->Failed() || (b->sr != 16000 && b->sr != 32000 && b->sr != 48000)) return;
    if (b->samples > kMaxBlock) { e->Fail(gen, 11); return; }
    if (e->recovering && (e->recoverySamples += b->samples) >= 10 * b->sr) {
        e->Fail(gen, 1); return;
    }
    if (!Has(b, e->m[0]) || !Has(b, e->m[1])) { e->Fail(gen, 2); return; }
    if (!e->referenceReady && (e->startup += b->samples) > 5 * b->sr) { e->Fail(gen, 1); return; }
    // Preserve aliased microphone input before committing any processed output.
    for (int i = 0; i < b->samples; ++i) {
        e->capture[i] = .5f * b->read[e->m[0]][i] + .5f * b->read[e->m[1]][i];
        if (!std::isfinite(e->capture[i])) { e->Fail(gen, 10); return; }
    }
    e->pending = b->samples;
    const uint64_t latency = uint64_t(b->sr) * 64 / 1000 + e->blockSize;
    for (int i = 0; i < b->samples; ++i) {
        uint64_t time = e->cursor + i;
        if (time < latency) { e->processed[i] = e->capture[i]; continue; }
        uint64_t wanted = time - latency;
        while (!e->haveOutput || e->current.generation != gen || wanted >= e->current.start + e->current.count) {
            e->haveOutput = e->output.Pop(e->current);
            if (!e->haveOutput) { e->Fail(gen, 4); return; }
        }
        if (wanted < e->current.start) { e->Fail(gen, 4); return; }
        e->processed[i] = e->current.data[wanted - e->current.start];
    }
    if (gen != e->generation.load() || e->Failed() || e->bypass.load()) return;
    // Pre-roll stays bit-identical, including distinct stereo microphone channels.
    for (int i = 0; i < b->samples; ++i) {
        if (e->cursor + i < latency) continue;
        b->write[e->m[0]][i] = b->write[e->m[1]][i] = e->processed[i];
        e->active.store(1);
    }
}
__declspec(dllexport) void __stdcall AECOutputInsert(void* context, AudioBuffer* b) {
    auto e = static_cast<Engine*>(context);
    if (!e || !Valid(b) || e->bypass.load() || e->Failed()) return;
    unsigned gen = e->generation.load();
    if (e->fullband && b->sr != 48000) return;
    if (e->applied != gen || (e->sampleRate != 16000 && e->sampleRate != 32000 && e->sampleRate != 48000)) return;
    if (b->sr != e->sampleRate || b->samples != e->pending) {
        e->Discontinuity(); return;
    }
    for (int c : e->r) if (!Has(b, c)) { e->Fail(gen, 6); return; }
    for (int i = 0; i < b->samples; ++i) {
        float ref = 0;
        for (int c : e->r) ref += b->read[c][i] / 8.f;
        if (!std::isfinite(ref)) { e->Fail(gen, 10); return; }
        e->packet.mic[e->fill] = e->capture[i];
        e->packet.ref[e->fill++] = ref;
        if (e->fill == e->sampleRate / 100) {
            e->packet.generation = gen; e->packet.rate = e->sampleRate;
            if (!e->input.Push(e->packet)) { e->Fail(gen, 9); return; }
            e->fill = 0;
        }
    }
    e->cursor += b->samples;
    e->pending = 0; e->referenceReady = true;
    if (e->recovering && (e->stableSamples += b->samples) >= e->sampleRate / 4) {
        e->recovering = false;
    }
}
}
