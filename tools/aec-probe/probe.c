/* Offline echo cancellation probe: drives snoofer-aec.dll's insert stages
   with a synthetic room and checks echo reduction, pass-through, re-framing
   and unsupported-rate bypass. No Voicemeeter or audio device is used. */
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <math.h>
#include <stdio.h>
#include <string.h>

#include "audio_buffer.h"

/* Mirrors internal/aec/native/aec.cpp. */
typedef struct AEC AEC;
typedef struct {
    int mic[2], reference[2], strength, bypass;
} AECConfig;
typedef struct {
    int active, sample_rate, erle_centi_db, delay_ms;
    unsigned frames;
    int failed;
} AECStats;

typedef HRESULT (__cdecl *CreateFn)(AEC **);
typedef HRESULT (__cdecl *DestroyFn)(AEC *);
typedef HRESULT (__cdecl *ConfigureFn)(AEC *, const AECConfig *);
typedef HRESULT (__cdecl *StatsFn)(AEC *, AECStats *);

static CreateFn create;
static DestroyFn destroy;
static ConfigureFn configure;
static StatsFn stats;
static InsertStage inputInsert, outputInsert;

enum { Channels = 8, MaxSamples = 1024, MicLeft = 2, MicRight = 3, RefLeft = 0, RefRight = 1 };
enum { EchoDelay = 40, EchoTaps = 2400 }; /* ms; 50 ms of decay at 48 kHz */

static unsigned seed;
static float noise(void) {
    seed = seed * 1664525u + 1013904223u;
    return (float)((seed >> 8) & 0xFFFF) / 32768.0f - 1.0f;
}

/* Room renders speech-like far-end audio and its echo at the microphone. */
typedef struct {
    int rate;
    long t;
    float lowpass;
    float *history; /* far-end samples, ring of `size` */
    long size;
    float taps[EchoTaps];
} Room;

static void room_init(Room *r, int rate) {
    static float history[48000];
    memset(r, 0, sizeof(*r));
    r->rate = rate;
    r->history = history;
    r->size = 48000;
    memset(history, 0, sizeof(history));
    seed = 12345;
    for (int i = 0; i < EchoTaps; ++i) {
        r->taps[i] = 0.5f * noise() * expf(-(float)i / (rate * 0.008f));
    }
}

/* far is the next sample the speakers play: low-passed noise with syllable
   envelopes and pauses. */
static float room_far(Room *r) {
    double s = (double)r->t / r->rate;
    double syllable = sin(2.0 * 3.14159265 * 3.0 * s);
    double phrase = fmod(s, 2.5) < 1.8 ? 1.0 : 0.0;
    r->lowpass += 0.25f * (noise() - r->lowpass);
    float v = (float)(0.3 * syllable * syllable * phrase) * r->lowpass;
    r->history[r->t % r->size] = v;
    r->t++;
    return v;
}

/* echo is what the microphone hears now: delayed far-end through the room. */
static float room_echo(Room *r) {
    long delay = (long)r->rate * EchoDelay / 1000;
    float sum = 0.0f;
    for (int i = 0; i < EchoTaps; i += 4) { /* sparse taps keep the probe fast */
        long at = r->t - 1 - delay - i;
        if (at >= 0) sum += r->taps[i] * r->history[at % r->size];
    }
    return sum;
}

static float readMic[Channels][MaxSamples], writeMic[Channels][MaxSamples];
static float readOut[Channels][MaxSamples], writeOut[Channels][MaxSamples];

static void buffers(AudioBuffer *in, AudioBuffer *out, int rate, int samples) {
    memset(in, 0, sizeof(*in));
    memset(out, 0, sizeof(*out));
    in->sr = out->sr = rate;
    in->samples = out->samples = samples;
    in->inputs = in->outputs = out->inputs = out->outputs = Channels;
    for (int c = 0; c < Channels; ++c) {
        in->read[c] = readMic[c];
        in->write[c] = writeMic[c];
        out->read[c] = readOut[c];
        out->write[c] = writeOut[c];
    }
}

typedef struct {
    double echoIn, echoOut; /* mic energy in and out while only the speakers play, final 5 s */
    double nearIn, nearOut; /* mic energy in and out while only the local voice speaks, final 5 s */
    int otherChannelsIdentical, micIdentical;
    AECStats stats;
} Result;

/* phase is the position within the room's 2.5 s phrase cycle: the speakers
   play for the first 1.8 s and the local voice speaks from 2.0 to 2.4 s. */
static double phase(long t, int rate) { return fmod((double)t / rate, 2.5); }

/* run plays `seconds` of the room at one rate and buffer size. Voicemeeter
   calls the input insert before the output insert in each cycle. */
