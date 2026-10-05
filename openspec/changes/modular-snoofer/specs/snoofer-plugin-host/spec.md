## ADDED Requirements

### Requirement: Optional compiled plugins
Snoofer SHALL provide a domain-independent core and explicitly registered compiled plugins. Disabled plugins SHALL perform no construction, validation, initialization, polling, callback registration or resource acquisition. Static metadata MAY remain available.

#### Scenario: Core alone
- **WHEN** no plugins are enabled and Voicemeeter is absent
- **THEN** the tray and TUI provide configuration and diagnostics without loading audio libraries.

#### Scenario: Disabled integration
- **WHEN** Stream Deck or VR is compiled but disabled
- **THEN** no HID discovery or SteamVR observation belonging to that plugin runs.

### Requirement: Declared dependencies
Plugins SHALL declare dependencies and use small direct APIs only for those dependencies. The host SHALL reject invalid dependency graphs for affected branches, start dependencies first and stop dependents first.

#### Scenario: Invalid graph
- **WHEN** a plugin has a missing dependency, cyclic dependency or ambiguous duplicate registration ID
- **THEN** the affected branch is unavailable with a reason and unrelated valid plugins remain usable.

#### Scenario: Enablement closure
- **WHEN** the user enables VR or disables audio
- **THEN** the UI presents respectively enabling audio with VR or disabling VR with audio as one operation.

### Requirement: Transactional lifecycle changes
Enablement changes SHALL save atomically and use clean whole-application restart. Ordinary settings SHALL apply without restarting the host. Failed saves SHALL retain the current active configuration.

#### Scenario: Persistence failure
- **WHEN** saving the confirmed plugin selection fails
- **THEN** the application reports the failure and neither stops plugins nor restarts.

### Requirement: Owned cleanup and independent failure
Plugins SHALL release owned resources on stop and partial start failure. Expected failure SHALL affect only that plugin and its dependents, with explicit retry and no command replay.

#### Scenario: Missing device and reconnect
- **WHEN** enabled hardware is disconnected and later returns
- **THEN** the plugin reports disconnected and reconnects through its bounded device lifecycle without restarting unrelated plugins.

#### Scenario: Failed startup
- **WHEN** audio startup fails
- **THEN** audio and VR are unavailable, core and independent media/deck functionality remain available, and retry does not duplicate healthy instances.

#### Scenario: Shutdown uncertainty
- **WHEN** cleanup cannot confirm a native resource has stopped
- **THEN** the host reports the failure and does not automatically launch an overlapping owner.

### Requirement: Non-disruptive exit
Whole-application shutdown SHALL quiesce new work, suppress policy reconciliation during teardown, stop plugins in dependency order and leave mixer routing, recorder transport and current Windows defaults unchanged.

#### Scenario: Stop while recording in VR
- **WHEN** Snoofer exits while VR routing and recorder transport are active
- **THEN** it releases its resources without switching to Normal or stopping recording and clears reachable deck displays.
