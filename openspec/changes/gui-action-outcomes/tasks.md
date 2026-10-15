# Tasks

## 1. Planning and baseline
- [x] 1.1 Review this design against actual GUI/pipe/registry/worker paths and the concurrent state-preservation dialog changes; commit the strictly validated plan before implementation.
- [ ] 1.2 Add focused failing regressions for rapid Mic Mute → Playback Mute, queue-only acknowledgments, delayed dispatch errors and cross-screen attribution. Preserve existing test assertions and fixtures.

## 2. Correlated result transport
- [x] 2.1 Add private action/result types, optional claimed completion at registry dispatch, and sixteen pending Desktop.Send reservations without holding the state lock while waiting.
- [x] 2.2 Attach once-only replies to their exact controls session, add the noncoalescing reply lane and explicit downstream-queue rejection, and settle outstanding waits on disconnect without replay.
- [x] 2.3 Test out-of-order/duplicate replies, full desktop/tray/provider queues, full result lane, state flooding, concurrent state/animation reads, cancellation, legacy fixture cleanup and late replies after reopen. Run focused Go tests and commit this checkpoint.

## 3. Audio ordering and worker results
- [ ] 3.1 Add failing tests for both rapid mute orders, same-preference ABA, reload/reset/lifetime invalidation, external StateToken changes and unchanged contextual stale guards.
- [ ] 3.2 Add worker-owned generations only for the two explicit global mute controls; capture explicit booleans/generations, advance them on every relevant mutation/boundary, and preserve registry revision plus save-before-apply CAS.
- [ ] 3.3 Carry explicit completion through audio queue, worker branches, priority/output save→reload and audio-linked GUI adapters. Settle cancellation/stop/duplicate legacy paths without parsing Notice text or using GUI IDs as acknowledgment origins.
- [ ] 3.4 Test delayed stale/save/reload/native errors, queue saturation, provider stop, and no replay of uncertain recorder/restart commands. Run focused audio/control tests and commit this checkpoint.

## 4. Provider target capture and completion boundaries
- [ ] 4.1 Add and verify Stream Deck editor, App Audio, Now Playing, Windows Media and AEC adapters according to the design matrix. Capture mutable semantic targets and reject changed/unavailable/recreated contexts before side effects. Commit the validated group.
- [x] 4.1a Windows Media completes after the captured SendInput call; fake-native success/error, queue saturation and stop tests pass (25 repetitions), with vet. Remaining 4.1 providers stay open.
- [ ] 4.2 Add and verify Hue and Discord adapters: async call/nonce completion, context capture, partial errors, connection loss, cancellation and stop. Preserve Submitted boundaries for coalesced controls and multi-step untracked workflows. Commit the validated group.
- [ ] 4.3 Add and verify Soundboard captured play/preparation/stop/settings completion, including supersession, no late playback and audio-linked volume. Commit the validated group.
- [ ] 4.4 Audit every claimed path against the provider matrix; prove each completes exactly once on queue refusal, processing exit, replacement and shutdown. Retain provider readback/status tests and existing desired-state reconciliation; add no automatic one-shot retries.

## 5. Frontend attribution and lifecycle identity
- [ ] 5.1 Replace global pending gates with action/busy-key tracking throughout shared widgets and all screens. Keep timers active on null polls, keep timeouts reserved, and add bounded screen-local feedback without a result history.
- [ ] 5.2 Make fake Desktop.Send return correlated controllable results; test concurrent actions, wrong/out-of-order results, busy controls, delayed errors, local timeout followed by real completion, navigation, disappearance and reconnect without replay.
- [ ] 5.3 Add displayed config revision and owner-bound host proposal IDs; correlate startup's second queue and lifecycle outcomes. Test full/busy groups, stale selection, confirm/cancel IDs, external config edits, owner close and restart/disconnect uncertainty.
- [ ] 5.4 Integrate with state-preservation's frozen local dialog tickets and focus rules. Update the UI contract, inspect local feedback at 1280×820 and 800×600, run focused GUI and Go tests, and commit the validated checkpoint.

## 6. Source verification and coordinated native acceptance
- [ ] 6.1 Run full Go tests/vet and frontend model/type/lint/format/browser/accessibility checks, plus strict OpenSpec validation. Record results and commit final source fixes without modifying user-owned files or deployed outputs.
- [ ] 6.2 Coordinate the canonical build/restart and unchanged native desktop smoke after stable source checkpoints; confirm Send results do not block state/close and inspect packaged feedback at both sizes. Keep pending until actually performed.
- [ ] 6.3 Coordinate appropriate live integrations to verify rapid independent mutes, stale target refusal, a delayed native error and an uncertain one-shot without replay. Preserve previous launch configuration and document unavailable hardware acceptance separately.
