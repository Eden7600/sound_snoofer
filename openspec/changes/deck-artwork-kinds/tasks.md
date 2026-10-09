# Tasks
## Implementation
- [x] 1. `docs(deck)`: specify artwork treatment by content type; update the UI contract.
- [x] 2. `feat(streamdeck)`: clip art shown whole, program icons in the icon slot and icon-sized on the dial, pies chosen by type; tests; previews.
- [x] 3. Validate: gofmt, `go test ./...`, `go vet ./...`, OpenSpec strict validation, canonical build; restart Snoofer. Race detector skipped: CGO is disabled on this toolchain.

- [x] 5. `feat(streamdeck)`: program icons above a one-line scrolling title; tests; preview.
- [ ] 6. `feat(artwork)`: 128 px artwork contract; scenes, Windows covers and extension covers at 128 px; tests; extension build.
- [ ] 7. Validate again; restart Snoofer.

## Acceptance
- [ ] 4. On the Stream Deck + XL: clip art is shown whole; app keys show crisp program icons above a one-line title; scene wedges and covers are sharp; covers and scenes look as before.
