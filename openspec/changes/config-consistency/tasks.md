# Tasks
## Implementation
- [x] 1. `fix(config)`: reset uses the running configuration; explicit mic priority in `default.json` with a named legacy default; remove the unread audio `stream_deck` schema; the example test uses the embedded default. Tests for reset without a config file at the state path.
- [ ] 2. `feat(config)`: configurable processor/VR process names and dial step sizes, with defaults and validation tests.
- [ ] 3. Validate: gofmt, `go test ./...`, `go vet ./...`, OpenSpec strict validation, canonical build.

## Acceptance
- [ ] 4. Reset choices from the GUI on the live setup restores voice defaults without errors.
