# Tasks
## Implementation
- [x] 1. `docs(audit)`: specify the audit reliability fixes.
- [x] 2. `fix(deck)`: publish empty key/dial lists; test the marshalled view for a model without dials.
- [x] 3. `fix(audio)`: check prefixed Normal/VR device rows for availability; tests for `normal-output` and a VR row.
- [x] 4. `fix(audio)`: soundboard readiness compares planned `Strip[7]` operations; test with an output slot carrying the soundboard.
- [x] 5. `fix(voicemeeter)`: retry a failed callback registration; test.
- [x] 6. `fix(deck)`: deliver reads that complete at the deadline (native; verified on hardware in task 16).
- [x] 7. `fix(nowplaying)`: skip malformed bridge messages; coerce report fields; fetch only http(s) artwork without credentials; tests.
- [x] 8. `fix(routing)`: gate every interface-mic strip; test with a patch index of 4 or higher.
- [x] 9. `fix(audio)`: save tape ownership only before tape playback starts; test pausing a recording with saved choices unwritable.
- [x] 10. `fix(audio)`: refuse audio settings edits whose saved choices would not load; test.
- [x] 11. `fix(gui)`: reschedule polling after render errors; guard missing deck slots (verified by the GUI check; a render-error regression test belongs to `gui-test-harness`).
- [x] 12. `fix(hue)`: reset backoff on the first stream message; cancel a client's stream on close; tests.
- [x] 13. `fix(discord)`: run one token exchange at a time; test.
- [x] 14. `fix(nowplaying)`: stop after the listener closes; retry a busy port; test.
- [x] 15. Validate: gofmt, `go test ./...`, `go vet ./...`, `node --test`, GUI check, OpenSpec strict validation, canonical build. (`go test -race` needs cgo on this toolchain and was not run.)

## Acceptance
- [ ] 16. On hardware: edit a Stream Deck Mini or MK.2 page; start Snoofer before Voicemeeter and confirm the callback registers once Voicemeeter runs; pause a recording from Snoofer; hold and release deck keys repeatedly with no phantom hold.

2026-10-15 partial task 16 hardware evidence: after 03e74d2, the user held the Plus XL Play key for about two seconds, released it and repeated three times; playback toggled once per press with no repeats while held or after release. Mini/MK.2 and Voicemeeter startup remain open.

Later live task 16 evidence: a real recording was paused from Snoofer's GUI; Third-Party Apps
reported Voicemeeter Recorder Paused, Stop Rec remained visible, and Rewind/Fast-forward were
disabled. Resume returned to recording and Stop ended the take. This completes the recording
Pause component; task 16 remains open for Mini/MK.2 and Voicemeeter startup acceptance.
