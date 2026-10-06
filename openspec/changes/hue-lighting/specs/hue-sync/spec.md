## ADDED Requirements
### Requirement: Hue Sync third-party control
The system SHALL provide an optional Hue Sync plugin that connects to the Hue Sync PC app's loopback third-party control socket and reflects its reported state.
#### Scenario: App running with third-party control enabled
- **WHEN** the socket accepts a connection and reports app state
- **THEN** Sync, brightness, mode and intensity show observed values
#### Scenario: App closed or control disabled
- **WHEN** the connection is refused
- **THEN** status shows N/A, the diagnostic names starting Hue Sync and enabling Third-party control, and reconnection backs off
#### Scenario: App restarts
- **WHEN** the socket closes and later accepts a connection
- **THEN** values remain unknown until a new state event arrives and no command is replayed
#### Scenario: Hue Sync loses its bridge
- **WHEN** the app reports bridge_disconnected
- **THEN** all Hue Sync controls except status are unavailable

### Requirement: Sync commands
The Hue Sync plugin SHALL send explicit one-shot sync commands and confirm them from observed state.
#### Scenario: Toggle sync
- **WHEN** Sync is pressed
- **THEN** start_sync or stop_sync is sent from the observed state and the key shows Wait until the state changes
#### Scenario: Sync unconfirmed
- **WHEN** no matching state arrives within 3 seconds
- **THEN** the key shows Error and the command is not retried
#### Scenario: Brightness dial
- **WHEN** the Hue Sync brightness dial turns
- **THEN** relative inc_bri steps are sent with ticks accumulated while a command is outstanding, and pressing the dial toggles sync
#### Scenario: Mode or intensity while not syncing
- **WHEN** sync is not running
- **THEN** mode and intensity selections are unavailable
#### Scenario: Dry run
- **WHEN** Snoofer runs with --dry-run
- **THEN** the plugin observes state but sends no commands
