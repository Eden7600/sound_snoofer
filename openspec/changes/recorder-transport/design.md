# Design
## Controller
`Controller.TapePaused` is set when Snoofer's stop pauses playback. It is cleared by:
- a second stop;
- a successful Play;
- a Record start;
- `Plan()` observing a state other than Stopped.

The worker clears it the same way on each poll, and copies it across controller resets like `TapeListening`.

### `Tape(TapePlayPause)`
| State | Writes | Verified |
|---|---|---|
| Recording | `Recorder.pause` | Paused, with record still 1 |
| Recording paused | `Recorder.record` | Recording |
| Playing | `Recorder.stop` | Stopped; `TapePaused` set |
| Playback paused externally (pause with record 0) | `Recorder.play` | Playing |
| Stopped | Today's Play path (route, then `Recorder.play`) | Playing; `TapePaused` cleared |

### `Tape(TapeStop)`
| State | Action |
|---|---|
| Recording, recording paused | `Record(ctx, false, live)`: today's verified stop |
| Playing, playback paused | `Recorder.stop`, verify Stopped, then `Recorder.stop` again. There is no readback for the second write, so its outcome is reported but not verified |
| Stopped with `TapePaused` | One `Recorder.stop`; `TapePaused` cleared |
| Stopped otherwise | Refused: "already at start" |

### Other commands
- **`Tape(TapeRewind / TapeForward)`:** refused while recording or recording paused. Otherwise one write, as today.
- **`Record(start)` during playback (playing or paused):** `Recorder.stop` and verify Stopped first. Then today's preparation runs, which still requires Stopped. A failed stop aborts the Start, and is not retried.

## Keys (`plugins/audio`)
- **Record** (`record-toggle`): Start when Stopped, Playing or playback paused; Stop when Recording or recording paused.
- **Play** (`tape-play`):
  - icon `tape-pause` while Playing or Recording (pressing pauses);
  - icon `tape-play` otherwise;
  - value Ready, Playing, Paused (tape paused, or playback paused), Rec (recording), or Paused (recording paused); N/A when unknown.
- **Stop** (`tape-stop`): available while Playing, playback paused, tape paused, Recording or recording paused.
- **Rew, FF:** available unless recording, recording paused or unknown.

## GUI
The Recording card's Play button reads Pause when the key's icon is `tape-pause`.
