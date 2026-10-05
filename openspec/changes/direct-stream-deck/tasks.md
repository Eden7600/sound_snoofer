## 1. Dependencies and protocol feasibility
- [ ] Implement mixer-surface-controls first, including fixed-bus/following-mute composition.
- [ ] Validate Windows HID enumeration, report lengths, image orientation and access on the actual + XL without Elgato software.
## 2. Implementation
- [x] Add isolated native HID adapter and product descriptor with bounded cancellable I/O.
- [x] Implement report parsing, press edges, encoder deltas and reconnect behavior.
- [x] Add built-in layout, validated configuration and shared-service actions.
- [x] Add changed-tile renderer and authoritative gain/mute/recording feedback.
- [x] Add media adapter and tray lifecycle integration.
## 3. Automated verification
- [ ] Test report fixtures, malformed input, held-key reconnect, queue overflow and shutdown.
- [ ] Test stale mic targets, simultaneous TUI edits, mute composition, slow display writes and readback failures.
- [ ] Test transport exactly-once dispatch, guard failures and dry-run suppression of all external effects.
- [ ] Run tests, vet, Windows build, strict validation and isolated preview; race checks where supported.
## 4. Hardware acceptance
- [ ] Test all six encoders, key coordinates, image orientation and hotplug on Stream Deck + XL.
- [ ] Verify A1/A2/mic gain and mute, including externally changed native settings and playback movement.
- [ ] Verify recorder/source/stage/rehearsal controls, media controls and operation with TUI closed and stock Elgato software absent.
