## Context and scope

The interview is complete. This design records its decisions and supplies concrete implementation defaults for edge cases. All implementation and builds remain in the existing sound_snoofer repository. The application becomes Snoofer; core plus audio preserves Sound Snoofer's practical function.

Current composition lives in desktop tray startup and standalone TUI startup. Both reach an audio-specific control worker; the tray also starts hardware integrations. Stream Deck rendering and IPC depend on audio State/Action types. VR discovery is partly embedded in native process observation and routing. A nil Stream Deck configuration currently means defaults, not disabled. These are the seams to change.

Retain the existing audio worker's serialized, OS-thread-affine ownership; deterministic routing and controller apply/verify separation are assets, not candidates for a framework rewrite.

## 1. Ownership and build composition

| Component | Owns | Required plugin dependencies |
| --- | --- | --- |
| Core | Tray, TUI shell, config envelope, lifecycle, diagnostics, semantic registry, private UI transport | None |
| Audio | Voicemeeter, routing, recorder, gains/mute, health recovery, Windows default protection, audio settings UI | None |
| VR | SteamVR observation, VR profile settings, activation policy | Audio |
| Stream Deck | HID lifetime, input, rendering, user pages and configurator | None |
| Windows media | Media commands and supported media state | None |

Core must start with no plugins, no Voicemeeter and no native audio companion. It displays configuration and diagnostics. It does not know microphone, recorder or headset types.

Use explicit Go composition, not runtime discovery. A small public plugin contract describes ID, label, dependencies and a start function yielding a stoppable instance. Keep external author-facing contracts outside Go internal packages. Built-in factories needed by an external composition root must also be importable. Document an external Go module using a local replace directive and explicit registration; rebuilding is expected, Go API compatibility across arbitrary versions is not promised.

Static descriptors and factories may be registered without running plugin code. Imports must not open devices, load DLLs, start goroutines or register global callbacks in init functions. Disabling cannot remove binary size or inert metadata, but must remove runtime plugin work. Build selection also allows plugins and their native build prerequisites to be omitted entirely.

Expose only real shared needs. The host passes declared dependency instances to construction; VR uses a narrow audio profile-policy API, not access to the controller or DLL. Undeclared dependencies and import cycles are rejected by composition review and lifecycle validation. Do not create a service locator, plugin bus, generic rules engine, sandbox or version-negotiation protocol.

## 2. Lifecycle and failure

Validate unique IDs, missing dependencies and cycles before starting the affected graph. Mark invalid nodes and dependents unavailable with precise reasons; retain the core and unrelated valid branches. Start dependencies first, stop dependents first. Each plugin owns its goroutines, timers, subscriptions and native handles, including cleanup after partial startup failure.

Disabled plugins are never constructed, started or invoked for config validation. The core may read their inert descriptors and preserve their config as opaque JSON. Enabled configuration is validated before that plugin starts. Missing physical hardware is normally a disconnected plugin state with its existing bounded reconnect behavior, not failure of the host.

An ordinary start error disables that branch for the session and offers explicit Retry. Retry starts the necessary failed branch in dependency order, without duplicating already healthy instances or replaying actions. Trusted in-process plugins do not provide crash containment: an unrecovered panic can stop the application.

Enablement UI presents the full dependency closure before applying: enabling VR also enables audio; disabling audio also disables VR. One confirmed operation persists the complete change atomically, then stops and restarts Snoofer. A failed save leaves the running selection unchanged. Ordinary settings do not restart the host.

For whole-application stop/restart, quiesce new actions and policy transitions first, then stop in reverse dependency order. Release callbacks before writer ownership, as existing native safety requires. Do not reset Voicemeeter routes, stop the recorder or restore old Windows defaults. Clear deck displays when reachable. Disabling only VR still uses whole-app restart: shutdown leaves routing alone, the next audio startup reconciles Normal because no VR policy is installed.

Shutdown waits are bounded with explicit diagnostics. A plugin that cannot confirm native cleanup must not trigger a second overlapping writer through automatic relaunch. Preserve existing native module retention on uncertain callback teardown; report restart failure and require recovery of the old process before relaunch. Never release a lock as proof that an outstanding native callback has stopped.

## 3. Configuration and saved state

Use one core envelope containing format version and plugin entries keyed by stable ID, each with enabled and settings. Keep per-plugin settings as raw JSON until the enabled owner validates them. Reject malformed JSON, duplicate keys and unknown envelope fields. Enabled plugins retain strict validation of their own fields; disabled or uncompiled settings are preserved without executing their code. Uncompiled enabled entries are shown as unavailable, never silently enabled as a substitute implementation.

