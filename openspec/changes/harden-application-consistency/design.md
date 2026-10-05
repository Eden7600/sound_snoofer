> Implementation completed for the software scope on 2026-10-05; see verification.md and tasks.md. The audit below is the historical baseline. Hardware acceptance remains pending. The hardware session uses a FIFO disconnect barrier (cancel and join readers before publishing disconnect) rather than adding session IDs to every event.

## Recommendation

Improve the existing application in five passes: correct unsafe or inconsistent behavior, unify state semantics, improve everyday controls, simplify proven duplication, then complete hardware acceptance. Add a regression for each real failure before fixing it. Preserve the deterministic planner, single serialized Voicemeeter worker, bounded queues, saved preferences, and native adapters.

This is an implementation plan, not a claim that the findings have been fixed. Findings below come from source tracing unless explicitly described as executed validation. Suspected timing failures require deterministic reproduction before implementation. No live audio or GUI smoke was performed during this audit.

## Audit baseline

Repository: `C:\Users\Eden7600\Documents\sound_snoofer`, commit `22d71ae`. The only pre-existing tracked modification was AGENTS.md. Its updated instructions were read and preserved. There is one repository AGENTS.md and no deeper override found. The audit traced launch, CLI, tray IPC, TUI drafts, worker actions, persistence, planning, execution/readback, recorder guards, mute ownership, Windows defaults, and Stream Deck dispatch/presentation. It sampled native adapters and existing tests; this is not a formal proof of every native call or every possible audio graph.

Executed checks:

- `go test ./... -timeout 30s`: passed all 13 packages.
- `go vet ./...`: passed.
- `go test ./... -cover -timeout 30s`: passed. Routing 89.3%, TUI 83.6%, controller 80.0%, CLI 75.3%, config 71.2%, control 52.2%, Stream Deck 55.0%, Voicemeeter 51.2%, Windows audio 19.1%, desktop 9.7%. Coverage identifies where to inspect; it does not establish behavioral correctness.
- Windows GUI-subsystem build with `-trimpath -ldflags '-H=windowsgui'`: passed, using a scratch output executable that was not launched.
- Strict validation of the existing 29 OpenSpec changes: passed.
- `go test -race ./...`: unavailable because CGO is disabled. No toolchain changes were made.

The first test attempt failed to access the default Go build cache. The successful checks used a cache in the writable Codex workspace. Native probe and desktop smoke tests are opt-in and remain unperformed; a green suite does not mean they passed. The repository contains 165 Test/Fuzz function declarations, including platform-gated and opt-in cases; this is not a count of executed scenarios.

## What should stay

- Pure routing calculations and fresh readback before dependent writes.
- Save-before-apply, atomic file replacement, external-edit tokens, and immutable published state conventions.
- Explicit one-shot recorder transport with no retry after an uncertain result.
- Device presence, regex ambiguity, requested preference, effective fallback, and observed mixer state as separate concepts.
- Existing small consumer interfaces, stdlib testing, and the current Bubble Tea UI.
- Mic Off, native mute, recording inclusion, and transport as distinct controls.
- Explicit hardware acceptance. Readback is not proof of audible sound.

## Work package 0 Resolve planning contradictions

Priority: prerequisite. Size: small. This is documentation work, not a routing change.

The active `normalize-direct-recording-stage` requirement says Direct permanently saves Pre and returning to Element retains Pre. The active `element-process-fallback` requirement, current implementation, and updated AGENTS.md preserve Post and derive effective Pre. Older `selectable-voice-routing` acceptance also says no automatic dry bypass and dry-run default, contrary to current fallback/live-default requirements. `b1-recorder-controls` retains an old Direct/Post inactive acceptance case.

Record explicit supersession to the current AGENTS.md behavior before implementing this plan. Do not silently implement older requirements or mark their obsolete acceptance scenarios complete. Keep historical intent readable and identify the replacement scenario. `openspec/config.yaml` also needs its outdated Post wording reconciled. Strict validation checks syntax, not these contradictions.

