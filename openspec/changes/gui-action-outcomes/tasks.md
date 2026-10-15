# Tasks

## 1. Planning and baseline
- [x] 1.1 Review this design against actual GUI/pipe/registry/worker paths and the concurrent state-preservation dialog changes; commit the strictly validated plan before implementation.
- [x] 1.2 Add focused failing regressions for rapid Mic Mute → Playback Mute, queue-only acknowledgments, delayed dispatch errors and cross-screen attribution. Preserve existing test assertions and fixtures.

## 2. Correlated result transport
- [x] 2.1 Add private action/result types, optional claimed completion at registry dispatch, and sixteen pending Desktop.Send reservations without holding the state lock while waiting.
- [x] 2.2 Attach once-only replies to their exact controls session, add the noncoalescing reply lane and explicit downstream-queue rejection, and settle outstanding waits on disconnect without replay.
- [x] 2.3 Test out-of-order/duplicate replies, full desktop/tray/provider queues, full result lane, state flooding, concurrent state/animation reads, cancellation, legacy fixture cleanup and late replies after reopen. Run focused Go tests and commit this checkpoint.

## 3. Audio ordering and worker results
- [x] 3.1 Add failing tests for both rapid mute orders, same-preference ABA, reload/reset/lifetime invalidation, external StateToken changes and unchanged contextual stale guards.
- [x] 3.2 Add worker-owned generations only for the two explicit global mute controls; capture explicit booleans/generations, advance them on every relevant mutation/boundary, and preserve registry revision plus save-before-apply CAS.
- [x] 3.3 Carry explicit completion through audio queue, worker branches, priority/output save→reload and audio-linked GUI adapters. Settle cancellation/stop/duplicate legacy paths without parsing Notice text or using GUI IDs as acknowledgment origins.
- [x] 3.4 Test delayed stale/save/reload/native errors, queue saturation, provider stop, and no replay of uncertain recorder/restart commands. Run focused audio/control tests and commit this checkpoint.

## 4. Provider target capture and completion boundaries
- [x] 4.1 Add and verify Stream Deck editor, App Audio, Now Playing, Windows Media and AEC adapters according to the design matrix. Capture mutable semantic targets and reject changed/unavailable/recreated contexts before side effects. Commit the validated group.
- [x] 4.1a Windows Media completes after the captured SendInput call; fake-native success/error, queue saturation and stop tests pass (25 repetitions), with vet. Remaining 4.1 providers stay open.
- [x] 4.1b AEC reports settings-save outcomes and captured-engine retry submission; changed-target, failed-reconfigure, queue-full and stop regressions pass (25 repetitions), with vet. Remaining 4.1 providers stay open.
- [x] 4.1c App Audio captures app/session/focus lifetimes and explicit mute intent, returns finite-write and settings outcomes, and preserves independent readback status. Full package tests pass 25 repetitions with vet; remaining 4.1 providers stay open.
- [x] 4.1d Stream Deck completes editor/save paths, captures focused deck lifetime for navigation, and rejects stale work. Save/validation/full-queue/stop checks pass 25 repetitions; navigation checks pass 30 repetitions and reject a late-target negative overlay. Full package tests and vet pass; Now Playing remains open.
- [x] 4.1e Commit the Now Playing identity refinement before implementation. Add native session instance tokens and fake-COM identity checks; add bridge connection generations and extension protocol 2 frame instances with stale-port/socket guards and explicit toggle/mute values. Verify reorder, removal/reappearance, reconnect, navigation, incompatible versions and missing-token refusal without live integrations.
- [x] 4.1f Complete Now Playing settings/focus/native/browser/dial adapters using those captured identities. Add initial failing queued-focus/lifetime cases, settings and native error outcomes, scrub replacement, queue-full/stop/late-completion tests; retain independent readback status and no-replay checks. Run focused Go, extension and isolated native checks and commit the checkpoint; Protocol 2 Chrome page and isolated Brave extension smoke also pass; packaged native companion and host acceptance stay in section 6.
- [x] 4.2 Add and verify Hue and Discord adapters: async call/nonce completion, context capture, partial errors, connection loss, cancellation and stop. Preserve Submitted boundaries for coalesced controls and multi-step untracked workflows. Commit the validated group. Hue full package passes in 12.4 s with vet; core outcomes pass 10 repetitions and resource/queue/late-callback edges pass 25. Independent review is clean.
- [x] 4.2a Discord captures connection/channel lifetimes and desired values, completes nonce and linked-audio calls, rejects overlapping Camera/Share toggles until their own response/readback boundary, and settles timeout/drop/stop once. Focused regressions pass 25 repetitions; full Discord/audio/snoofer checks and vet pass. Hue remains open.
- [x] 4.3 Soundboard completes captured play/preparation/start, Stop and Overlap saves; audio-linked volume retains the audio adapter completion. Supersession, stale clips, explicit prepare/native/save errors, queue saturation and shutdown pass 25 repetitions with vet and independent review.
- [x] 4.4 Audit every claimed path against the provider matrix; prove each completes exactly once on queue refusal, processing exit, replacement and shutdown. Retain provider readback/status tests and existing desired-state reconciliation; add no automatic one-shot retries. All claimed providers were independently reviewed; focused completion checks pass across the non-Hue packages, and Hue cancellation/replacement/partial-error tests pass.

