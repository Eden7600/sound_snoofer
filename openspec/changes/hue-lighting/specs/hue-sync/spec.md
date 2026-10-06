## ADDED Requirements
### Requirement: Hue Sync third-party control
The Hue plugin SHALL connect to the Hue Sync PC app's loopback third-party control socket and reflect its reported state alongside the bridge.
#### Scenario: App running with third-party control enabled
- **WHEN** the socket accepts a connection and reports app state
- **THEN** Sync, mode and intensity show observed values
#### Scenario: App closed or control disabled
- **WHEN** the connection is refused
- **THEN** sync status shows N/A, the diagnostic names starting Hue Sync and enabling Third-party control, reconnection backs off, and the room half keeps working
#### Scenario: App restarts
- **WHEN** the socket closes and later accepts a connection
- **THEN** values remain unknown until a new state event arrives and no command is replayed
#### Scenario: Hue Sync loses its bridge
- **WHEN** the app reports bridge_disconnected
- **THEN** all sync controls except status are unavailable

### Requirement: Sync commands
The Hue plugin SHALL send explicit one-shot sync commands and confirm them from observed state.
#### Scenario: Toggle sync
- **WHEN** Sync is pressed
- **THEN** start_sync or stop_sync is sent from the observed state and the key shows Wait until the state changes
#### Scenario: Sync unconfirmed
- **WHEN** no matching state arrives within 3 seconds
- **THEN** the key shows Error and the command is not retried
#### Scenario: Mode or intensity while not syncing
- **WHEN** sync is not running
- **THEN** mode and intensity selections are unavailable
#### Scenario: Dry run
- **WHEN** Snoofer runs with --dry-run
- **THEN** sync state is observed but no command is sent

### Requirement: Joined room and sync behavior
The Hue plugin SHALL route shared controls to whichever half currently drives the lights.
#### Scenario: Brightness while syncing
- **WHEN** the brightness dial turns while Hue Sync is syncing
- **THEN** coalesced relative sync brightness steps are sent instead of room writes, and the dial shows the sync brightness
#### Scenario: Brightness press while syncing
- **WHEN** the brightness dial is pressed while syncing
- **THEN** the room toggles on or off as usual
#### Scenario: Scene while syncing
- **WHEN** a scene is pressed while Hue Sync is syncing
- **THEN** sync is stopped and the scene is recalled only after the stop is confirmed
#### Scenario: Sync stop unconfirmed
- **WHEN** Hue Sync does not confirm the stop within 3 seconds
- **THEN** the scene shows Error and is not recalled
#### Scenario: Temperature while syncing
- **WHEN** Hue Sync is syncing
- **THEN** the temperature dial is unavailable with a Sync active status
