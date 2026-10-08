# Callback activity probe

Run from the repository root on Windows x64:

```powershell
# Compile and self-test only; does not load Voicemeeter.
./tools/callback-probe/run.ps1
# Explicit ten-second live callback capture after a successful self-test.
./tools/callback-probe/run.ps1 -Observe
```

Requires installed Visual Studio C++ tools and a Windows SDK. No compiler, SDK or DLL is downloaded by the script. Executable, objects, build command and timestamped logs stay in `.local/stall-probe`.

The output-insert callback copies samples unchanged and counts callbacks, synchronization flags and lifecycle events. No audio is saved or analyzed. This temporarily participates in the audio path; it is not a passive getter. An occupied callback slot is an error and is never taken over. No restart, routing or recorder command is issued. Ctrl+C requests normal cleanup. A 25-second watchdog exits only this probe if native calls hang; such an exit leaves cleanup unverified.

Compare captures during confirmed healthy silence, audible playback and a reproduced stall. `buffers` counts cumulative buffer callbacks, `synced` counts callbacks whose SDK synchronization flag equals one. Counts advancing do not prove physical audio output, and zero callbacks do not prove an engine stall without a healthy baseline and successful registration/start. `invalid` or `unknown` abort observation; malformed buffers cannot be safely passed through and make the run invalid. `change` is a stream-change notification, not independently a fault.

The executable supports `--self-test` without loading the native DLL. Its checks cover x64 layout, bitwise sample preservation, aliased buffers, synchronization counters, malformed input and lifecycle events. The runner compiles with warnings as errors and executes these checks before any live run.

ABI and semantics: https://github.com/vburel2018/Voicemeeter-SDK/blob/main/VoicemeeterRemote.h and https://download.vb-audio.com/Download_CABLE/VoicemeeterRemoteAPI.pdf (pages 21â€“26).

The callback implementation is shared with the production monitor in `internal/voicemeeter/callback/pass_through.h`; the same pass-through self-test runs when building the main app.

Use ./tools/callback-probe/run.ps1 -Paired for a ten-second input-plus-output pass-through observation while Snoofer is stopped. It reports input/output counts, consecutive repeated commands and unsynchronized inputs. Snapshots read counters independently; transient count differences are not atomic pairing measurements. The repeated-command totals identify startup pre-roll. This mode makes no routing changes and never saves audio.

To reproduce/verify neural background callback timing, stop Snoofer gracefully,
then run this opt-in observer from the repository root:

```powershell
$env:SNOOFER_PROBE_NEURAL = Join-Path $pwd 'bin/snoofer-neural-aec.dll'
$env:SNOOFER_PROBE_MODEL = Join-Path $pwd 'bin/models/localvqe-v1.3-4.8M-f32.gguf'
$env:SNOOFER_PROBE_MONITOR = Join-Path $pwd 'bin/snoofer-audio-monitor.dll'
./tools/callback-probe/run.ps1 -Paired
```

The deployed monitor owns registration. Neural processing observes input channels
0/1 and output channels 0–7; original input samples are restored, including aliased
buffers. No audio is saved. The probe requires Active at the end and verifies
duplicate start refusal, repeated stop, and restoration of prior process power
policy. Unset the three environment variables afterward and restore Snoofer.
Omit SNOOFER_PROBE_MONITOR to compare with the probe's direct registration.
Format counters distinguish differing input/output formats from repeated commands.
This is a callback continuity check, not an acoustic-quality test.
