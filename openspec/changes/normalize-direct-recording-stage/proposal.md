# Proposal

## Why
Direct mode currently permits Post recording intent, which disables microphone capture instead of selecting a usable stage. The user wants automatic protection against that invalid combination.

## What Changes
- Normalize Direct/Post recording intent to Direct/Pre on edit, effective load and save.
- Keep Pre selected when returning to Element; prevent Post selection in Direct mode.
- Preserve recording inclusion flags and transport.
- Use a 100 ms debounce for routing-only plans while retaining configured device-assignment debounce; schedule the pending deadline without waiting for the normal poll interval.

## Capabilities
### New Capabilities
- `recording-stage-validity`: Valid recording stage choices across processing modes; supersedes inactive Direct/Post behavior in in-flight recording specs.
- `routing-response-time`: Separate routing-only and hardware assignment debounce timing.
### Modified Capabilities
None; canonical specs are empty.

## Impact
Config intent normalization/persistence, TUI controls, affected routing tests and documentation.
