# Design

## 1. Signal path
Voicemeeter's Remote API gives a process one callback registration with a mode bitmask: input insert (1), output insert (2) or main (4). `Start`, `Stop` and `Unregister` take no handle. Per callback cycle, Voicemeeter calls input insert (command 10) before mixing and output insert (command 11) after mixing, before the bus masters.

- **Reference:** the output insert copies the two channels of the bus that plays to the speakers into a reference FIFO, downmixed to mono. That is the playback destination the audio plugin resolves (§4).
- **Capture:** the input insert takes the managed mic strip's channels, runs AEC3 against the reference and writes the result back to the strip. Other input channels pass through bit-identically, as the output insert already does.
- **Causality:** the reference from cycle *n−1* and earlier precedes the echo it causes, since the speaker-to-mic path is longer than one buffer. AEC3's delay estimator covers the rest of the device latency.
- **Framing:**
  - AEC3 takes 10 ms frames (480 samples at 48 kHz). Voicemeeter buffers (for example 256 or 512) are re-framed through FIFOs, so the mic is delayed by at most one frame.
  - At startup the output is the delayed input, never silence.
- **Sample rate:**
  - **Supported:** 48 kHz (also 32 and 16 kHz).
  - **Unsupported (for example 44.1 kHz):** the insert passes audio through, and the status says "Echo cancellation needs 48 kHz".
  - **Rate change:** command 3 (change) resets the engine.

## 2. Native pieces
### Vendored engine (`third_party/`)
- **`webrtc-audio-processing` v2.1:** freedesktop's standalone WebRTC audio processing, tag commit `846fe90`, the `webrtc/` tree and licences only.
- **`abseil-cpp` 20240722.0:** the subset it uses (algorithm, base, functional, memory, meta, numeric, strings, types, utility), without tests or build files.
- **Tracking:** `third_party/README.md` records sources, versions, hashes and licences.

### `scripts/build-aec.ps1`
- **Sources:** compiles an explicit list with `cl /std:c++20 /O2 /MT`, with the defines from the project's meson files (`WEBRTC_WIN`, `NOMINMAX`, `_USE_MATH_DEFINES`, `WEBRTC_LIBRARY_IMPL`, `WEBRTC_ENABLE_AVX2`, `WAP_DISABLE_INLINE_SSE` off). AVX2 files get `/arch:AVX2`, the same split as meson.
- **Output:** objects cache under `.local/aec/`, and the result links into `bin/snoofer-aec.dll` with the shim (§2).
- **Integration:** `scripts/build.ps1` calls it and lock-checks the DLL, like the media and soundboard companions.

### Shim (`internal/aec/native/aec.cpp`)
A C ABI around `webrtc::AudioProcessing`, configured for AEC3, high-pass on, with no AGC and no noise suppression beyond the echo suppressor.

| Export | Purpose |
|---|---|
| `AECCreate(AEC**)` / `AECDestroy` | Lifetime, from the control thread. |
| `AECConfigure(AEC*, const AECConfig*)` | Mic channel indexes (1–2) in the input insert, reference channel indexes (1–2) in the output insert, strength (0 strong, 1 balanced, 2 gentle) and bypass. Fields are stored atomically and a generation counter is bumped; the audio thread rebuilds the engine when it sees a new generation or sample rate. |
| `AECInputInsert(void* ctx, AudioBuffer*)` / `AECOutputInsert(void* ctx, AudioBuffer*)` | Called from the monitor's callback on Voicemeeter's audio thread. No locks, no allocation after warm-up, no logging. |
| `AECReadStats(AEC*, AECStats*)` | Active, sample rate, ERLE (centi-dB), delay (ms), frames processed and whether an engine error forced pass-through, read with atomics. |

**Strengths:** they map to AEC3 config presets.
- **Strong:** the default AEC3 suppressor, the most aggressive.
- **Balanced:** a milder suppressor.
- **Gentle:** linear cancellation emphasis, least suppression.

**Exceptions:** the shim catches everything at the C boundary. On any failure it switches to pass-through and reports the error in its stats.

### Monitor DLL
- **Hook:** `SnooferSetInsert(onInput, onOutput, ctx)` stores the stages (both or neither) and is refused while registered. `SnooferStart(remote)` registers output (2), or input and output (3) when stages are set.
- **Callback:** command 10 calls `onInput` (or passes through), and command 11 calls `onOutput` and then the existing pass-through and counters.
- **Changing modes:** stop, unregister, register again and start. A failed restart is reported like today's uncertain cleanup.

