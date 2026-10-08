# Tasks
## Implementation
- [x] 1. `feat(config)`: outputs schema, validation, saved route choices; tests.
- [x] 2. `feat(routing)`: slot bus allocation, playback exclusion, slot sends, tape buses, output status; planner and controller tests for the scenarios.
- [x] 3. `feat(audio)`: slot status and route controls, route rule edits, `audio.output-edit`; tests.
- [x] 4. `feat(gui)`: Outputs matrix on the Routing screen; GUI check; UI contract.
- [ ] 5. `feat(streamdeck)`: default Routing page and Home go-to key; presentation check.
- [ ] 6. Validate: gofmt, `go test ./...`, `go vet ./...`, GUI and desktop checks, OpenSpec strict validation, canonical build and renderer inspection.

## Acceptance
- [ ] 7. With speakers in a Music slot and headphones as Playback, music plays on the speakers and monitoring on the headphones with no writes from Snoofer fighting them; tape plays to both when enabled.
