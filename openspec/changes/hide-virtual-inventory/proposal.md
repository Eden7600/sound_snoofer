# Proposal

## Why
Virtual audio endpoints clutter the device inventory used to inspect physical microphones and playback devices.

## What Changes
- Omit recognized virtual endpoints from the TUI Devices list and CLI devices text/JSON output.
- Preserve complete internal discovery for routing, ownership and drift detection.

## Capabilities
### New Capabilities
- `device-inventory-view`: Filtered user-facing inventory. Canonical specs are currently empty.
### Modified Capabilities
None.

## Impact
Shared model filtering, TUI and CLI presentation, tests and documentation. No mixer writes or routing changes.
