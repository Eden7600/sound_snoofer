# Tasks
## Implementation
- [ ] 1. `feat(config)`: `profiles.activity` schema and validation; tests.
- [ ] 2. `feat(routing)`: silence-aware Auto priority and the wired mic set (patches, webcam input); planner tests for each scenario.
- [ ] 3. `feat(audio)`: pre-fader input levels, worker latch and sampling, Silent option labels; latch and worker tests.
- [ ] 4. Validate: gofmt, `go test ./...`, `go vet ./...`, OpenSpec strict validation, canonical build.

## Acceptance
- [ ] 5. With activity configured on the live setup, switch the lav off: after the silence time Auto moves to desk; switch it on and speak: Auto returns to the lav. The webcam is not assigned unless checked or used.
