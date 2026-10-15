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
- [x] 9. `fix(nowplaying)`: initially restrict replacement to the same origin; tested. Superseded by audit-completion task 4, which rejects every second live browser-name claim and retains reconnect generation guards.
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
in task 15; at that point a refused second Deck and frozen media-player acceptance remained open. Startup with
Voicemeeter absent now passes, as separately recorded in audit-reliability-fixes task 16.

Task 11 frozen-player component passes, 2026-10-15: run
`f341fc3299a24e00aa8fe066dea403b1` verified the exact process creation times/paths for
healthy PID 28360 and stalled PID 71744. All 32 observed stalled threads were frozen
with stable inventory from 19:59:54.035Z to 20:00:08.983Z; the independent 15-second
timer restored each owned suspension exactly once, without errors. Live Snoofer's
native UI showed healthy title beat 29 at 19:59:58.672Z and beat 33 at 20:00:04.288Z,
both during the freeze. The stalled title fell back and returned as beat 39 at
20:00:21.391Z after resume. This proves fresh healthy updates, not interpolated clocks.
No command was sent in this run. Both players exited normally with code 0; native
probes confirmed zero exact owned sessions, and only their generated 160,044-byte
WAVs were removed. Evidence is retained in chat scratch
work/owned-media-player/runs/f341fc3299a24e00aa8fe066dea403b1-stalled/freeze.jsonl and
work/owned-media-player/runs/f341fc3299a24e00aa8fe066dea403b1-ui.json.
Task 11 remains unchecked only for the blocked refused-second-deck component.

## Second-deck refusal acceptance helper
- [x] 16. Prepare the explicitly invoked scratch-only HID holder after this plan is
  committed: resolve one verified serial/model/path, acquire a read-only exclusive
  handle without report I/O, prove Win32 sharing error 32, and guarantee explicit
  release plus an at-most-90-second deadline and handle cleanup. No production or
  default-test behavior changes.

Task 16 preparation evidence: the scratch holder was reviewed, including atomic readiness.
Eight fake-handle cases passed 20 repetitions; its native entry skipped without opt-in. These
are preparation checks only; the unsuccessful live setup is recorded under task 17.

- [ ] 17. Root coordinates graceful stop and same-config canonical host restart with
  the holder active; observe the refused path error and the other deck's physical
  operation, release the holder, and verify normal recovery/temporary-error removal.
  Confirm helper exit and restored host configuration even on unsuccessful setup.
  Record the component result without completing task 11's frozen-player check.

Task 17 attempted, setup refused: after the exact host and controls child stopped normally,
the helper verified Plus serial A00WA6261J4HRK/model/path. Its exclusive GENERIC_READ/share-0
open itself returned a sharing violation; it exited 1 in 0.01 seconds without acquiring an
exclusive handle. No fallback or unrelated-app closure was attempted. This error belongs to
the helper's acquisition, not Snoofer's refused-deck reporting; that acceptance remains unproven.
The canonical host was restored with the same configuration. Both decks were Connected/display
On; callback Active (Buffers 4025), Recorder Stopped, Mic Stack On and both mutes Off were
observed. The original configuration fingerprint was retained and no disposable MP3, silent player WAV or WMP test process
remained. Task 17 stays unchecked. Chat scratch log:
work/deck-refusal-holder/runs/8d068724fc194c61896a4f21d7ad16d1/holder.stdout.log.

Task 17 follow-up, 2026-10-15: both bounded access-zero refinements also failed
before acquiring a held handle. Share-read failed with Win32 sharing error 32 in
work/deck-refusal-holder/runs/1d8c35eade704434845e263d47267509/holder.stdout.log;
the single final share-write attempt approved by `97b4e9c` failed with the same
error in work/deck-refusal-holder/runs/f18827b971f54e129cd69dee8ca6a39b/holder.stdout.log.
Each verified the exact Plus serial/model/path first. Incompatible device sharing
blocks this setup; the owner/access mode is unknown, and no further holder modes,
ACL/driver changes or unrelated-app closure will be attempted. Snoofer's refused-path
reporting and other-deck operation under refusal remain unverified. Before the final
attempt, host PID 58572 and controls PID 50120 exited normally; the same configuration
was restored as host PID 70128. The user confirmed both decks normal after the first
refinement. Final native UI recovery shows Plus A00WA6261J4HRK and Plus XL
AD4MA61312DUNP Connected with Display On, callback Active (Buffers 14239), Recorder
Stopped and Windows defaults Verified. Media shows only the user's Brave cards after
owned-player cleanup. This is UI recovery, not a new physical press confirmation;
refused-deck acceptance remains unproven and task 17 remains unchecked.

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

Task 18 attempted, not completed: the newly owned WMP process exposed no attributable
Windows SMTC session (only Brave) and had MainWindowHandle 0 with no targetable window.
Computer-control approval/wait timeouts exceeded the agreed window, so root stopped control.
CloseMainWindow returned false. The user then explicitly authorized termination of only the
owned test process; PID, executable path and start time were verified, termination and exit
were confirmed, and the exact 160,044-byte silent WAV was removed. Task 18 stays unchecked
because its bounded normal-close criteria were not fulfilled. This WMP attempt established
neither frozen-player nor uncertain-one-shot/no-replay acceptance. The separate owned-player
freeze later fulfilled task 11's frozen-player component above; uncertain one-shot remains open.

Final reconciliation: the unsuccessful installed-WMP route is abandoned, not a new
requirement to repeat. Frozen-player acceptance passes via the owned-player evidence
above; tasks 11/17 remain unchecked for genuine refused-deck acceptance. A later GUI
Pause during an owned freeze was Processed once with one effect/no replay, but did
not expose an uncertain native result (gui-action-outcomes task 6.3). All owned media
helper processes exited; their sessions and silent WAVs are gone. Host/controls 70128/11816 retain the same
configuration; audio-state and user-owned smoke-script fingerprints are unchanged.
The test notice is dismissed, Media shows only user Brave cards, and computer control
was released at 20:05Z. No further live attempts are planned.
