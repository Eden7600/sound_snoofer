## ADDED Requirements
### Requirement: Shared serialized control actions
The application SHALL accept typed actions from concurrent control surfaces through one audio owner with per-client acknowledgments and bounded state delivery.
#### Scenario: TUI and hardware act concurrently
- **WHEN** both submit changes
- **THEN** actions are serialized, each receives its own result, and neither freezes the other surface
#### Scenario: Target changes
- **WHEN** a queued mic gain action refers to a previous source generation
- **THEN** it is rejected rather than applied to the replacement microphone
### Requirement: Native gain control and observation
The application SHALL adjust and report verified A1, A2 and active physical microphone gain in dB using validated numeric parameters.
#### Scenario: Repeated encoder increments
- **WHEN** several ticks arrive before verification
- **THEN** their bounded accumulated adjustment applies once to the same target and the display distinguishes pending from observed gain
#### Scenario: Unknown or disconnected mic
- **WHEN** no active physical microphone gain can be read
- **THEN** adjustment is unavailable rather than using a guessed value
### Requirement: Native microphone mute
Microphone mute SHALL use Voicemeeter native mute, preserve assignments, patches and sends, and cover both physical mic and managed processing return.
#### Scenario: Mute while recording
- **WHEN** microphone mute is enabled
- **THEN** mic audio is silenced without stopping recording, changing source or suppressing computer capture
#### Scenario: Unmute
- **WHEN** the user removes mic mute
- **THEN** only Snoofer-owned mute is released and pre-existing manual mute is preserved
#### Scenario: Source switches while muted
- **WHEN** policy selects another microphone
- **THEN** its native mute is verified before the new microphone is routed
### Requirement: Playback-following speaker mute
Speaker mute SHALL use the native mute of the current physical playback bus without changing routing or device selection.
#### Scenario: Playback moves between A1 and A2
- **WHEN** the device changes with speaker mute enabled
- **THEN** mute is established on the new bus before routing and the old bus's owned mute is released safely
### Requirement: Fast verified controls
Gain and mute SHALL bypass device debounce and show pending, verified, unknown or failed state accurately.
#### Scenario: Mute verification failure
- **WHEN** the setter succeeds but readback fails
- **THEN** the UI does not claim the mic is confirmed muted
#### Scenario: Preview
- **WHEN** gain or mute is changed in dry-run
- **THEN** no native audio parameter is written
