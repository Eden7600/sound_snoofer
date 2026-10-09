# Tasks
## Implementation
- [x] 1. `docs(deck)`: specify the rows layout, area colours and touch strip; update the UI contract.
- [x] 2. `feat(streamdeck)`: embed Archivo with its licence, and add fitted text rendering with a bitmap fallback; tests.
- [x] 3. `feat(streamdeck)`: row key layout, area colours, state treatments, artwork band and corner inset; tests; a native-size preview at `docs/design/deck-rows-left.png`.
- [x] 4. `feat(streamdeck)`: touch strip panels in the row style; tests; preview.
- [x] 5. Validate: gofmt, `go test ./...`, `go vet ./...`, OpenSpec strict validation, canonical build; restart Snoofer. Race detector skipped: CGO is disabled on this toolchain.

## Acceptance
- [ ] 6. On the Stream Deck + XL: no text or icon is hidden by the rounded keycaps (and the 10 px radius assumption holds); long labels read without truncation; area colours are distinct, including magenta from red; the touch strip reads at a glance.
