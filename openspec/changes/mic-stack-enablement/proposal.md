## Why
The microphone target selector currently doubles as the stack's power control. Turning it off loses the selected target and profile resolution can re-enable it.

## What Changes
- Expose independent Mic stack enablement, Mic stack target, and existing Mic mute controls.
- Use one persistent master enablement shared by Normal and VR.
- Keep Normal/VR targets editable while disabled; target edits and profile changes never enable the stack.
- Remove Off from target controls. Preserve legacy saved Off safely as disabled with Automatic as its target.
- Retain full input/send teardown, A1/output playback, recording transport and independent mute behavior.

## Capabilities
### New Capabilities
- `mic-stack-enablement`: Independent master enablement and persistent profile targets.

## Impact
Touches audio choices, profile resolution and semantic audio controls. Supersedes the source-Off UI in tui-dashboard-off-source and the per-profile Off assumption in modular-snoofer; retains disconnect-mic-devices-when-off routing guarantees.

