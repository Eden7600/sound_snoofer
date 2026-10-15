# Tasks
- [x] 1. Review and commit the strictly validated plan before source/test changes.
- [x] 2. Remove unused worker protocol, adapt meaningful tests and older conflicting
  contracts; focused/repeated control and linked-provider checks; review and commit.
- [x] 3. Characterize and consolidate thumbnail encoding and OnOff; preserve policy
  and caller tests; focused tests/vet; review and commit.
- [x] 4. Reject all second live browser-name claims; prove admission/routing/reconnect
  regressions and retain generation/outcome coverage; review and commit.
- [x] 5. Install the approved portable race compiler, verify full Go tests/vet/race,
  frontend gates and full browser suite, strict specs and canonical build; restore host.
- [x] 6. Refine and validate bounded second-deck/frozen-player/uncertain-action
  preparation; coordinate approved execution and record actual acceptance or blocker.

Portable race compiler prepared: user authorized installation; official WinLibs GCC
16.2.0 / MinGW-w64 14.0.0 UCRT archive SHA256 verified against release metadata.
Compiler/runtime lookup and a real storage-package race run pass. Compiler, CGO
and PATH settings were process-scoped; no persistent toolchain settings changed.

Validation 5: Full Go tests/vet and `go test -race -p 4 ./... -timeout 90s` pass after
the test-only fixture repairs in `07b616f` and `ad94e6e`. Strict frontend gates,
297 browser tests in 42.2 seconds and all 106 OpenSpec changes pass. The canonical
build and native async/identity/callback self-tests pass; no production source changed
after that build. Host PID 73604 was restored with `--config bin/snoofer.json` and
unchanged persisted audio-state and user-owned smoke-script fingerprints. Remaining
native acceptance stays under task 6; Mini/MK.2/original XL acceptance is explicitly
waived by the user, not passed.

Earlier second-deck preparation: the revised scratch holder requests access zero with
read sharing, denying write opens without report I/O. Eight fake-only self-checks
passed twenty repetitions; hardware invocation still requires explicit opt-in.
This proves cleanup/guard logic only, not live sharing refusal.

Task 6 second-deck result: access-zero/share-read and the single final
access-zero/share-write attempt both failed acquisition with Win32 sharing error 32
after verifying the exact Plus serial. No held handle was acquired, so refused-deck
acceptance remains blocked by incompatible device sharing, not passed. No further
holder attempts are planned. Host PID 58572 and controls PID 50120 exited normally;
the same configuration was restored as PID 70128. Both decks were user-confirmed
normal after the first refinement. Final native UI recovery shows both exact serials
Connected/Display On, callback Active (Buffers 14239), Recorder Stopped and Windows
defaults Verified. This is UI recovery, not a new physical press confirmation. Full
logs, serials and limits are recorded in audit-hardening task 17.

- [x] 7. Reconcile original audits with actual removals, narrowed security guarantee,
  measured validation, explicit no-dial-model waiver and any remaining evidence gap.

Checkpoint 4: Same-origin Chrome/Firefox and concurrent-claim regressions failed before the fix. All bridge tests pass 25 repetitions; complete Now Playing tests and vet pass. Independent root review confirms admission is synchronized, incumbents retain ownership and fresh reconnects retain generation guards. The original reject-second-live-connection option is fulfilled; first-claimant authentication is not asserted.

Checkpoint 3: Six decoded-pixel baselines were captured before extraction and passed unchanged afterward, including transparency, aspect ratio and nonzero source bounds. Source immutability, provider file/dimension limits and invalid shared inputs pass. Artwork, Now Playing, Soundboard, Discord and snoofer package tests and vet pass. Root review confirms all eight OnOff callers migrated and GIF/decoding/cache policies remain local.

Checkpoint 2: Removed only unused worker IDs, acknowledgements, deduplication and batch edits; callbacks, Origin notice clearing, scoped feedback and native acknowledgements remain. Linked package tests/vet and three strict specs pass; focused regressions pass 25 repetitions. A negative overlay proves the notice-clearing check detects regression. Independent and root reviews found no issues.

## Validation fixture ownership
- [x] 5a. Fix confirmed Stream Deck fixture readiness and bounded waits; remove
  the unnecessary test-only frame drain. Prove the schedule, repeat focused checks,
  review and commit before rerunning full validation.
- [x] 5b. Initialize bridge Windows-session data before worker startup; retain
  matching assertions, verify repeated race checks, review and commit.

- [x] 5c. Await room brightness before the Hue Sync disconnect; retain immediate
  post-disconnect availability assertions, repeat with race instrumentation, review
  and commit.

