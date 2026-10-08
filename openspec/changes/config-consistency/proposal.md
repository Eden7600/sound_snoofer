# Config consistency
## Why
An audit of how declarative the configuration is found places where what runs differs from what the config says, or where the config describes behavior nothing reads:
- **Reset reads the wrong file.** Reset choices loads `state_path` (for example `bin/config.json`) as a full configuration, while the running audio plugin uses the inline config from `snoofer.json`. Reset therefore validates against a stale copy and fails when that file is absent.
- **Hidden mic priority.** When `profiles` is omitted, the audio plugin silently uses `["lav","webcam"]`. Neither the embedded default nor any visible config states it.
- **Dead schema.** `stream_deck` inside the audio config (`internal/config/streamdeck.go`) is decoded and validated, yet the audio plugin rejects it and nothing reads it. The deck layout lives in the Stream Deck plugin.
- **Missing fixture.** `TestVoiceExample` loads the deleted root `config.voice.json`. The embedded `internal/config/default.json` is the effective factory configuration and should be what is tested.
- **Hardcoded policy.** These are policy, not ABI:
  - processor and VR process names (`element.exe`, `vrserver.exe`);
  - dial step sizes (mixer gain, app volume, Hue brightness).

## What Changes
- Reset choices derives defaults from the configuration the worker is running (its `Load` dependency), never from a separately read file.
- `default.json` states `profiles.microphones` explicitly. The single legacy fallback for configs that omit it is a named, documented config default, not an inline literal.
- `stream_deck` is removed from the audio config schema. Strict decoding rejects it as an unknown field, as it does today through the explicit check.
- The example test validates the embedded default.
- New optional fields, each defaulting to today's value:
  - `studio.voice.processor_process` (default `element.exe`);
  - VR plugin `process` (default `vrserver.exe`);
  - audio `gain_step_db` (default 1);
  - app audio `volume_step` (default 0.02);
  - Hue `brightness_step` (default 2).
- Out of scope:
  - **Factory hardware patterns** (Volt, AirPods, SteelSeries, Insta360) stay in `default.json` because they seed the personal setup. The priority-list editor (separate change) makes them editable.
  - **The signal graph** (mic strips, B2/AUX/B3 processing, B1 recording, soundboard strip) stays in code. Those are CLAUDE.md invariants, and the slot routing change addresses the user-facing routing gap.

## Impact
- **Code:** `internal/config`, `internal/control/worker.go`, `plugins/audio`, `plugins/vr`, `plugins/appaudio`, `plugins/hue`, `internal/voicemeeter` (processor process name).
- **Compatibility:** existing configs keep working unchanged; every new field is optional.
