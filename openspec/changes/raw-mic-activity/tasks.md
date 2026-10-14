# Tasks
## Implementation
- [x] 1. `docs(audio)`: specify pre-insert activity metering.
- [x] 2. `feat(audio)`: pre-insert peak capture and `SnooferReadInputs` in the audio monitor; probe self-test; Go binding; `InputLevels` prefers fresh pre-insert peaks; tests.
- [x] 3. Validate: gofmt, `go test ./...`, `go vet ./...`, callback probe self-test, OpenSpec strict validation, canonical build.

## Acceptance
- [ ] 4. With activity on and echo cancellation processing the desk mic, stay quiet for longer than the silence delay: the desk mic is not Silent and Auto keeps it. Unplug the lav (or switch it off) while it is in use: Auto moves down after the delay.
