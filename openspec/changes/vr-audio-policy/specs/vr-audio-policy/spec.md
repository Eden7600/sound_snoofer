## ADDED Requirements
### Requirement: Runtime-gated headset eligibility
The system SHALL exclude classified VR endpoints from automatic priorities and manual source choices unless SteamVR is observed running and the endpoint is active and unambiguous.
#### Scenario: Enumerated but inactive runtime
- **WHEN** a headset endpoint remains enumerated after SteamVR exits or process observation fails
- **THEN** it is ineligible even when a generic device regex also matches it
#### Scenario: Reconnect and ambiguity
- **WHEN** SteamVR is running and a headset reconnects
- **THEN** unique active endpoints become eligible after device debounce and ambiguous matches remain unavailable with a reason
### Requirement: Independent headset preferences
The system SHALL persist independent microphone and playback preferences and derive effective choices without destroying normal selections.
#### Scenario: Prefer microphone only
- **WHEN** only headset microphone preference is enabled and a headset mic becomes eligible
- **THEN** the effective mic uses it while playback follows its normal policy
#### Scenario: Missing headset or runtime exit
- **WHEN** a preferred headset becomes unavailable
- **THEN** normal fallback applies visibly and the saved headset preference remains enabled
#### Scenario: Explicit source Off
- **WHEN** Source Off is selected while headset preference is enabled
- **THEN** mic devices remain disconnected, Volt A1 remains reserved, and VR does not reactivate the mic
#### Scenario: Stable selection
- **WHEN** repeated observations resolve to the same eligible devices
- **THEN** the system performs no device reassignment
### Requirement: Optional Windows default protection
The system SHALL offer persistent opt-in protection of configured Voicemeeter playback and capture defaults for all three Windows roles, independently of SteamVR runtime state.
#### Scenario: SteamVR replaces defaults
- **WHEN** protection is enabled live and a role points elsewhere
- **THEN** the system corrects and verifies that role against its configured active endpoint
#### Scenario: Missing target or unsupported setter
- **WHEN** a target is missing, ambiguous or the setter is unsupported
- **THEN** affected defaults remain untouched and the unmet protection is shown
#### Scenario: Repeated contention
- **WHEN** correction exceeds the bounded retry budget
- **THEN** automatic retries suspend with an actionable status rather than continuously switching defaults
#### Scenario: Disabled or preview
- **WHEN** protection is disabled or the program is in dry-run
- **THEN** no Windows defaults are written
#### Scenario: SteamVR exit
- **WHEN** SteamVR exits while protection remains enabled
- **THEN** protection continues while normal physical audio selection resumes
