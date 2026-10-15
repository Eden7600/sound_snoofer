## ADDED Requirements

### Requirement: GUI actions receive bounded correlated results
Each GUI action SHALL carry a unique session-local string ID and receive at most one terminal
result from Desktop.Send for that ID. The desktop SHALL reserve at most sixteen pending results,
explicitly refuse full queues or duplicate outstanding IDs, and continue serving state while
results are pending. Results SHALL use a noncoalescing lane separate from presentation state.

#### Scenario: Results arrive out of order during state updates
- **WHEN** two accepted actions complete in the opposite order while presentation frames replace one another
- **THEN** each Send resolves only with its own result, and neither result is lost or changes presentation version
- **AND** a duplicate result cannot resolve another action

#### Scenario: Pending capacity and downstream queues fill
- **WHEN** sixteen desktop requests remain unresolved, or the desktop, tray or provider input queue is full
- **THEN** another request receives an explicit Controls busy or provider-busy rejection before its operation begins
- **AND** no accepted result is silently dropped, and State/Animation remain responsive

#### Scenario: A child closes before completion
- **WHEN** a submitted action remains outstanding when the child, pipe or host session closes
- **THEN** the waiting call settles Unknown and session resources are released
- **AND** a later provider completion cannot reach a reopened child or cause the action to be resent

### Requirement: Results state only the established completion boundary
Results SHALL distinguish Submitted, Processed, Rejected, Failed and Unknown as defined in the
design. An enqueue acknowledgment, setter return, unrelated revision/notice change, timeout or
connection recovery SHALL NOT be presented as observed success. Provider readback/status SHALL
remain separate. A dispatch without an explicitly claimed completion SHALL return Submitted.

#### Scenario: A save finishes before native readback
- **WHEN** a requested preference saves successfully but its native state is still unconfirmed
- **THEN** the action reports Processed/Saved and the control retains its existing pending or unknown native status
- **AND** the GUI does not label the preference applied solely from that result

#### Scenario: A provider only accepts coalesced or remote work
- **WHEN** Hue accumulated brightness, a deferred media seek, browser media command, AEC retry or Discord Connect is accepted at its documented boundary
- **THEN** the action reports Submitted without claiming each intermediate value was written or observed
- **AND** subsequent native/readback health remains visible through existing provider status

#### Scenario: A partial native write fails
- **WHEN** a captured finite native operation returns an error after some effects may have occurred
- **THEN** the action reports that processing error without claiming nothing happened
- **AND** neither failure nor uncertainty triggers automatic replay

### Requirement: Claimed provider completions settle on every path
A provider claiming completion SHALL return worker validation, save and bounded native/RPC errors
to the originating action and settle exactly once on completion, cancellation, replacement or
stop. It SHALL use explicit error/result flow rather than parse presentation strings. The provider
matrix in the design defines the required boundaries; status-only providers need no adapter.

#### Scenario: An audio action fails after enqueueing
- **WHEN** the audio worker rejects a queued action as stale, unavailable or invalid, or its save/reload fails
- **THEN** that action receives the explicit error rather than an enqueue acknowledgment or another origin's acknowledgment
- **AND** existing saved-choice validation, origin notice clearing and scoped feedback remain intact

The original non-GUI acknowledgement retention is superseded by the user's 2026-10-15
`audit-completion` instruction; callback completion and GUI result identity remain unchanged.

#### Scenario: Async state updates are discarded
- **WHEN** an accepted Hue/Discord/Soundboard operation is cancelled, superseded, stopped or returns after its provider generation changed
- **THEN** its claim is settled exactly once even if the normal state-update closure is discarded
- **AND** a late callback neither overwrites the settled result nor starts an obsolete operation

#### Scenario: Operation-specific failures remain attributable
- **WHEN** Stream Deck or provider settings save fails, an App Audio session write fails, Windows media injection fails, a Hue request fails, Discord returns an RPC error, or Soundboard preparation/start fails
- **THEN** the corresponding claimed GUI action receives that error at the boundary defined in the design
- **AND** successful submission of another action cannot erase or absorb it

### Requirement: Queued actions retain their captured target
Registry revision, availability and operation checks SHALL remain in force. Provider queues SHALL
retain the published semantic target and necessary context/lifetime and reject changes at dequeue
before side effects. They SHALL NOT resolve stale input against newer focus, room, page, session,
channel or device identity. Existing intentional same-target accumulators MAY remain.

