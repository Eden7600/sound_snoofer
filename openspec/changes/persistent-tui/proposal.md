# Proposal

## Why

The watcher emits scrolling logs, making it difficult to see current routing at a glance. A persistent terminal dashboard should remain open while Voice Snooter observes devices and enforces rules.

## What Changes

- Add a full-screen tui command with routing, inventory, and recent-event views.
- Show mode, connection/attention status, selected playback output, desired operations, and current assignments.
- Support live/dry switching, config reload, manual refresh, scrolling, resizing, and quit.
- Keep existing CLI commands and routing behavior. Default to dry-run and keep the interface open through engine/DLL errors.
- Persistence means an ongoing foreground terminal session, not startup registration or a service.

## Capabilities

### New Capabilities

- terminal-dashboard: Persistent interactive monitoring and control of the existing routing engine.

### Modified Capabilities

None. This builds on the unarchived automatic-device-routing implementation and does not close its pending hardware acceptance.

## Impact

New internal/tui package, CLI dispatch, and pinned Bubble Tea v2 dependency. A dedicated OS-thread actor owns API access and the existing writer lock. No extra audio processing or new routing settings.
