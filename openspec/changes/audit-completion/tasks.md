# Tasks
- [x] 1. Review and commit the strictly validated plan before source/test changes.
- [x] 2. Remove unused worker protocol, adapt meaningful tests and older conflicting
  contracts; focused/repeated control and linked-provider checks; review and commit.
- [x] 3. Characterize and consolidate thumbnail encoding and OnOff; preserve policy
  and caller tests; focused tests/vet; review and commit.
- [x] 4. Reject all second live browser-name claims; prove admission/routing/reconnect
  regressions and retain generation/outcome coverage; review and commit.
- [ ] 5. Install the approved portable race compiler, verify full Go tests/vet/race,
  frontend gates and full browser suite, strict specs and canonical build; restore host.
- [ ] 6. Refine and validate bounded second-deck/frozen-player/uncertain-action
  preparation; coordinate approved execution and record actual acceptance or blocker.
Portable race compiler prepared: user authorized installation; official WinLibs GCC
16.2.0 / MinGW-w64 14.0.0 UCRT archive SHA256 verified against release metadata.
Compiler/runtime lookup and a real storage-package race run pass. Compiler, CGO
and PATH settings were process-scoped; no persistent toolchain settings changed.

Second-deck preparation: the revised scratch holder requests access zero with
read sharing, denying write opens without report I/O. Eight fake-only self-checks
passed twenty repetitions; hardware invocation still requires explicit opt-in.
This proves cleanup/guard logic only, not live sharing refusal.

- [ ] 7. Reconcile original audits with actual removals, narrowed security guarantee,
  measured validation, explicit no-dial-model waiver and any remaining evidence gap.

Checkpoint 4: Same-origin Chrome/Firefox and concurrent-claim regressions failed before the fix. All bridge tests pass 25 repetitions; complete Now Playing tests and vet pass. Independent root review confirms admission is synchronized, incumbents retain ownership and fresh reconnects retain generation guards. The original reject-second-live-connection option is fulfilled; first-claimant authentication is not asserted.

Checkpoint 3: Six decoded-pixel baselines were captured before extraction and passed unchanged afterward, including transparency, aspect ratio and nonzero source bounds. Source immutability, provider file/dimension limits and invalid shared inputs pass. Artwork, Now Playing, Soundboard, Discord and snoofer package tests and vet pass. Root review confirms all eight OnOff callers migrated and GIF/decoding/cache policies remain local.

Checkpoint 2: Removed only unused worker IDs, acknowledgements, deduplication and batch edits; callbacks, Origin notice clearing, scoped feedback and native acknowledgements remain. Linked package tests/vet and three strict specs pass; focused regressions pass 25 repetitions. A negative overlay proves the notice-clearing check detects regression. Independent and root reviews found no issues.

## Validation fixture ownership
- [ ] 5a. Fix confirmed Stream Deck fixture readiness and bounded waits; remove
  the unnecessary test-only frame drain. Prove the schedule, repeat focused checks,
  review and commit before rerunning full validation.
- [x] 5b. Initialize bridge Windows-session data before worker startup; retain
  matching assertions, verify repeated race checks, review and commit.

- [x] 5c. Await room brightness before the Hue Sync disconnect; retain immediate
  post-disconnect availability assertions, repeat with race instrumentation, review
  and commit.

Validation 5b, 5c: Bridge native sessions are initialized before worker startup; all bridge tests pass 25 race-detector runs. Hue establishes room brightness before disconnecting Sync and retains immediate assertions; 75 CPU-scheduled focused race cases and the full Hue race suite pass. Both package vet checks and root review pass.