## 5. Frontend attribution and lifecycle identity
- [x] 5.1 Replace global pending gates with action/busy-key tracking throughout shared widgets and all screens. Keep timers active on null polls, keep timeouts reserved, and add bounded screen-local feedback without a result history.
- [x] 5.2 Make fake Desktop.Send return correlated controllable results; test concurrent actions, wrong/out-of-order results, busy controls, delayed errors, local timeout followed by real completion, navigation, disappearance and reconnect without replay.
- [x] 5.3 Add displayed config revision and owner-bound host proposal IDs; correlate startup's second queue and lifecycle outcomes. Test full/busy groups, stale selection, confirm/cancel IDs, external config edits, owner close and restart/disconnect uncertainty.
- [x] 5.3a Host proposals retain owner, action, config revision and one-use identity; current-config selection and external-save guards are tested. Startup completion crosses the second core queue with captured registration and stop/full-queue checks. App/snoofer tests and vet pass; focused lifecycle/core checks pass 25 repetitions. Frontend integration remains open.
- [x] 5.4 Integrate with state-preservation's frozen local dialog tickets and focus rules. Update the UI contract, inspect local feedback at 1280×820 and 800×600, run focused GUI and Go tests, and commit the validated checkpoint. Final frontend run: 289 tests in 46.2 s; type/lint/format and 15 model tests pass, including independent feedback announcements while another action remains pending.

## 6. Source verification and coordinated native acceptance
- [x] 6.1 Run full Go tests/vet and frontend model/type/lint/format/browser/accessibility checks, plus strict OpenSpec validation. Record results and commit final source fixes without modifying user-owned files or deployed outputs.
- [x] 6.2 Coordinate the canonical build/restart and unchanged native desktop smoke after stable source checkpoints; confirm Send results do not block state/close and inspect packaged feedback at both sizes. Passed against the canonical c3cff86 build; live outcomes remain in 6.3.
- [ ] 6.3 Coordinate appropriate live integrations to verify rapid independent mutes, stale target refusal, a delayed native error and an uncertain one-shot without replay. Preserve previous launch configuration and document unavailable hardware acceptance separately.
- [ ] 6.3a With Soundboard idle and ready, use one unique, uncached disposable invalid MP3 through the normal catalogue and GUI. Verify the asynchronous production Media Foundation normalization HRESULT reaches the originating action as Failed, with no playback. Remove only that file and confirm catalogue removal; preserve user files/configuration and current audio choices. This covers delayed native failure only, not Unknown, uncertain one-shot or no replay.

2026-10-15 partial task 6.3 live evidence: the user confirms rapid Mic Mute + Playback Mute
pairs work On then Off. Separate GUI checks showed per-control Processed/Saved results and
both mutes restored Off. A real recording also paused/resumed/stopped successfully; the
no-eligible-recording-sources refusal was validation, not a delayed native failure. Real
stale-target rejection, delayed native error and uncertain one-shot/no replay remain open.
