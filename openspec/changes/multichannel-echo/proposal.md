# Multichannel echo reference and lossless framing
## Why
The current reference averages only the front two bus channels. Discrete 5.1 center, LFE and surround content is missing, and phase-opposed stereo can vanish. The speaker also upmixes stereo internally; Snoofer can only observe the signal sent to it.
## What Changes
- Preserve all eight reference slots of the resolved playback bus, including silent slots, with explicit multichannel WebRTC processing.
- Disable the bundled AEC3 left/right-only stereo detector so center-only content is not collapsed.
- Reframe incrementally without dropping samples; check reverse processing and callback pairing/rates, failing open on discontinuities.
- Extend the existing offline acoustic probe and native ABI/routing tests. Preserve live mode/strength, routing, gains and mutes.
## Impact
Native shim, Go ABI, echo targets and offline probe. The expanded config uses a versioned export to reject mixed executable/DLL versions. No new dependency or UI.
