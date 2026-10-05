## Why

Sound Snoofer currently composes audio, VR, Stream Deck and media behavior as one application. Users cannot omit hardware integrations without retaining their startup paths, and an unrelated future plugin would inherit audio-specific state and UI assumptions. Snoofer needs a small host with optional compiled plugins while preserving the working audio behavior.

## What Changes

- **BREAKING:** Rename the application and executable to Snoofer; retain the repository directory.
- Extract optional audio, VR, Stream Deck and Windows media plugins. Keep lifecycle, configuration, tray, TUI and semantic control services in core.
- Compile selected plugins into the executable; independently enable or disable them. Disabled plugins perform no initialization or background work.
- Declare dependencies and permit direct calls through small dependency APIs. VR depends on audio; Stream Deck consumes registered controls without depending on their providers.
- Apply plugin enablement changes through clean application restart. Apply ordinary settings and deck layout edits live.
- Replace VR preference toggles with separate Normal and VR microphone/playback profiles. SteamVR running selects VR; normal settings remain editable while overridden.
- Provide a TUI Stream Deck page/binding configurator, shared bindings and a reserved last dial for page navigation.
- **BREAKING:** Replace configuration structure and manually convert the owner's files during implementation. Preserve backups and safety journals; no automatic migration layer.

## Capabilities

### New Capabilities
- `snoofer-plugin-host`: Optional compiled plugins, declared dependencies, lifecycle and independent failures.
- `snoofer-semantic-controls`: Domain-neutral controls, configuration ownership and integrated presentation.
- `snoofer-audio-profiles`: Audio ownership and separate VR policy.
- `snoofer-deck-layouts`: User-owned pages and bindings.
- `snoofer-changeover`: Build composition, rename, manual conversion and validation.

## Impact

Touches application composition, desktop transport, configuration, control worker ownership, TUI and Stream Deck bindings/rendering. Reuse deterministic routing, controller verification, native adapters and Studio artwork. Audio safety invariants remain binding.

This change supersedes the composition and configuration assumptions of existing audio-centric changes, the prefer-headset model in `vr-audio-policy`, and fixed bindings in `direct-stream-deck`. It preserves their unrelated routing, recording, recovery, default-device protection and device protocol requirements. During implementation reconcile overlapping documents explicitly.

Hue, runtime plugin loading, process isolation, a marketplace, a general rules engine, macros and a web configurator are out of scope. This proposal authorizes no implementation by itself.
