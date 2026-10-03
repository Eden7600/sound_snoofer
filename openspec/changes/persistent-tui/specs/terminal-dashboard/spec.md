# Terminal dashboard

## Purpose

Keep audio-routing status visible in an interactive terminal while the existing controller continues monitoring and optionally enforcing rules.

## ADDED Requirements

### Requirement: Persistent dashboard
The tui command SHALL use a full-screen terminal view showing current mode, routing state, device inventory and recent events. It SHALL refresh continuously and support resizing and scrolling without unbounded event history.

#### Scenario: Inspect while monitoring
- **WHEN** the user switches between Routing, Devices, and Events views
- **THEN** monitoring continues and the screen shows the latest available state

### Requirement: Explicit live control
The TUI SHALL default to dry-run. Explicit --apply or the live toggle SHALL acquire writer ownership before any writes. Failure to acquire ownership SHALL keep the session dry and display the reason. Mode changes SHALL reset debounce and be acknowledged after the current serialized operation.

#### Scenario: Another writer is running
- **WHEN** the user requests live mode while another writer owns the session
- **THEN** the TUI remains dry-run and reports the conflict

### Requirement: Configuration reload
The TUI SHALL reload configuration on request. Invalid replacement configuration SHALL preserve the last valid configuration and report the error. Successful reload SHALL reset pending routing decisions.

#### Scenario: Invalid regex on reload
- **WHEN** the config file contains an invalid regex
- **THEN** the old valid configuration remains active and the error is displayed

### Requirement: Recoverable connection errors
The dashboard SHALL remain open when the DLL or engine is unavailable, report stale/invalid state explicitly, and retry connection or observation. Errors SHALL NOT masquerade as healthy current state.

#### Scenario: Voicemeeter unavailable
- **WHEN** a connection attempt or observation fails
- **THEN** the interface remains interactive and reports the error until recovery

### Requirement: Terminal and API cleanup
Quit and Ctrl+C SHALL cancel pending work, close the API session, release writer ownership, and restore the terminal. Redirected input/output SHALL be rejected with guidance to use the noninteractive commands. Device names SHALL NOT inject terminal control sequences.

#### Scenario: User quits
- **WHEN** q or Ctrl+C is pressed
- **THEN** the worker is cancelled and resources are released before the command exits
