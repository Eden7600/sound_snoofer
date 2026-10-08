# Tasks
- [x] Trace both queue ownership paths and retry generations.
- [x] Reproduce overflow after repeated pre-roll resets on the existing DLL.
- [x] Implement consumer-owned output discard and verify recovery across reset paths.
- [x] Run native regressions and Go/spec checks, build, commit and restore host.
- [ ] Confirm long-running hardware stability.

Reproduction: the prior DLL failed at warm-up reset cycle 10 with reason 9 (queue overflow). The corrected DLL passes 30 alternating explicit/lifecycle/discontinuity resets and resumes Active. Output discard is consumer-owned and keeps generation checks for concurrent old results.


Final validation: all three neural variants passed 30 alternating warm-up resets followed by sustained Active, plus existing callback/rate/block, recovery budget, pass-through and deadline probes. Original models retain exact fresh/recovered fixture parity; full-band 10 kHz gain remains +0.03 dB. Full Go suite, go vet and all 66 OpenSpec items passed. Canonical build succeeded and normal no-argument Snoofer host was restarted with saved preferences unchanged. Race detector remains unavailable in the current CGO-disabled toolchain; live long-duration hardware stability is not claimed.

