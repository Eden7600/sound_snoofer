# Recording playback and Home tidy-up
## Why
- **Playback:** you can record from the deck but cannot listen back without opening Voicemeeter.
- **Home layout:** recording keys are scattered across row 2 with gaps.
- **Echo cancellation:** it has no keys on Home.
- **Mic meter:** it shows the raw mic even when Element processes it, so it does not reflect what callers hear.

## What Changes
- **Recording playback (tape):** Play/Pause, Stop, Rewind and Fast-forward control whatever file Voicemeeter's recorder has loaded.
  - **Routing:** while the tape plays outside VST rehearsal, Snoofer routes it to the current Playback destination (the resolved A bus) and clears that routing when the tape stops.
  - **Surfaces:** controls appear on the deck and in the GUI's Recording card.
- **Recorder protection:** Stop, Rew and FF never stop or disturb a recording, and Play is refused while recording.
- **Home layout:**
  - **Row 1:** mic path, then Echo and Echo strength.
  - **Row 2:** recording, left to right: Record mic, Record PC, Mic stage, Record, Play, Stop, Rew, FF.
  - **Rows 3–4:** unchanged.
- **Mic meter:** while Element is the effective processing mode, the mic meter (deck dial and GUI strip) reads the AUX return strip, which is the processed mic. Mic gain still adjusts the source strip.

## Impact
- **Code:**
  - `internal/routing` (tape send ownership);
  - `internal/controller` (tape transport);
  - `internal/voicemeeter` (recorder writes, AUX level);
  - `internal/control` (actions);
  - `plugins/audio` (controls, meter);
  - `internal/streamdeck` (icons);
  - `plugins/streamdeck/default.go`;
  - `app/web`;
  - `docs/ui-contract.md`.
- **Invariants:** CLAUDE.md keeps recorder capture arming and tape playback sends separate. Tape A sends (A1–A5) become Snoofer-owned once tape playback is used, as they already are after VST rehearsal: they follow the Playback destination while the tape plays, and are 0 otherwise. B1/B3 stay 0. The file format, output directory and other recorder settings are untouched. Start/Stop recording semantics are unchanged.
