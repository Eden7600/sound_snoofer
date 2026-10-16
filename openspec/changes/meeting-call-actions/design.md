# Design

## Focus and actions
Extend the existing Meetings owner with focus and one-shot action delivery; do
not mix Deafen/Share/Leave into mute/camera reconciliation. Providers report action
capabilities and observed values for each exact joined peer. Use the existing
MeetingValue shape and adapter ownership. The core-contract.md pins the API.

Default to the most recently observed new joined call. Host receipt order is the
available ordering, not the browser's foreground tab or an inferred wall-clock
join time. Repeated heartbeats must not steal focus. An explicit focus button pins
that call until it ends/becomes unavailable; then choose the newest remaining
call. A reconnect/replacement is a fresh writable lifetime. Focus is runtime only.
Expose one focus button per call in the GUI and as bindable semantic controls.
Only the focused button is On; its label identifies the app/browser-tab, never
meeting codes, URLs or participants. Shared Mute/Camera remain independent of focus.

Focused Deafen, Share and Leave capture focus revision, exact peer lifetime and
action revision at input. Reject obsolete work before side effects; never fall
through to a replacement or a different app. Changing focus cancels unsent work.
Allow at most one outstanding action of each kind and retain bounded completion
deadlines. No retries after timeout, failure, disconnect or uncertain submission.
Provider callbacks run outside core locks. Shutdown settles pending work and
retains the current bounded cleanup contract.

Deafen requires a known joined supported call and known state. Meet does not
support Deafen and reports unavailable; never substitute playback mute. Correct
Discord's native Deafen control to require joined identity at publication and
execution, and invalidate queued commands across channel leave/change/rejoin.

## Meet presentation and leaving
Reuse the authenticated extension connection and isolated DOM adapter. Add
optional per-call Share and Leave observations. Missing fields in an old helper
mean unavailable. Keep protocol 2 / Meet capability 1 and exact generation,
document, call, control revision, cancellation and result correlation. Advance
the companion version for new behavior. Validate new fields at both boundaries.

Leave clicks the already observed unique joined Leave button once. Confirmation
requires observed departure of that captured call; transport acknowledgement is
not proof of departure. A stale/ambiguous/hidden/disabled button is unavailable.

Share uses supported, inspected page controls to open the normal presentation
workflow or stop the selected call's existing presentation. Never auto-select a
tab/window/screen, bypass browser user activation, grant capture permission, or
declare a chooser/menu opening to be active sharing. A menu-open result is
Submitted with a concise instruction to choose a source in Meet; only observed
presentation state may be On. Capture-source selection stays with the user.
If the supported site cannot open its menu from a companion action, surface that
limit and focus the call for the user's native click; do not fabricate success.
Record real selector/behavior observations before claiming site support.

## UI and layout
Call Controls contains universal Mute/Camera. A focused-call section shows its
focus buttons and Deafen/Share/Leave, with a concise indication of the target.
No focus or unsupported capability means N/A/inert. Preserve per-app diagnostic
readouts and lifecycle gates. Native keys use existing call, deafen, screen-share
and leave icons and palette; do not alter the renderer's general design.

The user's existing Plus XL Meetings Deafen/Share/Leave positions should use the
focused controls; put focus keys into verified unused positions below that row.
Make only this requested personal layout update, after preserving the current
configuration and through the existing save/restart workflow. No smaller deck or
shipped-default changes. Existing explicit per-app bindings remain local.

## Verification
Test newest vs pinned focus, repeated reports, disappearance, stale/ABA commands,
unsupported Deafen, no replay, independent actions, bounded shutdown, and preview.
Test exact Meet command/result identity, optional old fields, malformed inputs,
DOM node replacement, menu vs sharing truth, and observed Leave. Check GUI focus
and native key output, run appropriate Go/race, extension, GUI and strict spec
gates, then the canonical build with exact prior launch restored. Keep unperformed
live site/deck checks unchecked. User reported live Mute/Camera success on 2026-10-15;
this does not by itself establish all multi-call propagation acceptance.
