# Design

## Context

See proposal.md for motivation. The user now runs Potato and has confirmed the Discord microphone test after changing Element to Voicemeeter AUX Virtual ASIO. Volt input 1 is the desk mic; input 2 is always the lav (its battery was dead during diagnosis). The observed webcam is already on input 3. The confirmed Element graph is stereo pass-through, not a validated effects chain.

The current Go planner coordinates A1 ASIO, four input patch cells and playback sends. It puts webcam fallback on input 1, returns early when playback is unavailable and migrates sends for every strip. The API snapshot/allowlist covers A sends but not B sends. The controller verifies unique-target operations serially. The Bubble Tea worker owns DLL calls on a locked OS thread; actions currently support live/dry, reload and refresh. Existing tests use deterministic device snapshots and fake controllers.

No canonical specs have been archived yet. The earlier changes remain historical prerequisites. The optional voice profile introduces explicit exceptions to their fallback and scope constraints; legacy configurations keep those constraints unchanged.

## Goals / Non-Goals

**Goals:** Compile user intent into a deterministic route matrix; make rule state visible and durable; preserve microphone delivery through playback-device churn; extend the existing serialized controller with verifiable transition phases.

**Non-Goals:** Host plugins, edit Element graphs, change Discord settings, detect dead batteries from silence, infer ASIO stream health from an Element process, implement a general routing language, or alter gains/mutes/inserts. Metering and automated host-health failover are separate changes.

## Decisions

### 1. Reserve a separate processing send and return

Use B2 -> Element AUX Virtual ASIO channels 1/2 -> virtual:2 (AUX); B3 is the app-facing capture bus. Primary VAIO (virtual:1) remains app playback. AUX-to-B2 is always off. B1 remains available outside this profile.

| Voice state | Selected mic B2 | Selected mic B3 | AUX B3 |
| --- | --- | --- | --- |
| Disabled / no source | 0 | 0 | 0 |
| Direct | 0 | 1 | 0 |
| Element | 1 | 0 | 1 |

All other strips' B2/B3 sends are zero: enabling this profile explicitly dedicates these buses. This excludes desktop audio from voice capture and prevents accidental additional host sends. The TUI presents these bus reservations before live enforcement.

Alternative: Insert ASIO replaces the input at the insert return and complicates independent pre/post monitoring. A dedicated send/return matches the working setup and keeps both signals available. Do not route Element directly through Volt ASIO, which Voicemeeter owns on A1.

References checked during diagnosis: https://vb-audio.com/Voicemeeter/VoicemeeterPotato_UserManual.pdf documents B2/AUX ASIO correspondence; https://voicemeeter.com/user-guide-connect-audio-apps-to-the-voicemeeter-insert-driver/ describes insert replacement semantics. Windows capture endpoint names vary: the observed B3 endpoint is Voicemeeter Out 8; document the bus identity rather than hard-code an endpoint display name.

### 2. Opt-in profile and separate persistent intent

Add optional studio.voice with validated defaults: enabled=true, source=desk, mode=element, monitor=off. Source enum: desk/lav/webcam; mode: direct/element; monitor: off/pre/post. Defaults reflect this working setup only when the profile is explicitly added. Existing studio.playback_sources define app rules and all start enabled; virtual:2 is rejected because AUX is reserved.

Persist overrides in a versioned adjacent sidecar named <config-filename>.state.json, containing only voice selections and playback enabled flags keyed by semantic source. All plan/apply/watch/tui commands resolve the same sidecar and effective configuration. No live permission, physical bus number, device inventory or inferred fallback is saved. A narrow per-config state-write lock plus expected-content comparison prevents two TUI processes from silently overwriting each other's selections. Save through a flushed same-directory temporary file and atomic replacement before activating intent. Invalid state blocks live writes; provide an explicit Reset saved choices action that replaces it with validated defaults. A failed save preserves old intent. Reload validates the entire config/state pair, rejecting unknown or removed source keys rather than guessing.

Alternative: rewrite config.local.json on every keypress. Rejected because it mixes device matching and interactive selections and can clobber external edits. Dry-run changes may persist intent, but never mixer writes; the UI says so.

### 3. Stable microphone identities

With this profile, inputs 1/2 keep their Volt channel identities and input 3 owns the unique webcam match. On Volt departure, clear patches 0..3 and direct inputs 1/2; the effective source falls back to input 3 while preferred desk/lav remains unchanged. Explicit webcam selection stays webcam when Volt reconnects. Keep the existing ordered regex matcher and ambiguity diagnostics. An unrelated input-3 assignment blocks replacement; no eligible webcam leaves the assignment unchanged but its managed sends off.

