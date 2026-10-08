# Tasks
## Implementation
- [x] 1. `feat(config)`: pattern generation and list edit operations; tests (escaping, device form, list bounds, strict revalidation).
- [x] 2. `feat(audio)`: priority view (inventory, matches, suggestions) via shared routing match helpers; `audio.priority-edit` with save-before-apply and live reload; tests including stale save and reload failure.
- [x] 3. `feat(gui)`: Routing screen with the four list cards and suggestions; GUI check; UI contract.
- [x] 4. Validate: gofmt, `go test ./...`, `go vet ./...`, GUI check, OpenSpec strict validation, canonical build and renderer inspection.

## Acceptance
- [ ] 5. On the live setup, add a suggested device, reorder playback and see routing follow without restarting; a bad regex is rejected with routing intact.