Reuse atomic persistence and stale revision checks. Domain-owned saved choices and operational safety state remain distinguishable from static settings. Avoid a generic schema language: existing Go validators and TUI components are sufficient. Plugin-owned settings sections render within the common TUI; do not launch a separate UI per plugin.

The hard changeover manually converts the owner's configuration and saved choices. Retain Normal source/playback overrides and priority order, recording preferences, media/deck mappings, recovery budgets and uncertain-command markers, and native mute/ownership journals. Never reset safety limits merely because executable/config names changed. Inactive profile edits save without native writes until that profile becomes effective.

## 4. Semantic controls and transport

Providers register stable namespaced control IDs, label, kind, compatible input operations, state, availability and optional group/icon hints. Support command, toggle, selection, numeric adjustment and read-only status with only the fields needed by current controls. A binding targets one semantic control; a dial may use that control's declared rotate/press operations. No multi-action macros.

Core owns registration and transport, providers own meaning and authoritative state, and each surface owns rendering. Audio-specific domain models stay in audio. Reuse private inherited-pipe desktop transport and existing origin/revision concepts; no network service or JSON-RPC framework is needed.

Requests include identity/revision sufficient to reject stale actions after provider replacement, configuration changes or page changes. Preserve bounded queues and explicit requested/pending/observed/unknown/failed states. Commands are one-shot and never replay on retry, reconnect or provider restoration. Provider removal makes controls unavailable and invalidates queued work.

Stream Deck access to another plugin's semantic controls uses the core surface service; this is not a direct dependency on every bound provider. Direct domain-to-domain calls still require a declared dependency. Thus audio can be absent while the deck controls media, and media can be absent without breaking audio.

Keep the streamlined TUI: no generic Actions section. Retain the explicit System confirmation for disruptive engine restart requested from other surfaces. Do not infer successful native changes from dispatch alone.

## 5. Audio and VR profiles

Audio owns both routing execution and the profile-resolution mechanism. VR owns SteamVR detection and contributes its profile policy through the declared dependency. Audio and core do not enumerate SteamVR processes when VR is disabled.

SteamVR means exact current-session vrserver.exe running, not headset wear or endpoint health. Known running selects VR; known stopped selects Normal. Unknown observations do not invent a transition: retain the last known profile and mark detection stale; at startup before a known observation, remain Normal with unknown VR status. This explicitly replaces the old unknown-means-exclude-headsets policy.

Normal and VR have separate microphone and playback priority lists and saved runtime overrides, plus separate processing mode and monitoring preferences. Preserve today's preconfigured ordered lists plus runtime override model; do not add a general-purpose runtime priority-list editor. Automatic clears an override. Mute, gain and recording preferences are shared, so profile changes cannot unmute or start recording.

VR wins while SteamVR runs, even over a saved Normal override. Normal Off is a Normal-profile choice; a deliberate shared mute remains effective in either profile. Off in the effective profile preserves all current microphone-disconnect invariants without stopping playback or recorder transport.

An optional final VR list entry delegates that direction's source resolution to Normal. It must be last and cannot recurse. It uses Normal's override/priority resolution for that direction; VR processing/monitoring preferences still apply. Without this entry there is no implicit Normal fallback. Resolve microphone and playback independently. Preserve current explicit-override unavailable/fallback semantics and show requested versus effective selection; never overwrite the saved override during disconnect.

Use actual presence and unique endpoint matches. Installed ASIO alone is insufficient. Ambiguity excludes the ambiguous candidate with a reason rather than selecting arbitrarily. Re-evaluate hotplug, debounce and verify native assignments; preserve valid identity on equal priority and avoid source reassignment for gain/status changes. Returning to Normal restores its saved choices, subject to current availability.

One audio view has separate Normal microphone, Normal playback, VR microphone and VR playback sections, with processing/monitoring placed with their relevant controls. While VR overrides, Normal sections are subdued but remain focusable and editable, with per-section text explaining when edits apply. Shared controls remain visibly active. Do not add a combined selector. Hide active VR controls when the VR plugin is disabled; plugin management still shows its inert metadata.

Initial semantic source controls operate on the active profile and identify it in their state. Fixed-profile deck variants are deferred. Native routing, Volt A1/channel patches, Element loop prevention, recorder state, Windows default correction limits and callback-based recovery remain in audio with existing tests and safeguards. Callback progress still proves processing, not audible output.

