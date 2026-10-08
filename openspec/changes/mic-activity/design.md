# Design
## Configuration
```json
"profiles": {"microphones": ["lav", "desk", "webcam"], "activity": {"check": 2, "silence_db": -70, "silent_after_s": 10}}
```
- **`check`:** 1–4. The number of leading available options from `microphones` that stay wired and metered.
- **`silence_db`:** between −120 and −20 dBFS. Default −70.
- **`silent_after_s`:** 2–600. Default 10.

## Measurement
`voicemeeter.Client.InputLevels(strips []int)` reads `VBVMR_GetLevel` type 0 (pre-fader input) for the two channels of each physical strip and returns the peak per strip. Missing or invalid readings are omitted.

The worker samples wired mic strips every 100 ms while activity is configured, the mic stack is active and Voicemeeter is connected:
- desk → strip 0
- lav → strip 1
- webcam → strip 2

Readings are linear peaks, converted to dBFS.

## Latch
The worker owns a latch per source, with three states: unknown, active and silent.
- **Silent:** below the threshold continuously for `silent_after_s`.
- **Active:** above the threshold for 300 ms.
- **Unknown:** the strip is not wired, a reading is missing, or the mic stack is disabled. An unknown source's timers reset.

Before each planning step, the worker copies the silent set into the runtime-only `Config.SilentMics`. `ProfileConfig` carries it across the `ProfileBase` restore, like `TapeListening`.

## Planning
`ProfileConfig` (Normal, source Auto or the configured source missing):
1. Walk `microphones`, skipping options in `SilentMics`.
2. If nothing remains, walk the priority again without skipping.

`WiredMics` comes from the resolved Normal priority (Normal profile only): the first `check` entries that are current options, plus the effective source. It drives:
- **ASIO patches:** desk's pair (`Patch.asio[0..1]`) and lav's pair (`Patch.asio[2..3]`) are their channels when wired, and 0 otherwise.
- **Webcam:** `input:3` is assigned when the webcam is wired, and cleared otherwise.

The VR profile, explicit choices, mic stack Off or disabled, and configurations without activity all keep today's wiring.

## Flapping
- Silence needs `silent_after_s` of continuous silence.
- Recovery needs 300 ms of signal.
- The latch changes only on these edges, so a send-only plan change cannot flap faster than speech.
- Device changes still use the configured debounce.