## 3. Go side
- **`internal/voicemeeter`:**
  - **Interface:** `SetMonitoring(enable bool)` becomes `SetCallback(monitor bool, insert *InsertHook)`. It registers when either is wanted, and re-registers when the hook changes.
  - **Ownership:** registration stays on the adapter's owning worker, unchanged. The worker reads the wanted hook from `Dependencies.Insert` each step while live; preview and leaving live remove it.
  - **Hook life:** it lives until the callback is stopped and unregistered; only then may the AEC DLL be released.
- **`internal/aec`:** the Go wrapper for `snoofer-aec.dll`, on the AEC plugin's goroutine. Its `Hook()` returns the function pointers and context.

## 4. Plugins
- **`plugins/audio`:**
  - **`EchoTargets()`:** reports the managed mic strip's input-insert channels, the output-insert channels of the bus feeding the current playback destination, and whether the mic stack is on.
  - **Channels (Potato):** input inserts carry 2 channels per physical strip, then 8 per virtual strip; output inserts carry 8 per bus. The mic uses both channels of `Voice.Strip`, and the reference uses the first two of the A bus in `PlaybackTarget`. Otherwise a short reason is given: Not live, Needs Potato, Mic off, No mic or No playback.
  - **`SetEchoInsert(ctx, *voicemeeter.InsertHook)`:** installs the hook through the worker (asynchronously). Each request has a generation; the worker publishes the hook it registered and the generation it applied (`State.Insert`, `State.InsertGeneration`). Removal returns only once a state at the removal's generation or later shows no hook, or once the audio plugin stopped cleanly; otherwise it fails and the caller must keep the engine and DLL loaded.
  - **Rules:** only the managed mic strip is touched. A disabled mic stack leaves the engine idle (pass-through), and recorder and routing behaviour are unchanged.
- **`plugins/aec`** (requires `audio`; settings `mode` and `strength`):
  - **Mode:**
    - **Auto:** active while the playback destination is a speaker device (matched by `speaker_outputs`, Go regexps, empty means any non-headphone output).
    - **On:** always active.
    - **Off:** the hook is removed and input insert unregistered.
  - **Refresh:** every second it takes `EchoTargets`, calls `AECConfigure` and reads `AECStats`.
  - **Controls:**
    - `aec.mode` (selection; Auto, On, Off; a deck key cycles it);
    - `aec.strength` (selection);
    - `aec.status`: Active with a status line such as "−32 dB echo · 54 ms"; Idle with the reason (Headphones, Mic off, No mic, No playback, Not live, Needs Potato, Needs 48 kHz); Wait until the engine reports processing; Off; or Error (engine failure with the mic passing through, an unloadable engine or an unconfirmed hook removal).
    - No separate meter: deck meters are dBFS levels, and echo removed is not one.
  - **Engine life:** the DLL loads on first activation. Deactivating bypasses the engine at once and then removes the hook. Stop frees the engine only after removal is confirmed; otherwise it stays loaded and Stop reports the error.
- **GUI:** an Echo cancellation card on the Audio screen with mode, strength, live ERLE and delay, and the reason when idle (headphones, mic stack off, 44.1 kHz).

## 5. Safety
- **Audio-thread rules:** the input insert only runs while AEC is on. The callback never blocks, allocates or takes locks after warm-up.
- **Errors:** anything unexpected switches the engine to pass-through.
- **Shutdown:** stop and unregister the callback, then destroy the engine, then release the DLL. If cleanup is uncertain, nothing is unloaded, as with today's monitor.
- **Live use:**
  - New native code is tested offline first: a WAV-driven test harness feeds the shim synthetic echo (§6).
  - Live registration happens only through the normal plugin lifecycle.

## 6. Verification
- **Offline:** a harness (`tools/aec-probe`) plays speech-like noise through a synthetic room (delay plus decay) into the mic channel alongside the clean reference. It checks the shim reaches at least 20 dB ERLE after convergence, keeps near-end speech within 6 dB, passes other channels bit-identically, re-frames 256 and 512 buffers, and passes 44.1 kHz through.
- **Go tests:** plugin logic (Auto mode, targets, controls) uses a fake engine.
- **Hardware acceptance:** speakers with a call or recording. The far side hears no echo, local speech stays clear, and turning it off restores the original path.
