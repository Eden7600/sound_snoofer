# Pause routing writes
## Why
Testing Voicemeeter setups outside normal operation is hard while Snoofer keeps correcting them: it reassigns devices and resets sends within a second. Two switches let the user stop each kind of correction independently, without disabling the audio plugin or going to preview.

## What Changes
- **Disable device manipulation:** Snoofer stops writing device assignments (hardware outputs A1–A5 and input slots).
- **Disable send manipulation:** Snoofer stops writing every routing parameter:
  - strip and bus sends, including soundboard sends;
  - ASIO input patches;
  - mutes from the routing plan;
  - recorder routing and tape-playback protection.
- **Unaffected:** the user's explicit actions still write:
  - gains;
  - Playback and Mic mute;
  - recorder Start/Stop;
  - snippet play.
  
  Planning, observation, drift checks and status keep running.
- **Persistence:** both switches are saved audio intent, like Automatic recovery, so they survive restarts until turned off.
- **GUI:** two toggles in Audio › Routing & recovery. Each shows how many changes it is holding back. A header badge (Devices paused, Sends paused) shows while either is on.

## Impact
- **Code:**
  - `internal/config` (intent fields);
  - `internal/routing` (held operations);
  - `internal/controller` (recorder protection and preparation);
  - `internal/control` (edits);
  - `plugins/audio` (toggles);
  - `app/web` (header badge);
  - `docs/ui-contract.md`.
- **Invariants:** while a switch is on, its kind of write no longer meets the CLAUDE.md routing and recorder invariants. This is the intended test mode: the plan still shows what normal operation would do, and turning the switch off resumes it.