Validation 5b, 5c: Bridge native sessions are initialized before worker startup; all bridge tests pass 25 race-detector runs. Hue establishes room brightness before disconnecting Sync and retains immediate assertions; 75 CPU-scheduled focused race cases and the full Hue race suite pass. Both package vet checks and root review pass.

Validation 5a: A controlled connection publication proves the old edit is rejected without saving and a connected edit saves once. Readiness waits now precede outcome/editor edits, save-entry waits are bounded, and the unnecessary frame drain is removed. Each affected outcome/device case passed 900 normal and 60 race repetitions across CPU schedules; the editor case passed 30 race runs. Full Stream Deck race tests, vet and root review pass.

Earlier native media preparation: the scratch owned-player helper compiles with MSVC
/W4 /WX and passes pure identity, silent-WAV and deadline checks. Its scratch Go
overlay probe passes exact-identity/uniqueness/operation checks and skips without
opt-in before loading any DLL. Root reviewed both helpers. At that earlier checkpoint
no live player, media probe or revised HID holder had run; a desktop-control session
still needed approval. Later execution and cleanup are recorded below.

Task 6 owned-media feasibility: run `9ac52397863d4af2b71cd23ecf0e9d6f` exposed exact
healthy/stalled sessions. One Next call returned accepted/S_OK in 510 microseconds;
the owned stalled player logged one command and one effect 2.039 seconds later.
This establishes delayed effect, not a frozen player or an uncertain native result.
Owned PIDs 50084 and 63264 cleaned up normally with exit code 0, read-only removal
probes found zero matching sessions, and both 160,044-byte silent WAVs were removed.
The removal probes intentionally report their missing required session as failure;
they sent no commands. The native Media UI shows only the user's Brave cards after
owned cleanup. Logs remain under work/owned-media-player/runs/9ac52397863d4af2b71cd23ecf0e9d6f-{healthy,stalled}.
At that checkpoint, the bounded freeze helper was being prepared under committed plan
`9373dbf`; frozen-player and uncertain-one-shot/no-replay acceptance remained open.

Task 6 frozen-player acceptance passes in fresh run `f341fc3299a24e00aa8fe066dea403b1`.
Exact owned process identities were verified: healthy PID 28360, stalled PID 71744.
All 32 observed stalled threads stayed frozen from 19:59:54.035Z to 20:00:08.983Z;
the independent 15-second timer restored each suspension once without error or thread
inventory change. Native Snoofer captured healthy title beat 29 at 19:59:58.672Z and
beat 33 at 20:00:04.288Z, both inside the freeze. These are new title updates, not
interpolated clocks. The stalled title fell back during the freeze and returned as
beat 39 at 20:00:21.391Z. No command was sent. Both players exited normally with code
0, zero exact owned sessions remained, and only their generated 160,044-byte WAVs
were removed. Evidence: work/owned-media-player/runs/f341fc3299a24e00aa8fe066dea403b1-stalled/freeze.jsonl
and work/owned-media-player/runs/f341fc3299a24e00aa8fe066dea403b1-ui.json.

Final task 6 outcome: run `f97568a469ee4a799e3fabdda8f722d3` froze/resumed 31 threads,
but computer control rejected stale element 59720 before clicking; no action was sent.
Final run `260a5fe8f8134261884899ba5777c763` sent one actual GUI Pause during a
33-thread freeze. Snoofer showed Sending then Processed; the resumed player logged
exactly one command and one effect through cleanup. The spaced snapshots do not
establish a native timeout or correlated uncertainty, so that acceptance remains unverified; the GUI
does not establish a native HRESULT or call latency. Details and timestamps are in
gui-action-outcomes task 6.3 and chat scratch work/owned-media-player/runs/260a5fe8f8134261884899ba5777c763-ui.json,
with player/supervisor logs in the matching -healthy and -stalled directories.
Both runs' players exited normally with code 0, exact native sessions disappeared,
and only their generated 160,044-byte WAVs were removed. Every supervisor resume
succeeded. No more live attempts are planned.

Tasks 6 and 7 close the preparation/evidence/reconciliation procedure, not every
physical acceptance claim. Refused-deck remains blocked by sharing and real native
uncertainty is unverified; those original acceptance tasks stay unchecked. No-dial
models are waived, not passed. Final host/controls PIDs 70128/11816 retain the same
configuration and unchanged audio-state/user-owned check-desktop fingerprints. No
owned helper processes or silent WAVs remain; the test notice is dismissed and native
Media shows only user Brave cards. Computer control was released at 20:05Z.
