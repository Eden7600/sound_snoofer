# Tasks

Status: transport/source design and disposable-call probe complete on 2026-10-15.
Production implementation is authorized within the resolved scope. Implementation
and extension/provider/deck acceptance remain unperformed. The probe evidence in
observations.md establishes only the observed DOM candidate and lifecycle signals;
it does not verify implemented stale-action rejection. Each checkpoint remains a
separately reviewable block. Commit validated blocks according to AGENTS.md. Live
control requires session permission.

- [x] 1. Resolve the transport/source design before production edits: trace both
  consumers end to end; choose the narrow owner/placement, registration/shutdown,
  protocol compatibility, sender validation, command-result schema, deadlines and
  trust treatment; record concrete decisions in design.md. Confirm control IDs,
  call identity and selector/call-state feasibility requirements. Do not add a
  generic framework or silently expand pairing/authentication scope.
- [x] 2. With separate desktop-control permission, run an approved disposable Meet
  probe and record browser/layout/language, selectors, joined-call signals, observed
  state and one-action readback, background-tab behavior, prejoin/leave/rejoin and
  cleanup. Include mic/camera-on tests in the approved session scope without
  repeated prompts for covered steps. Amend the design with proven coverage or a
  blocker before dependent implementation. Evidence: observations.md; stale-command
  rejection and extension/provider/deck acceptance remain tasks 4, 6 and 7.
- [ ] 3. Implement the documented narrow transport reuse with focused tests for
  Meet-only, Media-only and both running; stop/restart ownership, protocol mismatch,
  obsolete sockets/ports, duplicate-browser admission and existing media behavior.
  Exercise each FIFO16 overflow, exact-connection retirement, the documented
  same-browser outage, other-socket isolation, generation reconciliation, bounded
  writes/results, capability transitions and fresh artwork recovery. Use the
  concrete lifecycle/schema/deadline regression checklist in design.md.
  Preserve the existing first-claimant limitation unless the design explicitly
  resolves it. Do not modify unrelated bridge behavior or personal configuration.
- [ ] 4. Implement the Meet adapter/provider and meaningful fake/fixture tests:
  known/unknown mic and camera state, external changes, explicit desired values,
  simultaneous controls, busy behavior, actual correlated outcomes and no replay;
  invalid origin/frame, old document/socket/call lifetime, same-URL rejoin, selection
  changes, a second call appearing after initial selection, no fallback after target
  loss, prejoin exclusion, permission/unsupported states and bounded cancellation.
  Use fixtures derived from the approved probe, with private call data removed.
  Cover joined-root/control replacement, revision changes away and back, observer
  readback, per-tab sample freshness, repeated/stale samples and heartbeat expiry.
  Cover port-loss adapter reconnect and immediate cancellation on socket replacement
  or feature disable, requiring fresh enablement/observation before new work.
  Do not mark stale-rejection coverage complete from the live DOM probe alone.
- [ ] 5. Add independent Meetings cards/gates, target selection and truthful local
  state/status; update its UI-contract section and focused GUI/attention fixtures.
  Make controls/status bindable in the existing deck editor; verify rendered known,
  unknown, pending and unavailable states plus keyboard/accessibility behavior.
  Demonstrate Discord/global mute behavior unchanged and no default/personal layout
  migration. Preserve source-muted distinction when audio state is available.
- [ ] 6. Run appropriate Go tests/vet/race, extension tests, frontend gates and GUI
  checks, strict validation of this change, and the canonical build for production
  changes. Record evidence and restore the prior host launch configuration as
  required by AGENTS.md; distinguish automated checks from hardware acceptance.
- [ ] 7. In a separately approved disposable-call/deck session, verify the published
  browser coverage: first press, state changes made in Meet, background tab, two
  Meet calls, Discord plus Meet, upstream Mic Mute, reconnect/reload and same-URL
  rejoin without unexpected unmute/camera-on. Verify stale input refusal, uncertain
  result without replay, normal cleanup and settings/layout preservation. Leave
  unavailable checks unchecked with their exact limits; do not use a user's active
  meeting or modify a personal deck layout without a separate placement request.

Default/user layout placement and Show Call, hand controls, Step Away, screen share,
Leave, push-to-talk and automatic meeting modes are follow-ups, not completion tasks.
