## Why
Gain knobs show settings but not signal activity. Add live level bars without changing bindings or gain/mute behavior.

## What Changes
- Publish optional, timestamped level telemetry on existing audio gain controls.
- Render a segmented dBFS bar beneath gain readouts on Stream Deck dials.
- Sample through the existing serialized native worker, independent of routing polls.
- Keep meter-only changes outside command revisions and avoid redrawing static keys.

## Capabilities
### New Capabilities
- `streamdeck-knob-meters`: Live audio levels on gain knob displays.

## Impact
Audio native reads, worker snapshots, semantic telemetry, Stream Deck rendering. No settings migration, dependency or new audio owner.