#### Scenario: Focus or a slot changes before dequeue
- **WHEN** a media/App Audio focus target, Hue room-scene slot, Discord call context or deck editor target changes after input was captured
- **THEN** the queued request is explicitly rejected and does not affect the replacement target
- **AND** resolving a new revision or current target is not used as a silent retry

#### Scenario: A target disappears and returns
- **WHEN** a target becomes unavailable or is recreated before a queued action executes
- **THEN** its old lifetime request is rejected even if its display label or position is reused
- **AND** partial or ambiguous work already submitted is reported honestly rather than replayed

#### Scenario: Discord only exposes a toggle RPC
- **WHEN** a queued Camera or Screen Share command reaches its captured connection and channel lifetime
- **THEN** a known captured state retains its desired transition and a changed observed state is rejected before toggling
- **AND** an unknown captured state remains usable only while still unknown, reporting the RPC response boundary without claiming a desired or observed value
- **AND** leaving and returning to the same channel invalidates the old command and uncertain results never trigger a replay

### Requirement: Now Playing commands identify source lifetimes
Now Playing SHALL preserve public control IDs while carrying private native-session,
bridge-connection and browser-frame lifetime identities from publication through dispatch.
Enumeration IDs, browser names and reused tab/frame IDs SHALL NOT substitute for those identities.
Source replacement or missing identity SHALL refuse queued work before a known side effect.
Captured playback and tab-mute toggles SHALL carry explicit desired values rather than invert a
newer state. This requirement adds no automatic retry or browser command acknowledgment protocol.

#### Scenario: Windows sessions reorder or disappear
- **WHEN** sessions with the same app identifier reorder, disappear/reappear, or their source handle closes and reopens before queued work executes
- **THEN** native instance tokens retain the same canonical COM identity only while its reference belongs to the current successful snapshot, and removed/recreated identities get new tokens
- **AND** a queued command either reaches its captured instance or is rejected, never the current occupant of an enumeration ID
- **AND** snapshot buffer retries preserve the committed session table until delivery succeeds, and removal/close releases retained references

#### Scenario: A browser connection is replaced
- **WHEN** a browser reconnects while old reports, disconnects or commands remain queued
- **THEN** the worker ignores obsolete reports, an old disconnect cannot remove the new connection, and a command cannot be written to a replacement connection
- **AND** callbacks from a replaced extension WebSocket cannot act for the current socket

#### Scenario: A browser document replaces the same tab and frame
- **WHEN** a new content-port instance occupies the same tab/frame before a captured command reaches extension routing
- **THEN** the extension refuses the mismatched instance and an old port cannot overwrite the new frame's report
- **AND** captured play/pause and mute values are applied explicitly when the identity remains valid
- **AND** tab mute remains a browser-tab API operation checked at dispatch, without claiming atomicity against later navigation or observed success

#### Scenario: Deferred seek outlives its target
- **WHEN** a dial seek is accepted into the accumulator and its captured source/session lifetime changes before the deferred write
- **THEN** that obsolete seek is discarded without selecting a replacement target or replaying it
- **AND** its earlier Submitted result remains distinct from later native/readback status

#### Scenario: An older companion or extension lacks identity support
- **WHEN** a native snapshot lacks an instance token or the browser protocol version does not match the identity-bearing format
- **THEN** native session commands are unavailable or the existing extension handshake explicitly requests the appropriate update
- **AND** no compatibility fallback dispatches by a mutable display ID

### Requirement: The two global mute preferences can be independently ordered
Only audio.mic-mute and audio.speaker-mute SHALL use separate worker-owned preference generations
instead of the shared audio epoch. A request SHALL capture its generation and explicit desired
boolean. The worker MAY accept it after an unrelated overall revision change only while that
preference generation/lifetime matches. Registry revisions and SaveIntent StateToken CAS SHALL
remain unchanged; every other audio action SHALL retain its existing strict revision/target policy.

#### Scenario: Both mute preferences are requested rapidly
- **WHEN** valid Mic Mute and Playback Mute requests from the same displayed snapshot reach the worker in either order
- **THEN** both desired booleans are saved while preserving the first accepted edit
- **AND** each action receives its own result, with native observation shown separately

#### Scenario: Same preference changes away and back
- **WHEN** a captured mute preference changes false to true and back before its queued request executes
- **THEN** the old request is rejected by its generation despite equal boolean values
- **AND** other callers advance that preference generation when they change it

#### Scenario: Intent is reloaded, reset or replaced
- **WHEN** a reload/reset boundary or intent lifetime change occurs after a mute request was captured
- **THEN** both mute generations are invalidated, even when the resulting booleans are unchanged
- **AND** an old queued request cannot cross the boundary

