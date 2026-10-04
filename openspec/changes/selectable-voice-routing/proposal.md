# Proposal

## Why

Device routing alone cannot express which microphone to use, whether voice passes through Element, or when monitoring should be audible. A working Potato setup is now confirmed by the user's Discord mic test; Sound Snoofer needs to preserve and manage that topology explicitly from its persistent TUI.

## What Changes

- Add an opt-in Potato voice profile: selected mic -> B3 directly, or selected mic -> B2 -> Element AUX ASIO -> AUX return -> B3.
- Offer desk (Volt channel 1), lav (Volt channel 2), and webcam source selection. Preserve the selected Volt channel across temporary webcam fallback; silence does not mean hardware disconnection.
- Add voice enable/disable, Direct/Element selection, Off/Pre/Post monitoring, and per-source app-playback rule toggles with visible desired and observed state.
- Persist explicit TUI selections without rewriting device regex configuration or persisting live-write permission.
- Enforce exclusive voice paths, a protected AUX-to-B2 prohibition, and break-before-make transitions with verified readback.
- Preserve existing behavior for configurations without the new profile. The new profile reserves webcam input 3 and AUX for processing; these ownership changes are explicit opt-in extensions to the older studio design.

## Capabilities

### New Capabilities

- `voice-routing`: Selectable microphone, processing, monitoring and app-playback policies, their TUI controls, persistence and safe reconciliation.

### Modified Capabilities

None in the canonical spec inventory (openspec/specs is currently empty). This change builds on the unarchived automatic-device-routing and persistent-tui changes. Its opt-in ownership and fallback requirements supersede the older input-1 webcam fallback and generic send migration only when the voice profile is configured. Archive the prerequisite changes before this one; do not silently rewrite their historical artifacts.

## Impact

Affected areas: internal/config, routing, controller, voicemeeter, tui, CLI dependency wiring, example configuration and setup documentation. Extend the numeric Remote API allowlist to edition-valid B sends and read the required routing state. Retain the current serialized DLL worker and writer mutex. No new audio engine, VST hosting, Discord automation, driver installation, or arbitrary routing scripting is required.

The user's confirmed test proves audible delivery through the configured pass-through setup, not plugin processing, restart recovery, disconnect handling, or monitoring. Those remain acceptance tests.
