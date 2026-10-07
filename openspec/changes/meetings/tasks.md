# Tasks
Each numbered block is one reviewable commit.

## Implementation
- [x] 1. `docs(meetings)`: specify the Meetings screen, the Insta360 and Discord plugins and the Mic mute link (this change).
- [x] 2. `feat(camera)`: `snoofer-camera.dll` (DirectShow enumeration, extension-unit node discovery, sized KS property get/set), `scripts/build-camera.ps1` wired into `scripts/build.ps1`, and the Go binding. An opt-in live test reads status from the Link 2 without writing.
- [x] 3. `feat(insta360)`: plugin worker and controls (privacy, tracking, framing, reset, state), verification, the privacy and full-body rules, connection report and `no_insta360`. Fake-device tests cover packet parsing across firmware lengths, the rules, pending and failed verification, absence and Stop.
- [x] 4. `feat(discord)`: IPC framing, nonce matching, authorization and token refresh, state subscriptions, controls and `no_discord`. Tests against a fake pipe peer cover handshake, approval, refresh, invalid_grant, reconnect without replay, verification and the credential never appearing in reports.
- [x] 5. `feat(audio)`, `feat(discord)`: the Mic mute API and the link. Tests cover connect imposing, external adoption, acknowledgement of own writes, deafen, a single in-flight write and no loops.
- [x] 6. `feat(gui)`, `feat(streamdeck)`: Meetings screen, deck icons, UI contract and docs/plugins.md setup. Update check-gui fixtures, screenshots and overflow checks; inspect native-size icons.
- [ ] 7. Validate (gofmt, test, vet, strict OpenSpec, GUI check, canonical build), relaunch and record results.

## Hardware acceptance
- [ ] 8. Link 2: privacy, each tracking mode, framing and reset change the camera and read back; they work while another app streams.
- [ ] 9. Discord: approval once, reconnect with the saved token, the mute link both ways, deafen, camera, screen share and Leave in a real call.
