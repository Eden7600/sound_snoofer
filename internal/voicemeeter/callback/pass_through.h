#pragma once
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <string.h>

/* Windows x64 ABI from the official VoicemeeterRemote.h. No DLL redistribution. */
typedef struct {
    long sr, samples, inputs, outputs;
    float *read[128], *write[128];
} AudioBuffer;
typedef long (__stdcall *Callback)(void *, long, void *, long);
typedef long (__stdcall *Register)(long, Callback, void *, char *);
typedef long (__stdcall *Call)(void);
static volatile LONG starting, ending, change, buffers, synced, invalid, unknown;
static volatile LONG sample_rate, frame_size, channels;

static long __stdcall observe(void *user, long command, void *data, long sync) {
    AudioBuffer *b;
    long i;
    (void)user;
    switch (command) {
    case 1: InterlockedIncrement(&starting); return 0;
    case 2: InterlockedIncrement(&ending); return 0;
    case 3: InterlockedIncrement(&change); return 0;
    case 11: break; /* Output insert: equal input/output channel counts. */
    default: InterlockedIncrement(&unknown); return 0;
    }
    b = (AudioBuffer *)data;
    if (!b || b->samples < 1 || b->samples > 65536 ||
        b->inputs < 1 || b->inputs > 128 || b->outputs != b->inputs) {
        InterlockedIncrement(&invalid);
        return 0;
    }
    for (i = 0; i < b->outputs; ++i) {
        if (!b->read[i] || !b->write[i]) {
            InterlockedIncrement(&invalid);
            return 0;
        }
    }
    for (i = 0; i < b->outputs; ++i) {
        if (b->read[i] != b->write[i])
            memmove(b->write[i], b->read[i], (size_t)b->samples * sizeof(float));
    }
    InterlockedExchange(&sample_rate, b->sr);
    InterlockedExchange(&frame_size, b->samples);
    InterlockedExchange(&channels, b->outputs);
    InterlockedIncrement(&buffers);
    if (sync == 1) InterlockedIncrement(&synced);
    return 0;
}

#define READ(x) InterlockedCompareExchange(&(x), 0, 0)
