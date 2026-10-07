# Design
## Tape transport
**States:** "listening playback" means the recorder reports Playing, or Paused without `Recorder.record`, while Recording to VST is off and `Recorder.B2` is 0.

| Control | Action | Allowed when |
|---|---|---|
| `audio.tape-play` | Stopped or paused: route, then `Recorder.play`. Playing: `Recorder.pause`. | Not recording, not Recording to VST (use the rehearsal snippet keys) |
| `audio.tape-stop` | `Recorder.stop` | Listening playback (never stops a recording) |
| `audio.tape-rew` / `audio.tape-ff` | `Recorder.rew` / `Recorder.ff` | Not recording |

- **Commands:** each control is an explicit one-shot command, never retried.
- **Verification:** Play and Pause verify the reported state within the verify window. Stop verifies Stopped. Rew and FF have no readback (the recorder exposes no position), so they report as submitted.
- **Routing at Play:**
  - Play first marks `Recording.TapeRoutingManaged` in the saved intent (save before apply).
  - It then writes `Recorder.A<n>` = 1 for the Playback destination's bus and 0 for the other A buses, verifies them, and only then starts the recorder. This way the first second is not silent.
  - With no Playback destination, Play is refused ("No playback"). With sends paused, Play is refused if a send needs changing.

## Planner ownership
- **What changes:** `addRehearsal` already owns the tape sends (A1–A5, B1, B3) under ToVST or `TapeRoutingManaged`. Its desired value for the Playback destination's bus becomes 1 when either:
  - pre monitoring is in VST rehearsal (as today), or
  - listening playback that Snoofer's Play started is in progress (new). The controller keeps a runtime `TapeListening` flag: it is set by a successful Play, carried across controller resets, and cleared once the recorder stops playing. Playback started elsewhere, or still running when VST rehearsal is turned off, is not routed. After a Snoofer restart mid-playback, the tape send is cleared.
  
  Every other tape send is 0.
- **When the tape stops:** the plan clears the send on the next reconciliation.
- **When the destination changes mid-playback:** the send moves with it.

## Native writes
`SetRecorder` additionally accepts `Recorder.play`, `Recorder.pause`, `Recorder.ff` and `Recorder.rew` with value 1. Every other recorder parameter keeps today's whitelist.

## Mic meter
- **Levels:** on Potato, `GainLevels` also reads strip 6 (AUX: input level kind 2, channels 18–19) as `Strip[6].Gain`.
- **Meter source:** the mic meter uses that reading when `Voice.EffectiveMode` is element. Element is effective only while it runs; otherwise the fallback is Direct, as today. Otherwise it reads the source strip.
- **Gain:** the gain target is unchanged.

## Surfaces
- **Deck icons:**
  - `tape-play`: a play triangle; it shows pause bars while playing.
  - `tape-stop`: a square.
  - `tape-rew` and `tape-ff`: double triangles.
- **Key text and availability:**
  - Labels are Play, Stop, Rew and FF.
  - Play shows Playing (cyan, with pause bars), Paused or Ready (neutral), Rec or N/A.
  - Stop, Rew and FF are unavailable while recording, and Stop also while nothing is playing.
- **Echo keys on Home:** `aec.mode` and `aec.strength`; selection keys cycle.
- **GUI:** the Recording card gains a transport row (Record, Play/Pause, Stop, Rew, FF) beside the capture toggles. The UI contract's "transport stays off the Audio page" rule gains this exception.
- **Default Home layout** (plugins/streamdeck/default.go) and the personal layout:

| Row | Keys |
|---|---|
| 1 | Mute, Playback mute, Monitor, Processing, Echo, Echo strength (the personal layout keeps Mic stack first) |
| 2 | Record mic, Record PC, Mic stage, Record, Play, Stop, Rew, FF |
| 3, 4 | Unchanged |
