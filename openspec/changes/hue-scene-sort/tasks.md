# Tasks
## Implementation
- [x] 1. `docs(hue)`: specify sortable scenes; update the UI contract and plugin docs.
- [x] 2. `feat(hue)`: scene sort setting and control, hue and brightness sort keys, scene order on controls; tests.
- [x] 3. `feat(deck)`: sort icon, default Lights row, GUI Sort control and card order; tests; GUI check.
- [ ] 4. Validate: gofmt, `go test ./...`, `go vet ./...`, GUI check, OpenSpec strict validation, canonical build.
- [ ] 5. Personal layout: Sort at Lights r1c4 with Sync, Mode and Intensity at c5–c7 (with a backup); restart Snoofer.

## Acceptance
- [ ] 6. Pressing Sort on the deck cycles Name, Hue and Brightness, reordering the scene keys and the GUI's cards, and the choice survives a restart.
