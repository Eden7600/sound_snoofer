# Tasks
## Implementation
- [x] 1. `docs(recording)`: specify tape-deck transport from the user's recorder command map; update the UI contract.
- [x] 2. `feat(controller)`: tape-paused state; Play/Pause, Stop, Rew/FF and Record by recorder state; tests with the fake recorder.
- [x] 3. `feat(audio)`: publish tape paused; key actions and states by recorder state; GUI Play/Pause follows the icon; tests; GUI check.
- [x] 4. Validate: gofmt, `go test ./...`, `go vet ./...`, GUI check, OpenSpec strict validation, canonical build; restart Snoofer. Race detector skipped: CGO is disabled on this toolchain.

- [x] 6. `fix(audio)`: Stop, Rew and FF show Ready; option labels only on selections, so empty values never read Automatic; tests.

- [x] 7. `fix(audio)`: transport keys keep their per-state availability; keys with nothing to do are inactive (greyed, inert), and N/A only when the state is unknown; the GUI disables them; tests; GUI check.

## Acceptance
- [ ] 5. With Voicemeeter:
  - Play, Pause (one stop), resume, and Stop return the tape to the start.
  - A recording pauses, resumes and ends from the deck.
  - Rew and FF are N/A while recording.
  - Record during playback stops the tape and starts recording.

2026-10-15 partial task 5 evidence: using the real GUI Recording row, Record started a take;
Pause retained Stop Rec and showed Play, with Voicemeeter Recorder explicitly Paused and
Rewind/Fast-forward disabled. Play resumed recording and Stop ended it. A new 3,458,740-byte
WAV was observed. These are GUI transport observations; physical Deck recording transport,
playback pause/reset, Record-during-playback and audible/file-content checks are not inferred.