Keep `recording-sources-playback-targets` separate: it is a proposal with unchecked implementation tasks, not an existing capability to reuse. Do not mix its capture/playback redesign into correctness fixes.

Done when each affected behavior has one current interpretation, older contradictory acceptance is replaced or explicitly superseded, and hardware tasks remain unchecked unless actually performed.

## Work package 1 Correct persistence and device selection

Priority: high. Size: small to medium.

**Confirmed path defect.** `internal/cli/cli.go:189` constructs `Mixer.Path` from the DLL flag (`*path + ".mutes.json"`). The worker uses the config path. With no DLL override, CLI writes `.mutes.json` in its working directory; with an override it targets a DLL-adjacent filename. The same profile therefore has different mute ownership depending on entry point, and unrelated profiles can share the CLI journal.

Use the resolved config path in the CLI. Do not automatically import a stray working-directory journal: its profile ownership is ambiguous. Document recovery for an existing misplaced journal and preserve it for inspection. Test apply/watch with two configs, a different working directory, and both empty/explicit DLL paths. Verify no writes land beside the DLL and pre-existing manual mute survives restart and unmute.

**Confirmed eligibility mismatch.** `routing.PlaybackOptions` in `internal/routing/availability.go` includes VR playback candidates through `vrPlaybackCandidates`; `Intent.Validate` in `internal/config/voice.go` accepts only ordinary studio playback patterns and ASIO. A headset listed in the picker can consequently fail to save unless it also matches an ordinary pattern.

Make persisted-name validation accept configured VR playback ownership too, while keeping connected/unique eligibility in the worker's fresh observation check. Reuse existing candidate rules without introducing a config-to-routing import cycle: a narrow config-owned candidate helper is sufficient if needed. Preserve saved choices when the endpoint temporarily disappears.

Tests: VR-only output selection, disconnected saved preference, runtime stopped/unknown, duplicate names, overlapping patterns, and save failure. A listed eligible choice must round-trip through edit, save, load, plan, and effective status.

## Work package 2 Make Mic Off and mute semantics complete

Priority: high. Size: medium.

**Confirmed controller omission.** The planner's `managedMicStrips` includes the configured VR input. However, the Mic Off exception to recorder conflicts in `internal/controller/topology.go:89` only recognizes strips 0, 1, 2, and 6. With a VR strip B1 send on and recorder state unknown/conflicting, execution can stop before clearing that send. This contradicts the Mic Off invariant.

Use one concrete definition of managed microphone strips, or explicit operation metadata derived from it, for this exception. Do not relax recorder conflict guards for unrelated computer capture or tape sends. Verify Off clears all managed mic/return sends and assignments, retains Volt A1/playback, and never sends transport commands. Parameterize VR input 4 and 5, recording stopped/active/unknown/conflicting, missing numeric readback, and partial failure.

**Confirmed presentation drift.** `internal/tui/dashboard.go:165` and `internal/tui/control_status.go` hard-code the original mic strips. The Stream Deck dependency code additionally accounts for the active VR strip. TUI mute status checks the physical mic, while Stream Deck mute status also checks the Element return. Thus the TUI can clear mute pending before the return is muted. TUI speaker toggling also does not use the deck's composed speaker/fixed-bus toggle behavior.

Share only the necessary pure decisions: managed strip membership, mute target parameters, composed playback mute action, and operation-to-control dependency. Put them at existing routing/control boundaries; keep terminal text, colors, and image layout local. Include all configured managed strips when checking Off, not just the newly effective source, which is absent after Off.

Test identical synthetic states through both presentations: physical muted/return unmuted, manual baseline mute, fixed-bus plus following mute, output migration, VR pre-monitor/capture, and source Off. TUI and deck must agree on requested/pending/observed/unavailable semantics while retaining their different layouts.

## Work package 3 Bound independent workers and disruptive actions

Priority: high. Size: medium; split into independent commits.

