# Tasks
## Implementation
- [x] 1. `docs(streamdeck)`: specify the dark deck.
- [x] 2. `feat(presence)`: session lock and display state watcher with a stub; message decoding tests.
- [x] 3. `feat(streamdeck)`: dark frames, brightness feature report, plugin dark state with dropped input, `brightness` setting, report detail; tests; UI contract.
- [x] 4. Validate: gofmt, `go test ./...`, `go vet ./...`, OpenSpec strict validation, canonical build; read-only watcher probe on the live session.

## Acceptance
- [ ] 5. Lock Windows: the deck goes black (backlight off) and presses do nothing; unlock restores it. Let the monitors sleep while unlocked: same. Deck brightness after unlock matches the setting.