## 6. Stream Deck layout and configurator

Support the current Stream Deck + XL protocol: 36 keys and 6 encoders. Keep existing serial-specific selection and hardware discovery behavior within the enabled plugin.

A page describes keys and the first five dials. The sixth/last physical dial (zero-based index 5) is globally reserved: rotation wraps pages, press returns Home, display shows current page name. Maintain at least one page and a valid Home page; deleting Home selects the first remaining page. Deleting the final page is rejected. Page IDs remain stable when renamed/reordered.

Shared bindings reserve positions across every page. A page cannot shadow a shared position or the navigation dial. The configurator rejects collisions with a clear explanation instead of silently discarding bindings. One control per binding; blank is supported.

Integrated TUI operations: create, rename, reorder and delete pages; select Home; assign and clear compatible key/dial bindings; assign shared bindings; preview the effective merged page before saving. Editing has Save/Cancel and applies a validated snapshot atomically and live. Save failure retains the prior active layout.

Seed a useful initial layout from the owner's existing bindings and first three gain dials, using Studio icons and live text. The default layout does not take over subsequent user edits. No drag-and-drop, artwork importer, folders, application-specific profiles or macros.

Bindings retain provider/control ID and saved display label even when the provider is disabled, omitted or failed. Render dim Unavailable and ignore input; configurator explains the reason. Restore automatically when the same stable control returns, without replaying old input.

Page transitions/layout edits invalidate queued input associated with the previous layout generation. Never reinterpret a held key or old encoder delta as a new page action. Preserve initial-held-key suppression on reconnect, bound rendering queues, send changed images only, and redraw the current effective layout after reconnect. Hardware absence permits offline layout editing. Unknown active page after an edit resolves to Home.

## 7. Delivery sequence and acceptance

Implement in dependency-ordered work packages in tasks.md. First introduce the host and isolate audio without changing routing behavior; then extract VR/media and genericize surfaces; then add profiles and the configurator; finally perform personal changeover and hardware acceptance. Keep intermediate builds usable rather than maintaining permanent legacy and plugin paths.

Build scripts select composition explicitly. Default build includes the four built-ins and the audio native companion. A core-only build must neither compile nor require the companion or Voicemeeter SDK/toolchain. Do not redistribute the vendor DLL. Preserve tray-first GUI launch without a blank console, controls child lifetime and single-writer ownership during rename.

Automated tests cover lifecycle/config/control seams and deterministic profile/layout behavior; preserve current routing/controller/native tests. Real-device checks separately cover Volt stall/reconnect/recovery, VR transitions, deck controls, recorder continuity and clean exit. Race checking is conditional on the already installed supported toolchain; report skipped checks, do not silently install tools.

Before conversion, back up personal config/state plus matching prior executable and native companion inside the repo. Record exact build and rollback commands. No automatic migration, compatibility aliases, Hue implementation or speculative plugin ecosystem. Completion means verified working configured behavior, not merely packages moved.

## Implementation details

The audio plugin retains a passive ownership journal for the VR hardware input and playback patterns. This is cleanup metadata, not VR activation: disabled VR does no observation. Audio retains those ownership patterns after VR stops or is disabled, without adding them to Normal priorities. A hardware-input ownership change requires explicit cleanup of the prior strip before replacing the journal. Maintenance commands remain plugin-owned; the interactive desktop uses the unified core UI.

## Post-implementation cleanup

Remove the unreachable standalone internal/tui package and its CLI launch adapter now that app owns the only interactive UI. Move audio-worker regression checks out of that package into internal/control; discard tests that target the deleted UI. Audio maintenance retains devices/plan/apply/watch and directs interactive requests to the tray. Remove obsolete loose executables and test/preview outputs from bin; keep the current running build, personal state, diagnostic source/evidence and the verified rollback bundle. Commit the accumulated implementation, hardening and callback work together; generated local artifacts remain ignored.

## Restored Bubble Tea presentation

Keep the installed Charm Bubble Tea v2 runtime and existing ANSI/colorprofile dependencies. Restore the previous calm blue section headers, padded two-column controls, strong selected-row highlight, active tab treatment, bordered viewport and persistent keyboard/status footer. Override sections remain subdued but visibly selectable. Pickers use the same frame and retain identity/revision validation. Keep the complete view within actual terminal dimensions, including small-terminal fallback; respect NO_COLOR and strip terminal control sequences from plugin text. This changes presentation, not audio or plugin actions.
