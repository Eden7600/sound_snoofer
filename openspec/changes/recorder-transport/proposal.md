# Recorder transport
## Why
The user mapped the Voicemeeter recorder's commands by state:

| State | Command | Effect |
|---|---|---|
| Recording | `Recorder.record = 1` | Recording active, ongoing |
| Recording | `Recorder.stop = 1` | Recording ended; position returns to 0, ready for a new recording |
| Recording | `Recorder.pause = 1` | Recording paused. Does not work in playback mode |
| Playing back | `Recorder.play = 1` | Plays the loaded recording |
| Playing back | `Recorder.stop = 1` | Pauses playback; a second stop stops it fully and returns to 0 |
| Playing back | `Recorder.rew = 1`, `Recorder.ff = 1` | Rewind and fast-forward; not applicable while recording |

Today's transport disagrees with this:
- **Playback pause** sends `Recorder.pause`, which does not work in playback.
- **Stop cannot return to the start.** After the first stop the recorder reads as stopped, so Stop becomes unavailable, and the second stop can never be sent.
- **Recordings cannot be paused** from the deck or the GUI.

## What Changes
Tape-deck transport: the existing keys act by recorder state.

| State | Record | Play/Pause | Stop | Rew / FF |
|---|---|---|---|---|
| Stopped | Start | Play | — | ✓ |
| Playing | Start (stops playback first) | Pause (one stop) | To start (two stops) | ✓ |
| Tape paused (by Snoofer's stop) | Start | Resume (play) | To start (one stop) | ✓ |
| Recording | Stop Rec | Pause recording | End recording | N/A |
| Recording paused | Stop Rec | Resume recording (`Recorder.record`) | End recording | N/A |

- **Tape paused:** Snoofer remembers that its own stop paused playback. The recorder reports the same values for "stopped mid-file" and "stopped at 0", so this is Snoofer's knowledge, not an observation. It is cleared by the second stop, Play, Record, or any observed state other than Stopped.

## Impact
- **Code:**
  - `internal/controller` (tape and record commands, tape-paused state);
  - `internal/control` (publishes tape paused);
  - `plugins/audio` (actions and key states);
  - `app/web/app.js` (the Recording card's Play/Pause label follows the key's icon);
  - `docs/ui-contract.md`.
- **Invariants:**
  - Start and Stop stay explicit one-shot commands, never retried. Each write is verified where a readback exists.
  - Record never starts on launch or reconnect.
  - Starting a recording during playback first stops playback, as part of the same user-requested Start.
  - No routing or format settings change.
