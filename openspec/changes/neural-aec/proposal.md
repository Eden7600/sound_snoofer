# Neural AEC engine selection

## Why
Try a trained local neural echo canceller and compare it with the existing multichannel WebRTC AEC3 without changing routing or losing the working default.

## What changes
Add a persisted Engine selection: WebRTC AEC3 (default), LocalVQE echo-only v1.4, LocalVQE voice cleanup v1.3. Package pinned upstream source and hash-verified weights through a reproducible native build. Expose truthful errors and 16 kHz mono processing limitations.

## Impact
AEC plugin, native bridge, build scripts, Audio GUI and UI contract. No change to audio routing, gain, mute, recorder, or deck positions.
