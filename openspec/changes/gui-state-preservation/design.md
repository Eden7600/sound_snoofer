# Design

## Current behavior and boundaries
`app.ts.receive` updates controls, compares a structural signature and calls `build`, which clears
widgets/updaters and replaces `#content`. It already defers that replacement while an input or
select inside the content has focus and no action is pending. Keep this rule: rebuilding a
focused live select would lose its local keyboard draft and focus-time revision. A later poll,
including a versioned no-change response from `view-state-transport`, must still complete the
deferred rebuild after blur. Telemetry and ordinary values continue updating in place.

This change preserves state only across automatic structural rebuilds of the current screen.
Explicit workspace navigation starts the destination with its normal initial state; no cache
persists across navigation, closing the GUI or host sessions. Provider observations, meters,
slider positions, selection values that already committed and pending/outcome state are not UI
drafts and are never restored from an old DOM snapshot. Keep existing live commit/cancel rules.

## Stable identities and capture/restore
Use a small GUI-local helper around the existing build, not a virtual DOM or generic label-based
serializer. Builders give restorable nodes explicit keys made from existing control IDs, provider
entity IDs and a field/action suffix. Never use display text, array position or a query-selector
ordinal as identity. Keys are unique within a screen; duplicate keys fail a focused test.

Before a same-screen rebuild, capture only registered state: open details, search values, dirty
explicit-commit form fields, selection ranges, focused interactive element and scroll offsets.
Build the new tree and apply current provider state first. Then restore compatible entries,
recompute local filters/validation without input/change dispatch, restore focus with `preventScroll`,
and restore scroll last. Preserve the document scroller and any registered independently scrolling
surface, notably the deck binding list. Clamp offsets when content shrinks. Restoration handlers
must not call request/send, dispatch synthetic events or arm pointer/hold actions.

Restore focus only to the exact same enabled, visible target. If it disappeared or became
unavailable, focus the current screen content (programmatic focus, outside the Tab sequence),
not a neighboring control. Preserve focus outside content, especially an open dialog or sidebar.
Removed targets have no retained entries: a later entity reusing their old position does not
inherit their text, details, selection or focus. Bound stored state to nodes present in the
current screen, with no growing history of departed targets.

### Identity and draft inventory
| Surface | Identity/context | Preserved local state |
| --- | --- | --- |
| Shared semantic widgets | Control ID plus owning editor context | Focus; dirty text for explicit Enter/Apply, including its original edit revision |
| Soundboard | Screen search; clip control ID | Search/filter and focused clip |
| Lights | Current room ID and scene/room control IDs | Other Rooms expansion, focus and scroll |
| App Audio | App control ID plus the name currently used in its edit payload | More expansion, Rename text, Combine choice; Excluded Add form |
| Media | Session control ID plus action suffix | Focus and scroll; observed playback progress is never cached |
| Routing | Microphone/slot ID; for index-addressed lists, configuration hash plus list/index/field | Unapplied name/pattern/Add fields and form choices; live dropdown drafts keep existing deferral |
| Stream Deck | Profile/layout target, page ID, and slot/shared context where applicable; remembered deck serial for device fields | Page Options, page/new-page/deck-name text, binding search/filter/list scroll, focus and local range |

Audit every existing input, select and details builder when implementing; the table defines
categories rather than authorizing new behavior for immediate controls. For example sliders,
checkboxes, Activity thresholds and Recent Window already commit on change; preserve focus
without turning them into saved local drafts. Add-form selections remain tied to their explicit
Add action. Recompute disabled Apply/Add buttons after restoring their fields.

## Draft validity and context changes
Dirty explicit-commit fields retain their text while temporarily unfocused and through unrelated
catalogue changes. A shared text widget must not overwrite that dirty text on the next ordinary
update. Its existing captured edit revision travels with the draft; restoration or refocusing
must not silently capture the new revision. Clear draft state on explicit cancel, or when the
provider observation matches the submitted value. A send/queue acknowledgment alone is not
proof of success; reuse existing error/pending feedback and do not add automatic retries.

