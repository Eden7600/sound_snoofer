# Tasks
## Implementation
- [x] 1. `docs(routing)`: specify microphone readiness and the complete editor.
- [x] 2. `feat(config)`: `ready` field; `ready`, `source` and activity edits; tests.
- [x] 3. `feat(audio)`: metering skips ready microphones; view gains Ready, Silent and Activity; tests.
- [ ] 4. `feat(gui)`: source select, Ready toggle, Silent badge, mono/stereo channels, Activity card, interface add without channels; GUI check; UI contract.
- [ ] 5. Validate: gofmt, `go test ./...`, `go vet ./...`, GUI check, OpenSpec strict validation, canonical build.

## Acceptance
- [ ] 6. Mark the lav ready, turn its mute switch on for longer than the activity delay: Auto stays on the lav. Set a stereo pair and switch a microphone's source in the GUI, without touching JSON.
