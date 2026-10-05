## 1. Native feasibility
- [ ] Confirm gain ranges and native mute semantics for strips, buses, pre/post recording and Element return.
- [x] Define durable mute ownership/baseline recovery and migration before implementing writes.
## 2. Implementation
- [x] Extract shared action/state boundary with per-client acknowledgment and target generations.
- [x] Add validated gain/readback and ordered relative adjustment.
- [x] Add persisted independent mic/playback mute and safe ownership transitions.
- [x] Add Controls/Graph mute status and hardware-ready gain state.
## 3. Automated verification
- [ ] Cover concurrent clients, full queues, cancellation, manual gain drift, clamping, invalid floats and stale targets.
- [ ] Cover no-device-writes on mute, independent computer capture, manual mute preservation, restart recovery and failed mute-before-route.
- [ ] Run tests, vet, Windows build, strict validation and isolated TUI smoke; run race checks where supported.
## 4. Hardware acceptance
- [ ] Verify direct and Element mute, tails, monitoring, recording, rehearsal and source changes without mute-triggered reassignment.
- [ ] Verify A1/A2/mic gain readback, manual changes and responsive controls.
