# Tasks
- [x] Trace both queue ownership paths and retry generations.
- [x] Reproduce overflow after repeated pre-roll resets on the existing DLL.
- [x] Implement consumer-owned output discard and verify recovery across reset paths.
- [ ] Run native regressions and Go/spec checks, build, commit and restore host.
- [ ] Confirm long-running hardware stability.

Reproduction: the prior DLL failed at warm-up reset cycle 10 with reason 9 (queue overflow). The corrected DLL passes 30 alternating explicit/lifecycle/discontinuity resets and resumes Active. Output discard is consumer-owned and keeps generation checks for concurrent old results.

