# Tasks
- [x] Trace shared callback handling, worker reset ownership and GUI commands.
- [x] Implement and verify callback tolerance and bounded pass-through.
- [x] Implement periodic/manual retries and focused worker tests.
- [x] Add GUI Retry and verify renderer/dispatch.
- [x] Run native probes, Go checks, specs and canonical build; commit and restore host.
- [ ] Verify sustained operation on the user hardware.

Validation: native WebRTC probe passed echo/double-talk, transient omissions/mismatches/duplicates, 250 ms stable recovery, ten-second recurring-gap budget, five-second absent-reference budget, and production lifecycle checks. Neural probe passed both original models at three rates/three block sizes plus full-band at three block sizes, longer gap budgets and 267 ms stable recovery, exact fixture parity before/after reset, and full-band 10 kHz preservation. Worker tests passed periodic deadlines, reset-error cooldown, manual and failed-load retry, disabled/stale requests and preference preservation. GUI dispatch and disabled state passed; inspected renderer screenshot. go vet and all 60 specs passed; full Go suite only fails existing TestVoiceExample because config.voice.json is missing. Race tests unavailable with current CGO-disabled toolchain. Canonical build succeeded; no-argument host launch restored with personal settings unchanged. Sustained real-hardware acceptance remains open.

