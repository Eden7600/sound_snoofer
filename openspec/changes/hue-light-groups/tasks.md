# Tasks
## Implementation
- [x] 1. `docs(hue)`: specify staged Decor, Mixed and Saturation dials; update the UI contract and plugin docs.
- [ ] 2. `feat(hue)`: model reads light function, colour, colour mode and gradients; room light groups and saturation math; tests.
- [ ] 3. `feat(hue)`: staged dials with press-to-apply, paced per-light writes, confirmation and expiry; tests.
- [ ] 4. `feat(streamdeck)`: saturation icon and default Lights dials; tests; preview.
- [ ] 5. Validate: gofmt, `go test ./...`, `go vet ./...`, GUI check, OpenSpec strict validation, canonical build.
- [ ] 6. Personal layout: Lights dials become Playback, Saturation, Decor, Mixed, Brightness (with a backup); restart Snoofer.

## Acceptance
- [ ] 7. On the deck:
  - Decor and Mixed stage, apply on press, and toggle without a stage.
  - Saturation changes colour lights without shifting their hue, and leaves white lights alone.
  - Applying in a full room is not throttled into failure.
