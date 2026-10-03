# Proposal

## Why

The annotated dashboard screenshot identifies wasted header/footer space, verbose
status text, a source control that cycles instead of showing choices, and recording
actions mixed into settings. Make these interactions compact and explicit.

## What Changes

- Remove the standalone app-title row; retain live/preview, connection, recorder
  status and view tabs. Remove idle instructional filler and the second footer row.
- Replace verbose routine notices with concise, accurate status. Keep actionable
  errors visible and full diagnostics in Events.
- Activating Source opens a selectable list of Desk, Lav, Webcam and Off;
  navigating the list does not change audio before confirmation.
- Remove the dedicated `0` mic-Off shortcut and all references to it.
- Order recording settings as Record Computer Audio, Record Microphone,
  Recording Mic Stage. Display stage values as Pre or Post.
- Separate Start Recording and Stop Recording from settings with a nonselectable
  spacer and an Actions heading.
- Remove the dashboard's recorder-continues-on-exit reminder and duplicate mic
  status footer. Keep the existing Signal Status panel and useful shortcuts.

## Capabilities

### New Capabilities
- `tui-dashboard`: Refine the existing in-flight dashboard capability's layout,
  source selection and recording control presentation using the same capability path.

### Modified Capabilities
No canonical specs exist yet (`openspec list --specs` is empty). This change
supersedes the direct-Off-hotkey and immediate Source-cycling requirements in
`tui-dashboard-off-source`. Reconcile those clauses into the final dashboard
spec when the changes are archived, rather than retaining contradictory rules.

## Impact

`internal/tui/dashboard.go`, `rules.go`, `tui.go`, worker notice presentation,
TUI tests and README controls. No changes to device priorities, routing semantics,
saved JSON keys, capture prerequisites or recorder transport behavior. This request
produces planning artifacts only; implementation tasks remain unchecked.
