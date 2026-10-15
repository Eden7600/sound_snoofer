#pragma once
#include <string.h>
/* Voicemeeter's audio callback buffer, Windows x64 ABI from the official
   VoicemeeterRemote.h (VBVMR_T_AUDIOBUFFER). Shared by the monitor and the
   echo canceller; no Voicemeeter code is redistributed. */
typedef struct {
    long sr, samples, inputs, outputs;
    float *read[128], *write[128];
} AudioBuffer;

/* An insert stage called from the monitor's callback on Voicemeeter's audio
   thread. It must not block, lock or allocate after warm-up, and it must
   write every output channel (copying input where it does not process).
   The input stage receives NULL at stream start/end/change to invalidate
   history synchronously. No audio is accessed for this lifecycle signal. */
typedef void (__stdcall *InsertStage)(void *context, AudioBuffer *buffer);

/* Shape checks bound all channel/sample work; sample rate is caller policy. */
static inline int audio_buffer_valid_shape(const AudioBuffer *b) {
    return b && b->samples > 0 && b->samples <= 65536 &&
        b->inputs > 0 && b->inputs <= 128 && b->outputs == b->inputs;
}

static inline int audio_buffer_has_channel(const AudioBuffer *b, long channel) {
    return audio_buffer_valid_shape(b) && channel >= 0 && channel < b->inputs &&
        b->read[channel] && b->write[channel];
}

/* Missing pairs stay untouched. Callers that require every channel must reject
   them before invoking inserts or reading peaks. memmove also permits overlap. */
static inline void audio_buffer_copy(AudioBuffer *b) {
    long channel;
    if (!audio_buffer_valid_shape(b)) return;
    for (channel = 0; channel < b->outputs; ++channel) {
        if (audio_buffer_has_channel(b, channel) && b->read[channel] != b->write[channel])
            memmove(b->write[channel], b->read[channel], (size_t)b->samples * sizeof(float));
    }
}
