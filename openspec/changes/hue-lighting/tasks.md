# Tasks
Each numbered block is one reviewable commit.

## Implementation
- [x] 1. Specify Hue lighting (this change).
- [x] 2. `feat(hue)`: bridge client: mDNS discovery, `/api/0/config` identity, link-button pairing, certificate pinning, CLIP v2 resource model and SSE parsing. Tests against an `httptest` TLS fake bridge cover none/one/many bridges, ID mismatch, pin mismatch, error 101 and SSE gaps.
- [x] 3. `feat(hue)`: plugin worker and controls: status, pair, group selection, scene recall with confirmation, brightness/temperature knob math, coalesced group writes, backoff, verification timeout and preview gating. Composition file with `no_hue`. Tests cover knob clamping and off-state rules, Mixed/N/A temperature, coalescing under fast ticks, 429 backoff, stale revisions, reconnect without replay, and Stop joining the worker.
- [x] 4. `feat(huesync)`: WebSocket client and plugin controls with `no_huesync`. Tests against a fake loopback server cover state events, refused connection, restart, bridge_disconnected, tick accumulation, confirmation timeout and preview.
- [x] 5. `feat(streamdeck)`: Hue icons, UI contract vocabulary and focused presentation tests; inspect native-size renderer output.
- [x] 6. `docs(hue)`: plugins.md setup for pairing, room selection, `auto_controls` scene pages and Hue Sync third-party control.
- [ ] 7. Run gofmt, `go test ./...`, `go vet ./...`, `go test -race ./...` (if supported), strict OpenSpec validation and `scripts/build.ps1`. Record results in validation.md.

## Hardware acceptance
- [ ] 8. Pair the real bridge through the link button; confirm discovery, restart persistence and that the key stays out of diagnostics.
- [ ] 9. Recall scenes from an automatic deck page; confirm Active state and changes made in the Hue app.
- [ ] 10. Turn and press the brightness and temperature dials on the configured room; judge responsiveness of the 250 ms coalescing and confirm no lag backlog.
- [ ] 11. Enable Hue Sync Third-party control; toggle sync, dial brightness and change mode/intensity; close and reopen Hue Sync.
