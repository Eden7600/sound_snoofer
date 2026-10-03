# Proposal

## Why
Source Off currently clears sends but leaves microphone devices open. Off should release the managed microphone devices as well.

## What Changes
- Clear managed microphone input assignments and ASIO input patches while Off.
- Keep Volt assigned to A1 and preserve playback placement; restore microphone input selection when a microphone is enabled.
- Preserve source preferences, computer capture and recorder transport.

## Capabilities
### New Capabilities
- `microphone-devices`: Microphone device ownership and release while Off. Extends the in-flight microphone-switch behavior; no canonical specs exist yet.
### Modified Capabilities
None.

## Impact
Routing planner, regression tests, documentation. Existing verified Remote API device-clear operations are reused; no new dependencies.