Only restore a field when both its stable target and its payload context still match. A deck
page name cannot transfer to another page or layout; remembered-deck text cannot transfer to
another serial; binding search/range and inspector fields cannot transfer to a different
page/layout/slot/shared context. Range indices are valid only for the same model geometry.
App Audio actions currently address names, so a renamed/regrouped app changes the edit context
even if a display position stays the same. An index-addressed Routing field additionally includes
the drawn configuration hash; never move it onto the current entry at that index after reordering.

When context changes or a target disappears, omit the incompatible local draft and show current
provider values; do not submit it, rebase it or preserve it under the next target. Existing focused
Routing edits retain their drawn-hash revision-zero rejection path until they leave the editor.
Local Add forms can survive unrelated catalogue changes only while every selected raw option
still exists with the same meaning. If an option disappears, clear that choice and require a new
selection; never select a neighboring/default device on the user's behalf.

Routing Add forms currently encode catalogue choices as numeric option values. Capture the
meaning of that raw value with the draft and invalidate it if the same number later names a
different driver/input/device. An explicit empty option retains its own meaning: New Output's
“No device” creates a named slot with no device fields, even when connected devices are offered.
Do not coerce that empty value through `Number("")`, which currently selects catalogue entry zero.
Cover empty selection, unchanged meaning, removed options and reused option numbers before the
Routing implementation; a missing choice requires a new selection before Add becomes available.

## Nonblocking confirmations
Use one renderer/controller for the existing styled dialog and a discriminated host/local
confirmation state. Keep the host's text, Cancel / Apply & restart actions and saved-choice
validation unchanged. Local confirmations use action-specific titles/buttons and the existing
warning text exactly:
- Delete Page: “Delete this page from the draft? Save applies the deletion.” This changes only
  the provider draft; Save/Discard behavior remains unchanged.
- Disable Own Layout: “Use the model layout on this deck? Its own layout is deleted.” Enabling
  Own Layout remains an immediate action without this confirmation.
- Forget Deck: “Forget this deck and its own layout?” Its existing persistence behavior remains.

A local ticket contains the captured request ID, operation/value and revision, plus the relevant
profile/page or remembered-deck serial and the invoker's stable focus key. Copy these at opening;
never close over `press(id)`, which looks up a later control revision. Confirmation sends that
captured request once only if the current control is still available, its revision and target
context still match, and the existing pending/reconnecting gates permit sending. If any target
check fails, disable the affirmative action and show “Selection changed; cancel and try again.”
Keep Cancel available. A final check on the affirmative click closes the race before dispatch;
the provider's existing revision validation remains the last guard.

Polling, telemetry, transport asset updates and structural rebuilds continue while the dialog is
open. The dialog is outside `#content`, so rebuild restoration must not steal its focus. Cancel,
Escape or closing a local dialog sends nothing. Default focus is Cancel; Tab is trapped by the
native dialog. Restore focus to its same surviving invoker after close, otherwise screen content.
No queued local dialogs. If a host confirmation arrives, abandon the local ticket without an
action and show the host confirmation; do not automatically send host Cancel. Clear local tickets
on workspace/context departure. Connection loss disables confirmation and never replays it later.

## Verification and sequencing
Implement after transport's shared frontend boundary is stable. Use fresh isolated fixtures;
make a structural change while another entity remains the same, not a telemetry-only update that
does not rebuild. Prove focused deferral and post-blur restoration with a no-change poll too.

First cover generic presentation state and stable focus, then explicit form drafts/context
invalidation, then the dialog conversion. Keep each checkpoint independently reviewable. Add
regressions that fail before each implementation, inspect both window sizes for the dialog, and
run the existing dropdown, deck, Routing, keyboard and accessibility tests after relevant changes.
Full frontend/model/OpenSpec checks are source-only. Canonical build and WebView2 acceptance remain
deferred while the user prohibits interacting with the running host; do not mark them complete.
