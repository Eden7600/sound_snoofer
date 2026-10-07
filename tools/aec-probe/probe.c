/* Offline echo cancellation probe: drives snoofer-aec.dll's insert stages
   with a synthetic room and checks echo reduction, pass-through, re-framing
   and unsupported-rate bypass. No Voicemeeter or audio device is used. */
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <math.h>
#include <stdio.h>
#include <string.h>

#include "pass_through.h"

/* Mirrors internal/aec/native/aec.cpp. */
typedef struct AEC AEC;
typedef struct {
    int mic[2], reference[8], strength, bypass;
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
typedef HRESULT (__cdecl *FailureFn)(AEC *, int *);
typedef HRESULT (__cdecl *ResetFn)(AEC *);

static CreateFn create;
static DestroyFn destroy;
static ConfigureFn configure;
static StatsFn stats;
static FailureFn readFailure;
static ResetFn resetFailure;
static InsertStage inputInsert, outputInsert;

enum { Channels = 8, MaxSamples = 4096, MicLeft = 2, MicRight = 3, RefLeft = 0, RefRight = 1 };
enum { EchoDelay = 40, EchoTaps = 2400 }; /* ms; 50 ms of decay at 48 kHz */

static unsigned seed;
static float noise(void) {
    seed = seed * 1664525u + 1013904223u;
    return (float)((seed >> 8) & 0xFFFF) / 32768.0f - 1.0f;
}

/* Room renders speech-like far-end audio and its echo at the microphone. */
typedef struct {
    int rate, deviceBuffer;
    long t;
    float lowpass;
    float history[48000]; /* far-end samples */
    long size;
    float taps[EchoTaps];
} Room;

static void room_init(Room *r, int rate, int channel) {
    memset(r, 0, sizeof(*r));
    r->rate = rate;
    r->size = 48000;
    seed = 12345 + 100u * (unsigned)channel;
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
    long delay = (long)r->rate * EchoDelay / 1000 + r->deviceBuffer;
    float sum = 0.0f;
    for (int i = 0; i < EchoTaps; i += 4) { /* sparse taps keep the probe fast */
        long at = r->t - 1 - delay - i;
        if (at >= 0) sum += r->taps[i] * r->history[at % r->size];
    }
    return sum;
}

static float readMic[Channels][MaxSamples], writeMic[Channels][MaxSamples];
static float readOut[Channels][MaxSamples], writeOut[Channels][MaxSamples];

static void setup_buffers(AudioBuffer *in, AudioBuffer *out, int rate, int samples) {
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
    double doubleSin, doubleCos, doubleEchoIn, doubleOutEnergy;
    long doubleSamples;
    unsigned expectedFrames;
    AECStats stats;
} Result;

/* phase is the position within the room's 2.5 s phrase cycle: the speakers
   play for the first 1.8 s and the local voice speaks from 2.0 to 2.4 s. */
static double phase(long t, int rate) { return fmod((double)t / rate, 2.5); }

/* run plays `seconds` of the room at one rate and buffer size. Voicemeeter
   calls the input insert before the output insert in each cycle. */
static int run(int rate, int samples, int seconds, int bypass, int scenario, int strength, Result *res) {
    static Room room[6];
    AEC *aec = NULL;
    AudioBuffer in, out;
    AECConfig config = {{MicLeft, MicRight}, {0, 1, 2, 3, 4, 5, 6, 7}, strength, bypass};
    long total = (long)rate * seconds, measureFrom = total - (long)rate * 5;
    long t = 0;

    memset(res, 0, sizeof(*res));
    res->otherChannelsIdentical = res->micIdentical = 1;
    if (create(&aec) != S_OK || configure(aec, &config) != S_OK) return 0;
    for (int c = 0; c < 6; ++c) { room_init(&room[c], rate, c); room[c].deviceBuffer = (scenario == 5 ? 2048 : scenario == 7 ? 512 : samples) + c * rate / 1000; }
    setup_buffers(&in, &out, rate, samples);
    while (t < total) {
        if (scenario == 7) samples = t < total / 2 ? 256 : 512;
        if (scenario == 5) {
            const int sizes[] = {64, 127, 256, 480, 512, 1024, 2048};
            samples = sizes[(t / 64) % 7];
        }
        if (samples > total - t) samples = (int)(total - t);
        in.samples = out.samples = samples;
        for (int i = 0; i < samples; ++i) {
            float speaker[6], echo = 0;
            for (int c = 0; c < 6; ++c) {
                speaker[c] = room_far(&room[c]);
                if (scenario >= 4) echo += 0.35f * room_echo(&room[c]);
            }
            if (scenario < 4) echo = room_echo(&room[0]);
            for (int c = 0; c < Channels; ++c) {
                readMic[c][i] = 0.01f * noise(); /* unrelated strip audio */
                readOut[c][i] = 0;
            }
            if (scenario == 0 || scenario == 1) {
                readOut[0][i] = speaker[0];
                readOut[1][i] = scenario == 1 ? -speaker[0] : speaker[0];
            } else if (scenario == 2 || scenario == 3) {
                readOut[scenario == 2 ? 2 : 4][i] = speaker[0];
            } else {
                for (int c = 0; c < 6; ++c) readOut[c][i] = speaker[c];
            }
            double p = phase(t + i, rate);
            float voice = ((p >= 2.0 && p < 2.4) || ((scenario == 6 || scenario == 10) && p >= 0.3 && p < 1.5)) ? 0.1f * (float)sin(2.0 * 3.14159265 * 440.0 * (t + i) / rate) : 0.0f;
            readMic[MicLeft][i] = echo + voice;
            readMic[MicRight][i] = 0.9f * echo + voice;
        }
        memset(writeMic, 0x7f, sizeof(writeMic));
        int restarting = scenario == 8 && t == (long)rate * 10;
        if (restarting) inputInsert(aec, NULL);
        inputInsert(aec, &in);
        if (scenario < 9 || t >= 3L * samples) outputInsert(aec, &out);
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
            if ((scenario == 6 || scenario == 10) && p >= 0.5 && p < 1.4) {
                double angle = 2.0 * 3.14159265 * 440.0 * t / rate;
                res->doubleSin += writeMic[MicLeft][i] * sin(angle);
                res->doubleCos += writeMic[MicLeft][i] * cos(angle);
                double echo = readMic[MicLeft][i] - 0.1 * sin(angle);
                res->doubleEchoIn += echo * echo;
                res->doubleOutEnergy += out2;
                res->doubleSamples++;
            }
            if (p < 1.8) {
                res->echoIn += in2;
                res->echoOut += out2;
            } else if (p >= 2.1 && p < 2.4) { /* the echo tail has decayed */
                res->nearIn += in2;
                res->nearOut += out2;
            }
        }
    }
    res->expectedFrames = (unsigned)(t / (rate / 100));
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
    configure = (ConfigureFn)(void *)GetProcAddress(dll, "AECConfigureV2");
    stats = (StatsFn)(void *)GetProcAddress(dll, "AECReadStats");
    readFailure = (FailureFn)(void *)GetProcAddress(dll, "AECReadFailure");
    resetFailure = (ResetFn)(void *)GetProcAddress(dll, "AECResetFailure");
    inputInsert = (InsertStage)(void *)GetProcAddress(dll, "AECInputInsert");
    outputInsert = (InsertStage)(void *)GetProcAddress(dll, "AECOutputInsert");
    if (!create || !destroy || !configure || !stats || !readFailure || !resetFailure || !inputInsert || !outputInsert) {
        fprintf(stderr, "%s lacks an expected export\n", path);
        return 2;
    }

    const struct { int rate, samples, scenario, strength; const char *name; } cases[] = {
        {48000, 256, 0, 0, "stereo 256"}, {48000, 512, 0, 0, "stereo 512"},
        {48000, 480, 1, 0, "phase-opposed stereo"},
        {48000, 256, 2, 0, "center only"}, {48000, 512, 3, 0, "surround only"},
        {48000, 256, 4, 0, "discrete 5.1"}, {48000, 2048, 4, 0, "large callback 5.1"},
        {48000, 256, 5, 0, "extreme callback jitter 5.1"},
        {48000, 256, 7, 0, "buffer transition 256 to 512"},
        {32000, 256, 4, 0, "32 kHz 5.1"}, {16000, 127, 4, 0, "16 kHz 5.1"},
        {48000, 256, 6, 0, "double-talk Strong"},
        {48000, 256, 6, 1, "double-talk Balanced"},
        {48000, 256, 6, 2, "double-talk Gentle"},
        {48000, 480, 8, 0, "5.1 convergence after stream reset"},
        {48000, 512, 9, 0, "5.1 with input pre-roll"},
        {48000, 512, 10, 0, "double-talk with input pre-roll"}
    };
    for (size_t c = 0; c < sizeof(cases)/sizeof(cases[0]); ++c) {
        char what[160];
        if (!run(cases[c].rate, cases[c].samples, cases[c].scenario >= 8 ? 30 : 20, 0, cases[c].scenario, cases[c].strength, &r)) return 2;
        double voiceDB = 10.0 * log10(r.nearOut / r.nearIn);
        printf("%s: reduction %.1f dB, voice %+.1f dB, estimated delay %d ms, frames %u/%u, failed %d\n",
               cases[c].name, reduction(&r), voiceDB, r.stats.delay_ms, r.stats.frames, r.expectedFrames, r.stats.failed);
        snprintf(what, sizeof(what), "%s: active, no lost frames, other channels identical", cases[c].name);
        check(r.stats.active && !r.stats.failed && r.stats.frames == r.expectedFrames && r.otherChannelsIdentical, what);
        snprintf(what, sizeof(what), "%s: local voice within 6 dB", cases[c].name);
        check(isfinite(voiceDB) && fabs(voiceDB) <= 6.0, what);
        if (cases[c].scenario != 6 && cases[c].scenario != 10) {
            snprintf(what, sizeof(what), "%s: at least 20 dB echo reduction", cases[c].name);
            check(reduction(&r) >= 20.0, what);
        } else {
            double amp = 2.0 * sqrt(r.doubleSin*r.doubleSin + r.doubleCos*r.doubleCos) / r.doubleSamples;
            double db = 20.0 * log10(amp / 0.1);
            double residual = r.doubleOutEnergy - (r.doubleSin*r.doubleSin + r.doubleCos*r.doubleCos) * 2.0 / r.doubleSamples;
            double echoDB = 10.0 * log10(r.doubleEchoIn / fmax(residual, 1e-20));
            printf("Double-talk voice amplitude: %+.1f dB, residual reduction %.1f dB (tone-based estimate)\n", db, echoDB);
            check(isfinite(db) && fabs(db) <= 6.0, "simultaneous local voice within 6 dB");
            check(isfinite(echoDB) && echoDB >= 10.0, "double-talk residual reduction at least 10 dB");
        }
        fflush(stdout);
    }
    if (!run(44100, 441, 2, 0, 0, 0, &r)) return 2;
    check(!r.stats.active && r.stats.sample_rate == 44100 && r.micIdentical && r.otherChannelsIdentical,
          "44.1 kHz: inactive, every channel bit-identical");
    if (!run(48000, 512, 2, 1, 0, 0, &r)) return 2;
    check(!r.stats.active && r.micIdentical && r.otherChannelsIdentical, "bypass: every channel bit-identical");
    for (int fault = 0; fault < 5; ++fault) {
        AEC *aec = NULL;
        AudioBuffer in, out;
        AECStats report;
        AECConfig config = {{MicLeft, MicRight}, {0, 1, 2, 3, 4, 5, 6, 7}, 0, 0};
        if (create(&aec) != S_OK || configure(aec, &config) != S_OK) return 2;
        setup_buffers(&in, &out, 48000, 256);
        inputInsert(aec, &in);
        outputInsert(aec, &out); /* Establish reference before strict fault checks. */
        inputInsert(aec, &in);
        if (fault == 0) inputInsert(aec, &in); /* Missing output. */
        if (fault == 1) { out.samples = 128; outputInsert(aec, &out); }
        if (fault == 2) { out.sr = 32000; outputInsert(aec, &out); }
        if (fault == 3) { out.read[4] = NULL; outputInsert(aec, &out); }
        if (fault == 4) { outputInsert(aec, &out); outputInsert(aec, &out); }
        inputInsert(aec, &in);
        stats(aec, &report);
        int same = 1;
        for (int c = 0; c < Channels; ++c) same &= memcmp(readMic[c], writeMic[c], 256*sizeof(float)) == 0;
        check(report.failed == (fault == 3) && !report.active && same, "transient gaps re-prime; invalid reference fails open");
        /* Missing output, three unpaired outputs, missing reference, unpaired output. */
        const int reasons[] = {0, 0, 0, 6, 0};
        int reason = 0;
        readFailure(aec, &reason);
        check(reason == reasons[fault], "callback fault reports its reason");
        /* A reset for a new stream rebuilds the engine; paired callbacks process again. */
        resetFailure(aec);
        setup_buffers(&in, &out, 48000, 256);
        outputInsert(aec, &out); /* Stale output before the rebuild is skipped, not a fault. */
        for (int cycle = 0; cycle < 10; ++cycle) {
            inputInsert(aec, &in);
            outputInsert(aec, &out);
        }
        stats(aec, &report);
        readFailure(aec, &reason);
        check(report.active && !report.failed && report.frames > 0 && reason == 0, "reset resumes processing on paired callbacks");
        destroy(aec);
    }

    /* Frequent gaps retain a bounded budget despite repeated APM rebuilds. */
    {
        AEC *aec = NULL;
        AudioBuffer in, out;
        AECStats report;
        AECConfig config = {{MicLeft, MicRight}, {0,1,2,3,4,5,6,7},0,0};
        if(create(&aec)!=S_OK || configure(aec,&config)!=S_OK)return 2;
        setup_buffers(&in,&out,48000,256);
        int earlyFailure=0;
        for(int n=0;n<2800;++n){
            if(n==900) for(int stable=0;stable<50;++stable){inputInsert(aec,&in);outputInsert(aec,&out);}
            inputInsert(aec,&in);
            if(n%3==0)outputInsert(aec,&out);
            stats(aec,&report);
            if(n<2600 && report.failed)earlyFailure=1;
        }
        check(!earlyFailure && report.failed,"250 ms stable pairs clear budget; repeated gaps still fail by ten seconds");
        destroy(aec);
    }

    /* Reproduce the live startup: four inputs precede the first output. */
    {
        AEC *aec = NULL;
        AudioBuffer in, out;
        AECStats report;
        AECConfig config = {{MicLeft, MicRight}, {0, 1, 2, 3, 4, 5, 6, 7}, 0, 0};
        if (create(&aec) != S_OK || configure(aec, &config) != S_OK) return 2;
        setup_buffers(&in, &out, 48000, 512);
        int same = 1;
        for (int cycle = 0; cycle < 4; ++cycle) {
            inputInsert(aec, &in);
            for (int c = 0; c < Channels; ++c)
                same &= memcmp(readMic[c], writeMic[c], 512 * sizeof(float)) == 0;
        }
        for (int c = 0; c < Channels; ++c) in.write[c] = in.read[c];
        inputInsert(aec, &in);
        for (int c = 0; c < Channels; ++c) {
            same &= memcmp(readMic[c], writeMic[c], 512 * sizeof(float)) == 0;
            in.write[c] = writeMic[c];
        }
        stats(aec, &report);
        check(same && !report.failed && !report.active,
              "startup input pre-roll passes through without a false failure");
        for (int cycle = 0; cycle < 10; ++cycle) {
            outputInsert(aec, &out);
            inputInsert(aec, &in);
        }
        stats(aec, &report);
        check(report.active && !report.failed && report.frames > 0, "first reference starts cancellation after pre-roll");
        inputInsert(aec, NULL);
        for (int cycle = 0; cycle < 470; ++cycle) {
            inputInsert(aec, &in);
            stats(aec, &report);
            if(cycle < 400) check(!report.failed, "startup tolerates gaps beyond one second");
        }
        stats(aec, &report);
        int reason = 0;
        readFailure(aec, &reason);
        check(report.failed && !report.active && reason == 1, "missing startup reference fails after five seconds");
        destroy(aec);
    }

    /* Drive the production monitor, not just the DLL entry points. */
    for (long event = 1; event <= 3; ++event) {
        AEC *aec = NULL;
        AudioBuffer in, out;
        AECStats report;
        AECConfig config = {{MicLeft, MicRight}, {0, 1, 2, 3, 4, 5, 6, 7}, 0, 0};
        if (create(&aec) != S_OK || configure(aec, &config) != S_OK) return 2;
        setup_buffers(&in, &out, 48000, 256);
        input_stage = inputInsert;
        output_stage = outputInsert;
        stage_context = aec;
        for (int failed = 0; failed <= 1; ++failed) {
            observe(NULL, 10, &in, 1);
            observe(NULL, 11, &out, 1);
            observe(NULL, 10, &in, 1); /* Leave an input waiting for output. */
            if (failed) for(int n=0;n<940;++n) observe(NULL, 10, &in, 1); /* Prolonged absence latches. */
            observe(NULL, event, NULL, 0);
            observe(NULL, event, NULL, 0); /* Repeated events are harmless. */
            if (failed) observe(NULL, 11, &out, 1); /* Late output cannot use old state. */
            stats(aec, &report);
            check(!report.active && !report.failed && report.delay_ms == -1 && report.erle_centi_db == -1,
                  "native lifecycle clears failure and stale metrics before new input");
            for (int cycle = 0; cycle < 10; ++cycle) {
                observe(NULL, 10, &in, 1);
                observe(NULL, 11, &out, 1);
            }
            stats(aec, &report);
            check(report.active && !report.failed && report.frames > 0,
                  "production callback resumes AEC after stream boundary");
        }
        input_stage = output_stage = NULL;
        stage_context = NULL;
        destroy(aec);
    }

    FreeLibrary(dll);
    return failures ? 1 : 0;
}
