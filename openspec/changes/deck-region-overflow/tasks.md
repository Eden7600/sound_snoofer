# Tasks
## Implementation
- [x] 1. `docs(deck)`: specify region overflow editing and the editor audit fixes; UI contract.
- [x] 2. `feat(streamdeck)`: `region-clip` edit, clipped stacks, `Clip`/`Stacked`/`Sets`/`Scrolls` in the editor view, Make Home only on change; tests.
- [ ] 3. `feat(gui)`: Overflow checkbox, shared-row sources, sets note, Make Home disabled on Home; GUI check.
- [ ] 4. Validate: gofmt, `go test ./...`, `go vet ./...`, GUI model and browser checks, OpenSpec strict validation, canonical build.
- [ ] 5. Personal layout: set `clip: true` on both Home regions in `bin/snoofer.json` while Snoofer is stopped; restart Snoofer.

## Acceptance
- [ ] 6. On the Stream Deck + XL, the page dial shows one Home set. In the editor, Overflow is off on both Home regions, and turning it on and off changes the sets note.
