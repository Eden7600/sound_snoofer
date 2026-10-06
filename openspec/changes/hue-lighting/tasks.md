# Tasks
Each numbered block is one reviewable commit.

## Implementation
- [x] 1. Specify Hue lighting (this change).
- [x] 2. `feat(hue)`: bridge client: mDNS discovery, `/api/0/config` identity, link-button pairing, certificate pinning, CLIP v2 resource model and SSE parsing. Tests against an `httptest` TLS fake bridge cover none/one/many bridges, ID mismatch, pin mismatch, error 101 and SSE gaps.
- [x] 3. `feat(hue)`: plugin worker and controls: status, pair, group selection, scene recall with confirmation, brightness/temperature knob math, coalesced group writes, backoff, verification timeout and preview gating. Composition file with `no_hue`. Tests cover knob clamping and off-state rules, Mixed/N/A temperature, coalescing under fast ticks, 429 backoff, stale revisions, reconnect without replay, and Stop joining the worker.
- [x] 4. `feat(huesync)`: WebSocket client and plugin controls with `no_huesync`. Tests against a fake loopback server cover state events, refused connection, restart, bridge_disconnected, tick accumulation, confirmation timeout and preview.
- [x] 5. `feat(streamdeck)`: Hue icons, UI contract vocabulary and focused presentation tests; inspect native-size renderer output.
- [x] 6. `docs(hue)`: plugins.md setup for pairing, room selection, `auto_controls` scene pages and Hue Sync third-party control.
- [x] 7. Run gofmt, `go test ./...`, `go vet ./...`, `go test -race ./...` (if supported), strict OpenSpec validation and `scripts/build.ps1`. Record results in validation.md.

## Revision 2 (after first use)
- [x] 12. `docs(hue)`: respecify: merged plugin, multi-interface discovery, Lights screen, enable-only Plugins page, scene slots, blank slots, personal Home layout, and the contract correction for Home dial index 2.
- [x] 13. `fix(hue)`: query mDNS on every multicast IPv4 interface. Unit-test the interface filter and report the found bridge while unpaired.
- [x] 14. `refactor(hue)!`: merge Hue Sync into the hue plugin with `sync_port` and `hue.sync*` controls; remove `plugins/huesync` and `no_huesync`. Tests cover the brightness dial while syncing, scene-stops-sync with confirmation and timeout, temperature while syncing, and the existing sync scenarios.
- [x] 15. `feat(hue)`: room scene slots, with tests for room change, blanks and stale slot presses.
- [x] 16. `feat(streamdeck)`: blank empty-slot keys, with presentation tests; correct the contract wording for the Home dials.
- [x] 17. `feat(gui)`: Lights screen and enable-only Plugins page. Update check-gui fixtures and screenshots, the UI contract and docs/plugins.md.
- [x] 18. Validate (gofmt, test, vet, OpenSpec, GUI check, canonical build). Apply the personal Home layout and remove the stale `huesync` entry while Snoofer is stopped, then relaunch. Record the results.

- [x] 19. `fix(hue)`: tolerate type-specific `status` shapes and skip malformed unused resource types in loads and events. Test with real-bridge payload shapes, and verify with a gated live load against the paired bridge.

## Revision 3
- [x] 20. `feat(hue)!`: remove color-temperature control (plugin, settings, model, GUI, icon, docs, contract, tests); remove `neutral_kelvin` and the Home dial 4 binding from the personal config while Snoofer is stopped; rebuild and relaunch.

## Hardware acceptance
- [ ] 8. Pair the real bridge through the link button (revision 1 discovery failed with "No bridge" on this multi-adapter host); confirm discovery, restart persistence and that the key stays out of diagnostics.
- [ ] 9. Recall scenes from the Home slots; confirm Active state and changes made in the Hue app.
- [ ] 10. Turn and press the brightness dial on the configured room; judge responsiveness of the 250 ms coalescing and confirm no lag backlog.
- [ ] 11. Enable Hue Sync Third-party control; toggle sync, change mode/intensity, check the joined brightness dial and that scenes stop sync; close and reopen Hue Sync.
