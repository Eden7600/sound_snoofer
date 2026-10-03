# Proposal

## Why

The user wants a single mic On/Off switch that disconnects every microphone bus.
The existing voice-delivery control already gates most consumers, but its name
is unclear and mic B1 sends may remain unmanaged or frozen by recorder errors.

## What Changes

- Present existing enabled voice intent as the master Microphone On/Off switch.
- Off enforces zero A1–A5 and B1–B3 sends on desk, lav, webcam and AUX return.
- Off also disconnects mic B1 sends without a recording profile, or during a
  recorder conflict; source-off takes precedence over recorder B1 freeze.
- Preserve selected source, processing, monitor and recording preferences;
  On recomputes configured routes. Computer audio and recorder transport continue.

## Capabilities

### New Capabilities
- `microphone-switch`: Master disconnect of managed microphone consumers.

### Modified Capabilities
None. Existing voice and recording capabilities are still in-flight changes.

## Impact

Voice routing, recorder-conflict guard, TUI labels, tests and docs. Reuses the
version-1 enabled choice without introducing another overlapping switch.