This differs from legacy fallback-to-input-1 and is intentional only for the opt-in profile. Preferred source is not a battery-health assertion. Defaults choose desk now; the user can select lav when charged.

### 4. Define owned cells and rule precedence

The voice profile owns every strip's B2/B3 buttons; inputs 1/2/3 and AUX A1..A5 buttons for monitoring; the existing input patch cells; and configured app sources' sends to current/former managed playback outputs and the managed Volt A1. Preserve unrelated hardware assignments, B1, gains, mutes, effects and inserts. Document that Pre means before Element, not necessarily before Voicemeeter's own strip processing.

Exclude inputs 1/2/3 and AUX from generic playback-send migration. Compute their A sends solely from monitoring intent. App enabled flags override migrated state; disabled means owned sends go off, not unmanaged. Post in Direct is inactive with its preference retained. Disabled voice suppresses monitoring. Protected routing constraints are visible locked rows, not optional toggles.

Split microphone/capture resolution from playback resolution: a missing playback candidate must no longer short-circuit voice planning. Clear owned monitor sends while retaining valid capture; do not clear unrelated output assignments. Invalid API snapshots still prohibit all writes.

### 5. Transition phases rather than only a final matrix

Retain a final desired matrix for diff/diagnostics. Build execution phases when a source, mode, patch, or monitored destination changes:

1. Disable and verify prohibited B2/B3 sends, incompatible capture routes and old monitor sends. Gate affected voice sends off before repatching or changing a mic assignment.
2. Perform and verify necessary device assignments and patches using existing inventory checks.
3. Enable the selected processing feed when applicable, then the sole capture source and selected monitor destination.
4. Read back the final matrix and publish convergence.

A target can appear in a disable phase and later enable phase. Replace the current unique-operation assumption with phase-local expected values and per-write preconditions. Recompute preconditions from verified own writes; do not treat own temporary gating as external drift. External mutations, inventory changes or verification failure stop remaining phases. Never enable a new path after a required disable failed. Use the existing debounce/retry/mutex behavior; unchanged state produces no gating or writes. Enabling routing verifies mixer state only, not whether Element is returning audio. A short audible gap during switching is acceptable.

Alternative: submit all final values in an unordered batch. Rejected because dry and processed voice can overlap, or routing can feed back during a partial transition.

### 6. TUI commands and honest status

Add a Rules view before detailed Routing/Devices/Events. Arrow keys select editable rows; Enter cycles source, mode and monitor enums; Space toggles voice/app rows. Keep q, l and r; use a separate refresh key so Space has one clear purpose in Rules. Show keyboard hints per view. Display desired selection, effective source and fallback reason, observed routes, pending/applying/error states, saved-state failures, and an explicit external-audio-not-verified label for Element. A verified matrix is labeled Applied, not Audio working.

Replace the integer-only action with typed commands carrying row identity and desired value. The existing worker serializes save, intent update, controller reset and publication. Reject invalid commands and preserve the single DLL-owning thread. Config reload does not leave an old queued command pointing at a removed rule. Bounded event history records mode/source changes and errors.

## Risks / Trade-offs

- Element can stop returning audio while routes remain correct -> retain explicit Direct recovery; do not silently send an unprocessed voice. Automated host-health detection is outside this contract.
- Speaker monitoring can cause acoustic feedback -> default Off, identify the actual monitor output in the TUI and test monitoring with headphones at low volume.
- AUX may receive unrelated Windows app playback -> document AUX as reserved for Element and verify app playback uses primary VAIO before live acceptance.
- All B2/B3 sends become managed -> show ownership clearly in preview and reject conflicting config rather than silently merge unrelated capture content.
- ASIO/A1 changes can interrupt the audio engine and affect Element -> real-device reconnect acceptance remains required; API readback alone cannot certify resumed audio.
- Persistence and hardware application cannot be atomic together -> save intent first, show pending/partial failures, reconcile on the next valid live pass; never promise rollback.

## Migration Plan

1. Implement and test the optional profile without changing behavior of legacy configs.
2. Back up config.local.json and record the currently working routing matrix before adding the profile. Keep example defaults aligned with desk/Element/monitor Off.
3. Verify Element uses AUX ASIO channels 1/2 at 48 kHz and the existing 512-sample starting buffer; preserve the user's graph. Verify Discord captures B3 and playback enters primary VAIO.
4. Preview the new profile in dry-run, then run the real-device acceptance tasks. The already reported Discord test is baseline evidence, not completion of the new feature tests.
5. On rollback, stop live enforcement and restore the captured mixer settings and config backup explicitly; removing the profile alone does not undo writes. Preserve the sidecar for diagnosis.
6. Archive prerequisite changes when their own outstanding acceptance work is complete, then this change. Do not mark tasks complete based only on artifact validation.
