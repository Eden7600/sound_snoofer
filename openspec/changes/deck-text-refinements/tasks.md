# Tasks
## Implementation
- [x] 1. `docs(deck)`: specify scrolling text, two-line values, pie bands and quieter scene and clip keys; update the UI contract.
- [x] 2. `feat(streamdeck)`: pie bands fill their band; scene and clip keys omit routine words, and their labels may use two lines; tests; preview.
- [ ] 3. `feat(streamdeck)`: two-line values in place of the icon; tests; preview.
- [ ] 4. `feat(streamdeck)`: cached faces and scrolling text on keys and the media dial; the surface redraws scrolling keys on a timer; tests.
- [ ] 5. Validate: gofmt, `go test ./...`, `go vet ./...`, OpenSpec strict validation, canonical build; restart Snoofer.

## Acceptance
- [ ] 6. On the Stream Deck + XL: a long song title scrolls smoothly, and other keys do not flicker; the Room key wraps a long room name; scene bands show only wedges; scene and clip keys read clearly without status words.
