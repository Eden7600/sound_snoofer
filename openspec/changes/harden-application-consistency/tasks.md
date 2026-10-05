## 1 Planning alignment

- [x] 1.1 Mark contradictory Direct/Post, Element fallback, and live-default requirements explicitly superseded; retain historical and hardware evidence.
- [x] 1.2 Read and reconcile overlapping mixer, VR, recovery, tray, responsive-edit, and scoped-deck changes before implementation.

## 2 Persistence and selection correctness

- [x] 2.1 Reproduce CLI mute-journal misplacement; derive it from the config path and test cross-entry-point/profile isolation.
- [x] 2.2 Reproduce VR-only playback picker rejection; align configured candidate validation and test the edit/save/load/plan flow.
- [x] 2.3 Document recovery of an existing misplaced journal without guessing profile ownership or deleting it.

## 3 Microphone invariants

- [x] 3.1 Reproduce Mic Off on VR inputs 4/5 with unknown/conflicting recorder state; fix the managed-strip exception without weakening other guards.
- [x] 3.2 Reuse managed strip and mute target decisions in controller/TUI/deck; verify Element return and fixed-bus composition.
- [x] 3.3 Verify no recorder transport, computer capture, unrelated output, or Volt A1 changes from Mic Off/native mute beyond their defined ownership.

## 4 Worker and session lifecycle

- [x] 4.1 Reproduce voice-profile removal retaining Windows-default protection; publish explicit disabled policy.
- [x] 4.2 Add latest-policy delivery and bounded revocation acknowledgment before preview/ownership release; test delayed worker/cancellation.
- [x] 4.3 Reject stale restart requests/confirmations and bind confirmation to current context; retain fresh transport checks.
- [x] 4.4 Reproduce deck backlog across disconnect and changing recorder state; discard undispatched old-session work and preserve one-shot command meaning.
- [x] 4.5 Preserve Open controls focus across coalesced state updates without adding an unbounded command queue.

## 5 State and UX

- [x] 5.1 Separate successful observation age from worker progress; test stale data, blocked worker, and recovery.
- [x] 5.2 Keep persistent diagnostics through repeated deduplicated failures and reevaluate current health after restart outcomes.
- [x] 5.3 Replace presentation-string status decisions at default/VR boundaries with small typed outcomes.
- [x] 5.4 Keep essential status/details available in compact layouts and disable hidden editing below minimum size.
- [x] 5.5 Add control-specific action reasons, configured device labels, unavailable automatic recovery explanation, and concise contextual help.
- [x] 5.6 Add reset and unsent-edit discard interactions while preserving immediate ordinary close and one-shot transport.
- [x] 5.7 Validate bounds, keyboard flows, resize, NO_COLOR, Unicode, terminal sanitization, and requested/effective/pending semantics.

## 6 Targeted simplification

- [x] 6.1 Replace duplicate action enum numbering with authoritative aliases; keep wire compatibility.
- [x] 6.2 Consolidate atomic replacement only if state-lock/token and durability guarantees remain intact.
- [x] 6.3 Measure process-enumeration duplication and reuse one fresh process list per observation; preserve transaction freshness.
- [x] 6.4 Measure absent-deck discovery wakeups and bound discovery frequency if needed.
- [x] 6.5 Handle CLI output failures; remove demonstrated unreachable code and misleading helper names in touched scopes only.

## 7 Automated release validation

- [x] 7.1 Add focused regression/integration checks above and bounded stdlib fuzz seeds for config/HID parsing.
- [x] 7.2 Add scripts/check.ps1 using existing commands, strict exit-code handling, isolated outputs, and explicit skipped native/race checks.
- [x] 7.3 Run gofmt on touched Go files, full tests, vet, GUI build, strict OpenSpec validation, and supported race testing.
- [x] 7.4 Measure numeric/device input-to-ack and ack-to-readback separately; preserve existing latency/call-count checks.
- [x] 7.5 Run isolated dry-run tray/TUI smoke with copied config/sidecars and document results.

## 8 Physical acceptance and documentation

- [ ] 8.1 Listen-test Volt/webcam/headset switching, Off, mute/Element tails, fallback, and playback/capture continuity.
- [ ] 8.2 Verify native recording file/playback, Pre/Post content, active transport guards, and no transport replay on restart/reconnect.
- [ ] 8.3 Verify actual deck input/output, disconnect backlog, controls closed, Windows-default disable/contention, and SteamVR transitions.
- [ ] 8.4 Verify tray Quit, Explorer restart, sleep/resume, and hidden hotplug without replacing a running binary implicitly.
- [x] 8.5 Reconcile README, build instructions, release identity, current specs, and task evidence; leave every unperformed check pending.

## Verification record

See [verification.md](verification.md) for commands, measured results, native preview evidence, and remaining physical acceptance. Task 7.3 records race testing as unsupported, not passed.
