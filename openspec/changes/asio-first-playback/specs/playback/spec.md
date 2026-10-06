## ADDED Requirements
### Requirement: ASIO-first eligibility
The system SHALL select at most one physically present interface in configured priority order for A1 before resolving microphone and playback priorities. Other ASIO devices SHALL be unavailable to downstream selection.
#### Scenario: Conflicting interfaces
- **WHEN** microphone and playback prefer different connected ASIO interfaces
- **THEN** the ASIO priority winner remains loaded and the other preference falls through to its next eligible candidate
#### Scenario: Presence and ambiguity
- **WHEN** a driver is installed without hardware or matches are ambiguous
- **THEN** absent hardware is skipped and ambiguous selection fails closed
#### Scenario: No ASIO or mic disabled
- **WHEN** no interface is eligible or the mic stack is disabled
- **THEN** ordinary playback remains available and disabling mic does not unload an eligible clock interface
### Requirement: Semantic playback surface
The GUI and default Stream Deck SHALL expose one Playback gain, meter and mute instead of fixed A1/A2 controls.
#### Scenario: Destination changes
- **WHEN** playback moves to another bus or device
- **THEN** controls follow that destination, stale gain actions are rejected and gains remain unchanged unless explicitly adjusted
### Requirement: Authoritative playback mute
Snoofer SHALL persist requested playback mute and impose both muted and unmuted states on the selected output with verified readback.
#### Scenario: Drift or restart
- **WHEN** native mute differs from the persisted preference after external changes or restart
- **THEN** Snoofer restores its requested state without adopting native state
#### Scenario: Muted transition
- **WHEN** playback changes destination while muted
- **THEN** mute is verified before enabling the new route and obsolete ownership is released only after routing settles

### Requirement: Adjacent playback and microphone dials
The default and personal Home layout SHALL place Playback at dial 0 and Mic at dial 1, leaving dial 2 empty.
#### Scenario: Compact mixer layout
- **WHEN** the revised layout is loaded
- **THEN** Mic is immediately right of Playback and other keys, pages and dials retain their positions