**Confirmed Windows-default lifetime defect.** In `internal/control/worker.go:216`, default requests are sent only when a voice intent exists. Reloading a valid non-voice config sends no disabled request. `internal/windowsaudio/defaults_windows.go` retains its last request and reconciles on a 100 ms timer, so a previous enabled/live request can keep enforcing defaults after that profile has been removed.

Always publish explicit disabled policy for absent intent. Replace stale policy updates instead of dropping a newer disable behind an older request. Stop/acknowledge default enforcement before reporting preview or releasing live writer ownership. The current worker releases ownership before the independent Windows worker has acknowledged revocation; source inspection establishes a timing window, but its exact observed effects need a controlled fake-worker reproduction. Keep COM on its own thread. A small request generation and completion signal is enough; no general event bus.

Test enabled to disabled, live to preview, voice to non-voice reload, queued policy updates, cancellation during role correction, and another writer acquiring ownership. After revocation is acknowledged, no new default setter may execute. Bound waiting and report uncertain shutdown rather than claiming completion. A native call already in progress cannot simply be canceled by context.

**Confirmed restart guard inconsistency.** `internal/control/worker.go:309` handles restart without the revision check used by gain/transport/edits. A stale `Confirm` can outlive the state that prompted it. Bind confirmation to the current request/revision and clear it on relevant changes. Re-read transport immediately before dispatch. Reject stale confirmation rather than repurposing it. Preserve one-shot unknown-result handling; do not add automatic retries.

**Hardware queue risk from traced flow.** `internal/desktop/tray_windows.go:187` reports deck errors without clearing `deckQueue`. The queue translates retained raw events against later state in `internal/streamdeck/queue.go`. A disconnect during backlog can leave undispatched intent that executes later, and a delayed Record toggle can change meaning as native transport changes. The decoder suppresses held keys on reconnect, but that does not clear already queued actions.

Reproduce using queued fake events, then bind commands to a device session and preserve one-shot transport meaning at enqueue time. Drop undispatched events on session loss. Never replay uncertain/in-flight transport. Keep ordered setting edits and same-target gain coalescing; verify reverse encoder turns cancel correctly. Distinguish discard due to disconnect from rejection due to a full worker queue.

**Focus delivery risk.** `controls.focus` puts a Focus flag in the same latest-state slot used by `publish`; a following state can overwrite it before the pipe writer consumes it. Preserve focus as a small latched command or separate bounded notification. Do not replace state coalescing with an unbounded FIFO. Test slow rendering plus repeated Open controls.

## Work package 4 Make status accurate and usable at every size

Priority: medium, except truthful mute/Off status in package 2. Size: medium.

**Confirmed diagnostic lifetime issues.** `publish` updates `ObservedAt` even for publications without a fresh successful snapshot. The same timestamp is used for worker-stall detection. Separate last successful observation from worker publication/progress time. Retain the last observation visibly as stale when reads fail; do not imply it is current.

`recovery.observe` returns a nonempty historical restart status before evaluating later A1 health (`internal/control/recovery.go:88`). A successful restart can therefore mask a subsequent stream warning. Keep last action outcome separate from current health. Test recovery success followed by missing sample-rate observation, zero sample rate, read failure, and recovery of observations.

Persistent errors must be state, not only emitted events. The worker clears `state.Error` before each step, while controller events deduplicate identical payloads. Add a repeated-failure regression, particularly the early mute reconciliation error path where the same error can be suppressed. Clear a diagnostic on demonstrated resolution or explicit supersession. Do not remove useful log deduplication to solve UI state.

Replace exact presentation-string decisions such as `Defaults != "Verified"` and `VRMic != "Available"` with small typed statuses plus a reason. Scope this to real state decisions; do not create a universal status framework. Carry queued, saved, verifying, failed, unknown, and fallback meanings consistently across surfaces. Keep per-control status; avoid marking every control failed for one unrelated problem.