#### Scenario: Saved choices change externally
- **WHEN** another writer changes the saved-choice StateToken before a mute edit saves
- **THEN** the existing CAS refuses the save and the originating action receives the error
- **AND** no file overwrite, forced reload/rebase or replay bypasses the check

#### Scenario: A contextual audio target changes
- **WHEN** a queued gain, routing, recorder, restart, gain-press alias or linked-mute action has an obsolete overall revision or physical/context target
- **THEN** its existing strict rejection remains in force
- **AND** the two-preference exception does not broaden into arbitrary commuting audio edits

### Requirement: Pending actions and feedback stay local
The GUI SHALL track pending state by action and busy key, allow unrelated controls concurrently,
and explicitly refuse another input for a busy key. Feedback SHALL remain attached to its origin
control and screen across navigation. Unrelated state or result changes SHALL NOT clear it.
Navigation, polling, local telemetry, focused editor deferral and connection recovery SHALL continue.

#### Scenario: Another screen is open when an action fails
- **WHEN** an action completes with an error after the user navigates elsewhere
- **THEN** its error is retained for the originating screen without appearing as that other screen's error
- **AND** returning to the origin shows the unresolved feedback; dismissing it affects only that feedback

#### Scenario: A different action remains usable
- **WHEN** one control is waiting for completion
- **THEN** only its busy key and mirrors are busy, and an unrelated available control can submit
- **AND** a repeated attempt for the busy key is explicit rather than silently dropped

#### Scenario: The display deadline expires
- **WHEN** an action has no result after 2.5 seconds, including during null state polls
- **THEN** it shows Outcome unknown while keeping its Promise and busy reservation until completion or disconnection
- **AND** a late result updates only that action and no timer, navigation or recovery resends it

#### Scenario: A state revision or notice changes
- **WHEN** controls, notices or confirmation state change while an action is outstanding
- **THEN** that change alone neither clears pending nor establishes success
- **AND** unrelated pending work does not force a focused input/dropdown to rebuild or recapture its revision

### Requirement: Lifecycle proposals and confirmations identify their intent
Enablement selection SHALL capture the displayed configuration revision. A host proposal SHALL
have a unique identity, owner session, source action and saved revision. Another action in its
lifecycle group SHALL receive explicit busy feedback. Confirm/Cancel SHALL identify and consume
only that proposal and preserve existing stale-save checks. Local confirmations SHALL retain their
captured request and target rather than dispatch a fresh current-control lookup.

#### Scenario: An old confirmation or cancellation arrives
- **WHEN** a request names a missing, consumed, replaced or differently owned proposal
- **THEN** it is rejected and cannot save or cancel another proposal
- **AND** closing the owning GUI abandons its unconfirmed proposal without applying it

#### Scenario: Configuration changes before selection or confirmation
- **WHEN** the config revision differs from the displayed selection or the proposal's saved revision
- **THEN** the respective request receives a stale error without overwriting the external edit or rebasing the user's choice
- **AND** no restart is initiated for the rejected request

#### Scenario: Confirmation saves and initiates restart
- **WHEN** the exact current proposal is confirmed and its save succeeds
- **THEN** the host can report Saved; restart requested without claiming restarted health
- **AND** if shutdown prevents delivery, the original wait becomes Unknown without blocking shutdown or replaying confirmation

#### Scenario: A local confirmation is cancelled or becomes stale
- **WHEN** a local dialog is cancelled or its captured revision/target becomes invalid before confirmation
- **THEN** cancellation sends nothing and a stale affirmative action is refused
- **AND** polling continues and no newer target is substituted

### Requirement: Verification preserves integration and no-replay invariants
Automated tests SHALL use isolated temporary state and fake integrations, cover each new completion
owner's shutdown/replacement paths, and retain the existing revision, no-replay, keyboard and
accessibility coverage. Source-only checks SHALL NOT be reported as native acceptance.

#### Scenario: Legacy native fixture has no result producer
- **WHEN** an unversioned fixture state is supplied and an action awaits a result that fixture does not emit
- **THEN** the controls still render and emit the captured revisioned action, and EOF/window close releases the wait
- **AND** the existing user-owned desktop smoke script remains unchanged

#### Scenario: Native acceptance is not yet performed
- **WHEN** source checks pass before the coordinated canonical build and native/live verification
- **THEN** those isolated checks have not changed deployed binaries or personal configuration
- **AND** canonical packaging and real-device acceptance remain explicitly unchecked until actually performed
