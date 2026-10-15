# Tasks
## Implementation
- [x] 1. `docs(audit)`: specify the audit hardening.
- [x] 2. `fix(gui)`: send priority and output edits from older lists as stale; GUI check (a targeted browser regression test belongs to `gui-test-harness`).
- [x] 3. `fix(nowplaying)`: bound WinRT media calls; build the media helper.
- [x] 4. `fix(build)`: stage all selected companions/models/licenses until esbuild, Tailwind and Go succeed; preserve the sole application output `bin/snoofer.exe`; repair neural-AEC setup; test failure/success/tagged paths and interrupted dependency recovery with isolated fake commands.
- [x] 5. `fix(voicemeeter)`: treat "no callback registered" as a clean stop; callback probe.
- [x] 6. `fix(deck)`: seed the key baseline from the current input report.
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
