# Tasks
## Implementation
- [x] 1. `docs(recording)`: specify tape-deck transport from the user's recorder command map; update the UI contract.
- [x] 2. `feat(controller)`: tape-paused state; Play/Pause, Stop, Rew/FF and Record by recorder state; tests with the fake recorder.
- [ ] 3. `feat(audio)`: publish tape paused; key actions and states by recorder state; GUI Play/Pause follows the icon; tests; GUI check.
- [ ] 4. Validate: gofmt, `go test ./...`, `go vet ./...`, GUI check, OpenSpec strict validation, canonical build; restart Snoofer.

## Acceptance
- [ ] 5. With Voicemeeter:
  - Play, Pause (one stop), resume, and Stop return the tape to the start.
  - A recording pauses, resumes and ends from the deck.
  - Rew and FF are N/A while recording.
  - Record during playback stops the tape and starts recording.
