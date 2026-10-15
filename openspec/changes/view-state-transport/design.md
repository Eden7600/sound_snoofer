# Design

## Current flow and scope
The tray builds `ViewState` every 200 ms. `controls.publish` encodes it into a one-slot inherited-pipe queue; `Desktop.State()` returns the last snapshot on every browser poll. Artwork lives in each control, while `Animation` is excluded from JSON and uses a complete-map side channel. The current animation sender compares only the first artwork identity, and `omitempty` drops its final empty-map clear.

Keep this flow, full control snapshots, and the existing queue. Do not introduce per-control deltas, hashes, an asset server, event-driven rendering or registry revision changes. Comparing only control revisions loses telemetry. Repeatedly encoding full artwork just to compare snapshots retains much of the original cost.

## Preparing and publishing frames
Replace the animation-only sender bookkeeping with session-local presentation bookkeeping in the transport package:
- Copy the `Controls` slice before stripping `Artwork` from its control values. Build a complete nonempty-artwork map keyed by control ID and a complete animation map for controls with more than one frame. Do not mutate provider or registry snapshots; immutable strings/frame slices may be shared for reading.
- Encode only the stripped `ViewState` for comparison. Its JSON includes all presented fields, including meter/progress values and timestamps, provider `ViewData`, plugin labels/status/enablement, notices and confirmation. Compare bytes, artwork-map values and complete animation-frame contents/delays against the last successfully prepared frame.
- When none changed, `publish` enqueues nothing. Otherwise increment a positive session `Version`, include the latest complete stripped `State`, and include only changed complete artwork/animation maps. Initial publication includes both maps, including empty ones.
- Commit the comparison baselines and version only after encoding succeeds, so a malformed provider payload cannot suppress the next valid asset update. Keep existing contextual encoding errors.
- `focus` uses the same preparation path with forced emission. Unchanged focus carries the current version and stripped state; a newer snapshot is prepared normally, including changed maps. Focusing must not advance the version merely because the window is shown.

Use optional pointer-to-map fields for `Artwork` and `Animations`: omitted means retain the cache, `{}` means replace it with an empty cache. Full maps make removal unambiguous and keep cache size bounded by the latest snapshot. Session versions are JSON numbers, separate from control revisions; at the existing cadence their exact JavaScript integer range greatly exceeds any process lifetime.

## Queue replacement and child lifetime
Keep the existing byte queue and merge logic, extending it to artwork. Every publisher frame contains the complete latest state and version. When replacing a queued frame, keep the newest state/version, OR the focus flag, and inherit each older asset map only when the newer frame omits that map. An explicit newer empty map overrides an older populated map. This also handles a focus frame replacing queued changed state because focus prepares the latest snapshot through the same sender.

The comparison baseline describes frames accepted by this session's queue, not pipe acknowledgments. Map inheritance makes that safe when a queued frame is superseded. Pipe failure still detaches the child; a new child gets new bookkeeping and a full initial state plus asset maps. Nothing is persisted and no actions are replayed.

## Desktop cache and polling API
The desktop atomically replaces state and supplied maps under its existing mutex. Store the current presentation version, the version of the latest artwork-map replacement, and the version of the latest animation-map replacement. If queue merging attaches an earlier map to a newer frame, marking that replacement with the newer version is safe and may only cause an additional complete-map response.

Change the private Wails API to `State(sinceVersion)`, returning an optional response with `Version`, `State`, `AnimationVersion` and optional `Artwork`:
- Before the first snapshot, return no response.
- A nonzero caller version equal to the current version returns no response (`null`), including after an unchanged focus frame.
- Otherwise return the latest complete stripped state and animation generation. Include the complete artwork map when the caller passes zero, predates its replacement, or supplies a version greater than this session's current version. An empty map must survive Wails JSON serialization.
- Keep `Animation(id)` as the on-demand frame lookup; animations never enter polled state.

The browser starts with version zero and advances its cursor only after accepting the response into its local cache. New browser contexts therefore request all artwork even when the child already has state. There is no server-side per-browser acknowledgment or retained history.

Keep accepting unversioned `{State: ...}` pipe frames used by the existing native smoke script. Treat these as complete legacy snapshots: extract their inline artwork into the desktop's complete artwork cache, copy/strip their controls, and assign a fresh local version; apply an explicitly supplied animation map normally. Normal tray frames always have a positive version. No legacy JavaScript API is needed: the smoke script uses the bundled GUI and does not call `State()` itself.

## Frontend integration
Keep version/artwork caching and reconstruction in the shared host/polling boundary. On a changed response, replace an explicitly supplied artwork map and populate copied control objects' `Artwork` from that cache; absent IDs become empty artwork. Screens continue to consume the existing `Control` shape. Hydration must not mutate the stripped cached response or fixture objects.

The regular poll continues calling the existing `receive` path with hydrated state or `null`. A no-change poll still updates progress clocks, timer countdowns, meter expiry/peak decay, pending-action timeouts and the current deferred structural rebuild after input blur. Preserve polling cadence, error reporting and `finally` scheduling.

Expose the returned animation generation to shared artwork widgets. When it changes, animated images re-fetch `Animation(id)` even if their first still image is unchanged; static images do not restart. Check both artwork identity and captured generation after an async animation lookup and during playback so old results/timers cannot overwrite new or removed animations. A global animation generation may restart other visible animated images on a rare animation-set change; this avoids per-control invalidation bookkeeping. Telemetry-only updates never change this generation or restart animation playback.

## Configuration and invariants
Do not cache configuration or login-startup reads in this change. External configuration edits, invalid-file notices, enabled selections, confirmation construction, reload-before-action and stale-save validation retain their current behavior. This fixes repeated transport/asset serialization, not all tray snapshot construction cost. No action policy, user-visible layout, draft preservation, audio routing or provider schema changes are included.

## Verification
- Go tests exercise unchanged suppression; meter/progress-only changes with stable revisions; other state fields; asset add/change/remove; empty-map JSON round trips; queue coalescing/reversion/focus; initial/new-child state; encoding failure recovery; provider immutability; desktop no-change and late/reopened caller responses; complete animation changes; legacy frames.
- Browser tests use the versioned fake host, with an explicit frozen-telemetry mode so unchanged behavior is tested rather than hidden by its current automatic meter timestamp refresh. Verify artwork replacement/removal and cached navigation, local clocks/expiry/deferred blur on null responses, stale animation lookup rejection, unchanged action requests, and continued polling after errors. Preserve existing fixture scenarios and animation tests.
- A focused Go benchmark reports bytes and allocations for 60 controls with 64 KiB encoded artwork each: initial publication, repeated unchanged snapshots, telemetry-only changes, and artwork replacement. Compare actual pipe and Wails payload sizes with the old full-state encoding. Assert that unchanged publication emits no frame, unchanged IPC returns null, and steady telemetry contains no artwork bytes; report measurements without a machine-specific timing gate.
- Run Go tests/vet, model tests, TypeScript/lint/format checks, GUI tests, strict OpenSpec validation and the coordinated canonical build/native smoke. Do not alter the user-owned native smoke script or create another application output. Hardware-only visual acceptance remains explicit if unperformed.
