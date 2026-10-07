#pragma once
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