**Confirmed compact-view omissions.** `dashboardView` only renders statusRows in the wide layout (`w >= 100`), so narrower supported windows omit health/default/processing reasons and quick help. The smallest-window screen still accepts the normal edit keys despite hiding controls. Long labels can consume the entire row before the value/status appears. Error details are truncated to one line without a dedicated inspection path.

Recommended UI changes:

1. Keep mode, recorder state, observation age, and the highest-priority actionable error visible at all supported sizes. Use a compact status block or a keyboard detail overlay; retain only Controls and Graph tabs.
2. Reserve space for selected values and pending/fallback markers before truncating labels. Wrap selected control explanations where useful. At unsupported sizes, disable hidden edits and expose only resize, help, and close/quit behavior.
3. Show disabled action reasons before activation: preview, recorder unknown, no eligible sources, rehearsal enabled, or routes pending. Revalidate in the worker regardless. Distinguish submitted from verified.
4. Replace the ineffective Auto-recover On/Off experience with a visible unavailable capability and explanation while retaining saved preference compatibility. Do not implement an unvalidated detector.
5. Explain Source Off versus native mute, Pre/Post fallback, and recorder capture versus tape playback in selected-row help. Use configured headset labels while keeping stable IDs internally.
6. Give X reset a concise in-app confirmation describing that all saved choices will reset. On closing with unsent local edits, offer keep editing or discard; do not auto-send transport or delay closing behind a native call. Preserve immediate close when nothing is unsent. These are proposed UX changes requiring the included spec update, not current behavior.
7. Show active config path and reload errors in details. Keep user-facing text concise and technical parameter names in diagnostics. A small bounded diagnostic history is optional only if persistent current details prove insufficient.

Validation sizes: 42x10, 60x20, 80x24, 100x30, and 140x40; also resize below the minimum while a picker is open. Cover NO_COLOR, long Unicode endpoint names, control/escape characters, screen reader/manual keyboard use, disconnected selected devices, and repeated edits while saving. Test visible meanings and bounds, not every decorative character.

## Work package 5 Simplify only proven redundancy

Priority: medium after correctness. Size: small independently reviewable changes.

- Replace the duplicated numeric ActionKind iota list in `internal/tui/worker.go` with named aliases to the authoritative control constants. Preserve current wire values. Remove wrapper aliases only when call sites become clearer. Do not move the worker again.
- Consolidate the shared control decisions from package 2. Do not merge the different TUI and deck queue mechanisms just because both are queues: one tracks absolute drafts and the other relative hardware events.
- Reuse the atomic replacement sequence in `config/state.go` and `config/journal.go` through one narrow private helper if it removes code without weakening state locking, expected-token comparison, fsync, close checking, or temporary cleanup. Keep intent validation separate.
- Investigate duplicated process enumeration: `Client.snapshot` invokes separate SteamVR and Element probes, each enumerating all processes. Read one same-session process snapshot per parameter observation and derive both statuses. Hoist the current-session lookup outside the per-process loop. Measure call counts and elapsed time before/after; do not add stale caches across transaction checks.
- The deck absent-device discovery loop wakes on state updates as well as its timer. Measure idle discovery frequency during rapid numeric updates; keep state consumption responsive but rate-limit device discovery if confirmed excessive.
- Propagate CLI output/JSON encode errors instead of returning success after failed output. Test a failing writer and cancellation. Keep machine output separate from terminal sanitization.
- Remove unreachable source-cycling code only after checking callers; rename `NormalizeRecordingStage` to reflect its current rehearsal-ownership behavior if all references are updated. Keep compatibility fields such as Enabled until an explicit migration warrants removal.
- Apply gofmt and standard import grouping only in touched files. Do not create a mass formatting commit or replace small deterministic scans with registries, caches, or generic frameworks.

Done when each simplification removes duplicate ownership or a demonstrated extra operation, tests preserve behavior, and no new runtime dependency or speculative abstraction is introduced.

## Work package 6 Strengthen validation and release evidence

Priority: start with the first fix and finish before release. Size: medium plus hardware sessions.

