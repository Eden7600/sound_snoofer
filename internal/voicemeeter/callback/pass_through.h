#pragma once
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <string.h>

/* Windows x64 ABI from the official VoicemeeterRemote.h. No DLL redistribution. */
#include "audio_buffer.h"
typedef long (__stdcall *Callback)(void *, long, void *, long);
typedef long (__stdcall *Register)(long, Callback, void *, char *);
typedef long (__stdcall *Call)(void);
static volatile LONG starting, ending, change, buffers, synced, invalid, unknown;
static volatile LONG sample_rate, frame_size, channels;

/* Optional insert stages (echo cancellation). They change only while the
   callback is not registered, so the audio thread reads them without locks. */
static InsertStage input_stage, output_stage;
static void *stage_context;

/* Pre-insert peaks of the physical input channels (Potato: 5 strips x 2),
   stored as float bits, and the input inserts seen; the owner's worker takes
   and clears them. Non-negative floats order like their bit patterns. */
#define PEAK_CHANNELS 16
static volatile LONG input_peaks[PEAK_CHANNELS], input_inserts;

/* Both inserts have equal input/output channel counts and every pointer set. */
static int valid(const AudioBuffer *b) {
    long i;
    if (!audio_buffer_valid_shape(b)) return 0;
    for (i = 0; i < b->outputs; ++i) {
        if (!audio_buffer_has_channel(b, i)) return 0;
    }
    return 1;
}

/* Raises each physical channel's stored peak before any insert stage runs.
   Lock-free and bounded by samples x PEAK_CHANNELS. */
static void record_peaks(const AudioBuffer *b) {
    long c, i, count = b->inputs < PEAK_CHANNELS ? b->inputs : PEAK_CHANNELS;
    for (c = 0; c < count; ++c) {
        float peak = 0.0f;
        LONG bits, stored;
        for (i = 0; i < b->samples; ++i) {
            float x = b->read[c][i];
            if (x < 0.0f) x = -x;
            if (x > peak) peak = x; /* NaN compares false and is skipped. */
        }
        memcpy(&bits, &peak, sizeof(bits));
        stored = input_peaks[c];
        while (bits > stored) {
            LONG prior = InterlockedCompareExchange(&input_peaks[c], bits, stored);
            if (prior == stored) break;
            stored = prior;
        }
    }
    InterlockedIncrement(&input_inserts);
}

static long __stdcall observe(void *user, long command, void *data, long sync) {
    AudioBuffer *b = (AudioBuffer *)data;
    (void)user;
    switch (command) {
    case 1: case 2: case 3:
        /* Invalidate insert history before any buffer from the new stream. */
        if (input_stage) input_stage(stage_context, NULL);
        if (command == 1) InterlockedIncrement(&starting);
        if (command == 2) InterlockedIncrement(&ending);
        if (command == 3) InterlockedIncrement(&change);
        return 0;
    case 10: /* Input insert, registered only with an input stage. */
        if (!valid(b)) {
            InterlockedIncrement(&invalid);
            return 0;
        }
        record_peaks(b);
        if (input_stage) {
            input_stage(stage_context, b); /* Writes every channel itself. */
        } else {
            audio_buffer_copy(b);
        }
        return 0;
    case 11: break; /* Output insert. */
    default: InterlockedIncrement(&unknown); return 0;
    }
    if (!valid(b)) {
        InterlockedIncrement(&invalid);
        return 0;
    }
    if (output_stage) output_stage(stage_context, b); /* Reads only. */
    audio_buffer_copy(b);
    InterlockedExchange(&sample_rate, b->sr);
    InterlockedExchange(&frame_size, b->samples);
    InterlockedExchange(&channels, b->outputs);
    InterlockedIncrement(&buffers);
    if (sync == 1) InterlockedIncrement(&synced);
    return 0;
}

#define READ(x) InterlockedCompareExchange(&(x), 0, 0)
