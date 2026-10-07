## ADDED Requirements
### Requirement: Discord voice controls
The system SHALL control Discord mute, deafen, camera, screen share and leaving the voice channel through Discord's local RPC pipe, using the user's own Discord application.
#### Scenario: First use
- **WHEN** credentials are configured and the user presses Connect
- **THEN** Discord shows its approval popup once, and the refresh token is saved without appearing in reports
#### Scenario: Discord closed
- **WHEN** Discord is not running
- **THEN** the controls are unavailable with "Discord closed" and the pipe is retried without prompting
#### Scenario: In-call controls
- **WHEN** the user is not in a voice channel
- **THEN** Camera, Screen share and Leave are unavailable

### Requirement: Linked mic mute
Snoofer's Mic mute preference SHALL also govern Discord mute, and a mute change made in Discord SHALL become the preference.
#### Scenario: Mute in Snoofer
- **WHEN** Mic mute is toggled in Snoofer while Discord is connected and not deafened
- **THEN** Voicemeeter and Discord are both muted, and the control is Pending until both agree
#### Scenario: Mute in Discord
- **WHEN** the user toggles mute in Discord
- **THEN** the Mic mute preference changes and Voicemeeter follows
#### Scenario: Deafened
- **WHEN** Discord is deafened
- **THEN** Discord's implied mute is neither adopted nor overwritten, and Mic mute is re-imposed on undeafen
#### Scenario: Reconnect
- **WHEN** Discord connects
- **THEN** Snoofer's preference is applied, and Discord's earlier state is not adopted