Use the existing Go tests and fakes. Add one focused reproducer for each defect, with table-driven variants for meaningful edges. Prefer assertions on native operation sequence, persisted path/content, and published state to implementation-mirroring helper tests. New integration seams are justified only for real worker boundaries currently unreachable through a fake.

Cross-boundary acceptance matrix:

| Flow | Required automated evidence | Separate physical evidence |
| --- | --- | --- |
| Choice to effective route | Picker, validation, save/load, plan, pending, readback agree; stale/save-failed choices do not apply | Volt/webcam/headset disconnect and restoration |
| Mic Off and mute | All configured mic strips; return mute; recorder conflict; playback preserved; no transport writes | Direct/Element tails silent, computer playback/capture uninterrupted |
| TUI plus deck | Concurrent edits, target changes, burst turns, overflow, detach, reconnect; no transport reinterpretation/replay | Actual knobs/keys, layout, HID reports, media behavior |
| Windows defaults | All six roles; ambiguity; partial failure; contention; explicit disable and revocation barrier | Supported Windows setter behavior; SteamVR competition |
| Recorder | Missing values differ from zero; stopped/recording/paused/playing/unknown; preparation race; no uncertain retry | File exists, plays, contains intended Pre/Post/computer mix |
| Desktop lifetime | Focus survives state updates; controls detach leaves owner alive; cancellation is bounded | Tray Quit, Explorer restart, sleep/resume, hidden hotplug |

Add a small `scripts/check.ps1` using existing Go/OpenSpec/build commands. It must fail on native command nonzero exit codes, build to an explicit replacement/scratch path, keep hardware probes opt-in, and report race as skipped when unavailable. Do not silently install CGO or overwrite/stop a running executable. Add CI only if the repository has a chosen remote workflow and runner policy; the portable local check comes first.

Use stdlib fuzzing for the two genuine parser boundaries, config JSON and HID reports, with a few regression seeds and bounded development runs. Do not introduce a framework or blanket coverage target. Prioritize worker lifecycle and Windows/desktop boundaries over raising already-strong routing coverage for its own sake.

Measure input-to-ack and ack-to-verified separately for numeric edits and device changes. Retain existing call-count/latency tests; instrument with existing clocks or bounded diagnostics rather than a telemetry platform. Do not promise 20 ms end-to-end latency or audible continuity from debounce settings.

Complete an isolated `--dry-run` GUI smoke using a copied config and sidecars, because preview still saves choices. Then perform explicitly planned live acceptance with real Volt, headset, Element, recorder, and deck. Record configuration/edition, procedure, observed result, and remaining gap. Never mark these steps complete from mocks or spec validation.

Update README into a short quick start plus links to canonical setup, behavior, troubleshooting, and acceptance docs. Remove historical machine-specific claims and duplicated shortcut/invariant prose. Use scripts/build.ps1 as the desktop build path; document the difference from a plain console-subsystem go build. Track one current binary/release identity instead of relying on an old `-next.exe` reference.

## Implementation order and stop conditions

1. Reconcile specs; land config-path and picker/validation fixes with their regressions.
2. Fix Mic Off execution and shared mute target semantics.
3. Fix Windows policy revocation, then restart confirmation and deck session/transport handling in separate changes.
4. Fix observation/error lifetimes, then compact UI and action explanations.
5. Simplify duplicated definitions and atomic-write mechanics; optimize observation only with measurement.
6. Run the complete validation command, isolated preview, and hardware acceptance; update docs and task evidence.

Each code change must read its applicable existing proposal/spec/design/tasks and update its scope before implementation. Stop expanding a patch when an unrelated defect is found; record it separately. Keep conventional commits. Leave unperformed tasks unchecked. No automatic rollback is proposed: correct fresh-state reconciliation is already the recovery model, and replaying an old mixer snapshot can overwrite legitimate user changes.

The outcome is a smaller set of authoritative decisions, predictable controls, and verifiable behavior across entry points—not more features.
