## ADDED Requirements

### Requirement: Independent microphone controls
Audio SHALL expose a shared master Mic stack enablement, profile-specific Mic stack targets and independent Mic mute. Targets SHALL exclude Off and retain their saved selection while the stack is disabled.

#### Scenario: Disable and edit target
- **WHEN** the user disables the stack and changes its target
- **THEN** the target saves without re-enabling the stack or connecting microphone devices.

#### Scenario: VR activation
- **WHEN** SteamVR activates or deactivates while the stack is disabled
- **THEN** the master stays disabled and each profile retains its own target.

#### Scenario: Restore
- **WHEN** the user re-enables the stack
- **THEN** normal availability and priority resolution reconnect the active profile's saved target without clearing shared mute.

### Requirement: Full stack teardown
Disabled stack routing SHALL disconnect managed microphone input assignments, ASIO input patches, microphone sends, monitoring and processing return. Playback, Volt A1 reservation and recorder transport SHALL remain unaffected.

#### Scenario: Muting versus disabling
- **WHEN** the user mutes an enabled stack
- **THEN** native mute changes without disconnecting input assignments or changing enablement or target.

#### Scenario: Disabled stack
- **WHEN** enablement is false with connected microphone devices
- **THEN** the routing plan clears the managed microphone stack while retaining output playback and issuing no recorder transport command.

### Requirement: Safe persistence
Enablement and targets SHALL save independently through existing atomic revision-checked persistence. Legacy source Off SHALL load as disabled with Automatic as its target.

#### Scenario: Restart
- **WHEN** Snoofer reloads disabled choices
- **THEN** the stack remains disabled with its last saved target and mute preference preserved.

