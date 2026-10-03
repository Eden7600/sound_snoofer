# Proposal

## Why
Global transition gates toggle unrelated voice and recording sends when settings change, causing visible churn and avoidable audio interruptions.

## What Changes
- Disable only changed sends or sends dependent on a device/patch being changed.
- Order removals before device/patch writes and route additions.
- Preserve verification, ownership, drift detection and microphone Off behavior.

## Capabilities
### New Capabilities
- `routing-transitions`: Minimal, ordered routing transitions. Canonical specs are empty; this supersedes broad gating in the in-flight voice and recording designs.
### Modified Capabilities
None.

## Impact
Routing transition planner and regression tests. No configuration or native API changes.
