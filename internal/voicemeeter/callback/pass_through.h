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

/* Both inserts have equal input/output channel counts and every pointer set. */
static int valid(const AudioBuffer *b) {
    long i;
    if (!b || b->samples < 1 || b->samples > 65536 ||
        b->inputs < 1 || b->inputs > 128 || b->outputs != b->inputs) {
        return 0;
    }
    for (i = 0; i < b->outputs; ++i) {
        if (!b->read[i] || !b->write[i]) return 0;
    }
    return 1;
}

static void pass_through(AudioBuffer *b) {
    long i;
    for (i = 0; i < b->outputs; ++i) {
        if (b->read[i] != b->write[i])
            memmove(b->write[i], b->read[i], (size_t)b->samples * sizeof(float));
    }
}

static long __stdcall observe(void *user, long command, void *data, long sync) {
    AudioBuffer *b = (AudioBuffer *)data;
    (void)user;
    switch (command) {
    case 1: InterlockedIncrement(&starting); return 0;
    case 2: InterlockedIncrement(&ending); return 0;
    case 3: InterlockedIncrement(&change); return 0;
    case 10: /* Input insert, registered only with an input stage. */
        if (!valid(b)) {
            InterlockedIncrement(&invalid);
            return 0;
        }
        if (input_stage) {
            input_stage(stage_context, b); /* Writes every channel itself. */
        } else {
            pass_through(b);
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
    pass_through(b);
    InterlockedExchange(&sample_rate, b->sr);
    InterlockedExchange(&frame_size, b->samples);
    InterlockedExchange(&channels, b->outputs);
    InterlockedIncrement(&buffers);
    if (sync == 1) InterlockedIncrement(&synced);
    return 0;
}

#define READ(x) InterlockedCompareExchange(&(x), 0, 0)
