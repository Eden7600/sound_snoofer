# Design
## Reset
`Work` already receives `Dependencies.Load`, which is what reload uses. The audio plugin's `Load` returns the startup inline configuration with saved choices, and the CLI's returns `LoadEffective(path)`. Reset uses `deps.Load(path)` and then clears `Intent`, so the CLI keeps its behavior and the plugin stops reading `state_path` as a configuration file.

## Mic priority default
`config.DefaultMicrophonePriority` holds `["lav","webcam"]`. `default.json` writes it out. The audio plugin uses the named default only when an older config omits `profiles`, and the fallback is documented at that point.

## Process names
`voicemeeter.Client` detects the processor by a process name supplied by the audio plugin from `studio.voice.processor_process`. A name is a bare executable name (no path separators). It is matched case-insensitively, as today. The VR plugin reads `process` from its settings the same way. Validation rejects empty-after-trim values that are present.

## Step sizes
Each plugin validates its step:
- `gain_step_db` must be in (0, 6];
- `volume_step` must be in (0, 0.25];
- `brightness_step` must be an integer in [1, 25].

Absent means the current value. Clamps such as the −60..+12 dB gain range remain hardware facts in code.
