# Tasks
## Implementation
- [x] 1. `docs(audit)`: specify the audit hardening.
- [x] 2. `fix(gui)`: send priority and output edits from older lists as stale; GUI check (a targeted browser regression test belongs to `gui-test-harness`).
- [x] 3. `fix(nowplaying)`: bound WinRT media calls; build the media helper.
- [x] 4. `fix(build)`: stage all selected companions/models/licenses until esbuild, Tailwind and Go succeed; preserve the sole application output `bin/snoofer.exe`; repair neural-AEC setup; test failure/success/tagged paths and interrupted dependency recovery with isolated fake commands.
- [x] 5. `fix(voicemeeter)`: treat "no callback registered" as a clean stop; callback probe.
- [x] 6. `fix(deck)`: seed the key baseline from the current input report (initial implementation; hardware evidence below supersedes this approach in task 14).
- [x] 7. `fix(deck)`: report every deck path that fails to open; retire temporary path reports on recovery/removal; surface and report-lifecycle tests.
- [x] 8. `fix(hue)`: ignore light-write results from before a room change; test.
- [x] 9. `fix(nowplaying)`: replace bridge connections only from the same origin; test.
- [x] 10. Validate: gofmt, `go test ./...`, `go vet ./...`, GUI check, OpenSpec strict validation, canonical build.
- [x] 10a. `fix(host)`: add plugin context to the host deadline branch without changing
  cleanup policy. First extend the held-instance timeout regression to fail on the
  missing ID and cover preserved context identity plus earlier cleanup errors; then
  apply the narrow fix, run focused lifecycle tests/vet and strict validation, and
  commit the diagnostic checkpoint before a coordinated live recheck.

## Acceptance
- [ ] 11. On hardware: restart Voicemeeter while Snoofer runs and confirm the monitor stays enabled; press a deck key once right after connecting it; connect two decks with one refused and see its error; freeze a media player and confirm Now Playing keeps updating.
- [x] 12. Root coordinates the canonical build and graceful shutdown recheck with the
  prior host launch configuration. Preserve the original anonymous timeout as failed
  evidence; record success or the named awaited plugin without force termination or
  claiming the two connected decks caused it. Any underlying repair needs its own
  confirmed evidence and reviewed scope.

- [x] 13. Add the explicitly opted-in native initial-input probe; run default package tests and strict validation, then coordinate the read-only hardware probe with the host stopped. Record query/decode results for Plus and Plus XL and restore the same host configuration. Do not mark first-key acceptance complete from a probe.

Probe evidence, 2026-10-15: Plus (0084) and Plus XL (00C6) both returned 512-byte input reports beginning 01 00 00 00 with zero payload. Decoder rejected them as invalid key count. The XL returned the same report with its unbound key 8 held. The real XL first Play press after reconnect did nothing; the second worked (user report). This is failed first-key acceptance, not a successful baseline. The read-only probes ran in 0.03 s; default Stream Deck tests pass. Logs: .local/deck-baseline-probe.log and .local/deck-baseline-held-probe.log.

## First-input policy correction
- [x] 14. `fix(deck)`: implement the user's released-at-connect policy for keys and dial buttons, remove the unsupported production current-report query, and keep any historical query test-local. First prove Plus and Plus XL startup key/dial-button regressions fail, then verify first down exactly once, repeated-down deduplication, release/repress, malformed-before-first-input and fresh reconnect state. Retain existing hold, stale-generation, dark/inert and multi-deck coverage; run gofmt, focused and full Stream Deck/package plugin tests, vet and strict validation.
- [x] 15. Root builds canonically, restores the prior host configuration and confirms the first displayed key and dial button work after connection/reconnection on Plus and Plus XL. Record observed behavior and preserve the failed first-key evidence above; do not complete this from automated tests.

Task 14 verification: all eight Plus/Plus XL first key/dial-button cases and the protocol first-down assertion failed before the fix. Focused regressions pass 10 times (4.191 s); full internal/streamdeck (15.919 s) and plugins/streamdeck (5.344 s) pass, including hold, stale-generation, dark/inert and actual fake-surface reconnect key/dial input. Both packages pass vet; all three affected OpenSpec changes validate strictly. Native query remains only in the opt-in historical test. Root's requested routing renderer preview was generated at .local/deck-status-acceptance.png; no physical input acceptance is inferred.

Task 15 hardware acceptance, 2026-10-15: canonical build of 03e74d2 passed and restored the previous bin/snoofer.json configuration as PID69960. With keys released during reconnect, the user confirmed all four first presses worked: XL Play key29 and dial3 toggle playback; Plus dial3 toggles playback and Media key5 changes page. Original failed acceptance remains recorded above. Full repository Go tests and vet also pass.

2026-10-15 partial task 11 hardware evidence: Voicemeeter was shut down normally from its
menu and confirmed absent while Snoofer remained running. After relaunch, Recorder reported
Stopped and Audio Callback Monitor Active; Buffers advanced 218493→219302 without a callback
error. Voicemeeter/Recorder retained the actual disconnect-gap error as history. This completes
the real Voicemeeter process-restart component. First Deck key/dial presses are already verified
in task 15; a refused second Deck and frozen media-player acceptance remain open. Startup with
Voicemeeter absent is separately pending in audit-reliability-fixes task 16.

## Second-deck refusal acceptance helper
- [ ] 16. Prepare the explicitly invoked scratch-only HID holder after this plan is
  committed: resolve one verified serial/model/path, acquire a read-only exclusive
  handle without report I/O, prove Win32 sharing error 32, and guarantee explicit
  release plus an at-most-90-second deadline and handle cleanup. No production or
  default-test behavior changes.
- [ ] 17. Root coordinates graceful stop and same-config canonical host restart with
  the holder active; observe the refused path error and the other deck's physical
  operation, release the holder, and verify normal recovery/temporary-error removal.
  Confirm helper exit and restored host configuration even on unsuccessful setup.
  Record the component result without completing task 11's frozen-player check.

## Conditional installed-player feasibility
- [ ] 18. After this plan is committed, create one uniquely owned scratch silent
  WAV. In a newly user-approved computer-control session, root launches the
  installed WMP only if no existing player would be reused, verifies ownership of
  the new process, and checks for an attributable Windows SMTC session using the
  existing read-only native probe with bounded observation. Stop on absent SMTC,
  setup/authentication/security prompts or uncertain ownership; configure nothing.
  Close only the owned new player normally, confirm exit and remove only its WAV.
  Preserve existing media playback, focus, configuration and processes. Record
  feasibility only: no debugger/freeze helper or build, and no completion of task
  11's frozen-player or gui-action-outcomes 6.3's uncertain/no-replay acceptance.
