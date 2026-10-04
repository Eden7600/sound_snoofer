# Tasks

## 1. Configuration and persistent intent

- [x] 1.1 Add optional Potato voice defaults and source/mode/monitor enums; verify config tests reject invalid enums, Banana and AUX playback conflicts while legacy fixtures remain valid.
- [x] 1.2 Implement per-config versioned state loading, atomic save, conflict detection and reset; verify missing, corrupt, incompatible, unwritable and concurrent-edit cases with temporary files.
- [x] 1.3 Resolve the same effective intent in plan/apply/watch/tui; verify command tests show identical desired routes and never persist live permission.
- [x] 1.4 Document profile ownership, sidecar precedence and reset semantics beside a new opt-in example; verify example decoding and default desk/Element/Off state.

## 2. Remote API and desired routes

- [x] 2.1 Extend snapshots and allowlisted numeric writes to edition-valid B sends; verify backend tests reject invalid strips, buses and non-boolean send values and propagate read failures.
- [x] 2.2 Implement preferred/effective mic resolution with stable webcam input 3; verify desk/lav separation, explicit webcam selection, disconnect/reconnect, ambiguity and unrelated input-3 ownership tests.
- [x] 2.3 Compile voice modes and protected B2/B3 routes into a final matrix; verify table-driven tests across enabled/disabled, every source, Direct/Element and no-source cases, including exclusion of app audio.
- [x] 2.4 Implement Off/Pre/Post monitoring and optional app-playback rules with explicit precedence over migration; verify A1/A2 changes, stale sends, Post-in-Direct and disabled-rule tests.
- [x] 2.5 Decouple capture planning from playback availability; verify no-playback tests retain valid voice delivery, suppress monitoring and preserve unrelated assignments.
- [x] 2.6 Document numeric ownership and semantic bus identities in remote-api/setup documentation; verify listed parameters match the implemented allowlist.

## 3. Verified transitions

- [x] 3.1 Introduce disable/configure/enable phases with phase-local expected state; verify every intermediate state in transition tests prevents competing voice paths and AUX-to-B2 feedback.
- [x] 3.2 Integrate phase verification, cancellation and fresh-state replanning with the existing controller; inject failures at disable, patch and enable steps and verify later enables stop, partial status is accurate and restart converges.
- [x] 3.3 Verify idempotence, debounce, manual drift, inventory invalidation and single-writer behavior against the extended matrix; document accepted switch gaps and lack of transactional rollback.

## 4. TUI rule management

- [x] 4.1 Add typed worker commands for source, mode, monitor, voice/app toggles and saved-state reset; verify save-before-apply, rejected stale commands, invalid reload and save-failure tests.
- [x] 4.2 Add the Rules view with editable and protected rows, context-aware keyboard hints and preferred/effective source; verify keyboard navigation, resize, disabled/inactive/pending/applied/error rendering and terminal text sanitization.
- [x] 4.3 Show observed versus desired state and external-audio-not-verified messaging; verify dry-run controls make zero mixer writes and live transitions retain writer ownership rules.
- [x] 4.4 Update README with controls, persistence and Direct recovery steps; verify an interactive terminal smoke test covers rule edits, tab switching, reload and clean quit.

## 5. Automated integration

- [x] 5.1 Run go test ./... and go vet ./... plus a Windows executable build; record results and preserve passing legacy fixed-route and studio tests.
- [x] 5.2 Run OpenSpec strict validation and confirm all implemented requirements map to tests or explicit hardware acceptance tasks; leave unperformed hardware checks unchecked.

## 6. Real-device acceptance

- [x] 6.1 Back up the working config and capture a routing snapshot before opt-in migration; verify the backup exists and new dry-run preview matches desk -> B2 -> AUX -> B3 with monitoring Off.
- [ ] 6.2 Verify desk Direct and Element pass-through separately in Discord, then verify a deliberately audible Element effect; record the actual graph and result without treating pass-through as plugin validation.
- [ ] 6.3 With a charged lav, verify input-2 selection and separation from the desk mic, then verify explicit webcam, Volt disconnect fallback and preferred-source restoration after reconnect.
- [ ] 6.4 Using headphones at low volume, verify Off/Pre/Post monitoring and switching voice mode; verify only the selected monitor path is audible and AUX-to-B2 stays off.
- [ ] 6.5 Verify AirPods/speaker changes and A1/A2 migration preserve app playback and monitoring rules; verify mic delivery with no eligible playback device and document Element behavior across A1 engine changes.
- [ ] 6.6 Close Element and verify no automatic dry bypass; select Direct to recover, restart Sound Snoofer and verify saved choices with dry-run default. Record any required manual Element recovery.
