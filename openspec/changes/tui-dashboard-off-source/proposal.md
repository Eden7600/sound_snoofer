# Proposal

## Why

The current TUI is dense and its separate mic switch is confusing: Enter on a
boolean row silently does nothing, while dry-run saves choices without applying
them. The user wants Off to be a microphone source and a much nicer interface.

## What Changes

- Source cycles Desk, Lav, Webcam, Off; remove the separate mic enabled row.
- Enter and Space both change selectable controls; a visible 0 shortcut selects Off.
- Preserve old disabled intent as source Off, and selecting a mic re-enables it.
- Add a styled responsive dashboard with grouped controls, signal status,
  recorder status, contextual help and conspicuous dry-run/pending/error feedback.

## Capabilities

### New Capabilities
- `tui-dashboard`: Responsive audio control dashboard and unambiguous source Off.

### Modified Capabilities
None; predecessor specs are still in-flight. This supersedes the master switch UI.

## Impact

Intent normalization, routing, TUI rendering/input, tests and documentation. No
new runtime dependency, no automatic live mode, no recorder transport changes.