static int run(int rate, int samples, int seconds, int bypass, Result *res) {
    static Room room;
    AEC *aec = NULL;
    AudioBuffer in, out;
    AECConfig config = {{MicLeft, MicRight}, {RefLeft, RefRight}, 0, bypass};
    long total = (long)rate * seconds, measureFrom = total - (long)rate * 5;
    long t = 0;

    memset(res, 0, sizeof(*res));
    res->otherChannelsIdentical = res->micIdentical = 1;
    if (create(&aec) != S_OK || configure(aec, &config) != S_OK) return 0;
    room_init(&room, rate);
    buffers(&in, &out, rate, samples);
    while (t < total) {
        for (int i = 0; i < samples; ++i) {
            float speaker = room_far(&room);
            float echo = room_echo(&room);
            for (int c = 0; c < Channels; ++c) {
                readMic[c][i] = 0.01f * noise(); /* unrelated strip audio */
                readOut[c][i] = c == RefLeft || c == RefRight ? speaker : 0.02f * noise();
            }
            double p = phase(t + i, rate);
            float voice = p >= 2.0 && p < 2.4 ? 0.1f * (float)sin(2.0 * 3.14159265 * 440.0 * (t + i) / rate) : 0.0f;
            readMic[MicLeft][i] = echo + voice;
            readMic[MicRight][i] = 0.9f * echo + voice;
        }
        memset(writeMic, 0x7f, sizeof(writeMic));
        inputInsert(aec, &in);
        outputInsert(aec, &out);
        for (int c = 0; c < Channels; ++c) {
            int same = memcmp(readMic[c], writeMic[c], samples * sizeof(float)) == 0;
            if (c == MicLeft || c == MicRight) {
                res->micIdentical &= same;
            } else {
                res->otherChannelsIdentical &= same;
            }
        }
        for (int i = 0; i < samples; ++i, ++t) {
            double p = phase(t, rate);
            double in2 = (double)readMic[MicLeft][i] * readMic[MicLeft][i];
            double out2 = (double)writeMic[MicLeft][i] * writeMic[MicLeft][i];
            if (t < measureFrom) continue;
            if (p < 1.8) {
                res->echoIn += in2;
                res->echoOut += out2;
            } else if (p >= 2.1 && p < 2.4) { /* the echo tail has decayed */
                res->nearIn += in2;
                res->nearOut += out2;
            }
        }
    }
    stats(aec, &res->stats);
    destroy(aec);
    return 1;
}

static int failures;
static void check(int ok, const char *what) {
    printf("%s: %s\n", ok ? "PASS" : "FAIL", what);
    if (!ok) failures++;
}

static double reduction(const Result *r) {
    return 10.0 * log10(r->echoIn / (r->echoOut > 1e-20 ? r->echoOut : 1e-20));
}

int main(int argc, char **argv) {
    const char *path = argc > 1 ? argv[1] : "bin\\snoofer-aec.dll";
    HMODULE dll = LoadLibraryA(path);
    Result r;

    if (!dll) {
        fprintf(stderr, "cannot load %s (error %lu)\n", path, GetLastError());
        return 2;
    }
    create = (CreateFn)(void *)GetProcAddress(dll, "AECCreate");
    destroy = (DestroyFn)(void *)GetProcAddress(dll, "AECDestroy");
    configure = (ConfigureFn)(void *)GetProcAddress(dll, "AECConfigure");
    stats = (StatsFn)(void *)GetProcAddress(dll, "AECReadStats");
    inputInsert = (InsertStage)(void *)GetProcAddress(dll, "AECInputInsert");
    outputInsert = (InsertStage)(void *)GetProcAddress(dll, "AECOutputInsert");
    if (!create || !destroy || !configure || !stats || !inputInsert || !outputInsert) {
        fprintf(stderr, "%s lacks an expected export\n", path);
        return 2;
    }

    for (int samples = 256; samples <= 512; samples *= 2) {
        char what[160];
        if (!run(48000, samples, 20, 0, &r)) return 2;
        printf("48 kHz, %d-sample buffers: reduction %.1f dB, voice %+.1f dB, engine ERLE %.2f dB, delay %d ms, frames %u, failed %d\n",
               samples, reduction(&r), 10.0 * log10(r.nearOut / r.nearIn), r.stats.erle_centi_db / 100.0, r.stats.delay_ms, r.stats.frames, r.stats.failed);
        snprintf(what, sizeof(what), "%d-sample buffers: active and reduces echo by at least 20 dB", samples);
        check(r.stats.active && !r.stats.failed && reduction(&r) >= 20.0, what);
        snprintf(what, sizeof(what), "%d-sample buffers: local voice kept within 6 dB", samples);
        check(10.0 * log10(r.nearIn / r.nearOut) <= 6.0, what);
        snprintf(what, sizeof(what), "%d-sample buffers: other channels bit-identical", samples);
        check(r.otherChannelsIdentical, what);
        snprintf(what, sizeof(what), "%d-sample buffers: every 10 ms frame processed", samples);
        check(r.stats.frames >= 20 * 100 - 2, what);
    }

    if (!run(44100, 441, 2, 0, &r)) return 2;
    check(!r.stats.active && r.stats.sample_rate == 44100 && r.micIdentical && r.otherChannelsIdentical,
          "44.1 kHz: inactive, every channel bit-identical");

    if (!run(48000, 512, 2, 1, &r)) return 2;
    check(!r.stats.active && r.micIdentical && r.otherChannelsIdentical, "bypass: inactive, every channel bit-identical");

    FreeLibrary(dll);
    return failures ? 1 : 0;
}
