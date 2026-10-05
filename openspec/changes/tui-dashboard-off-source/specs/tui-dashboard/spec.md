# TUI dashboard

## Purpose

Make daily audio control legible and keyboard-accessible with a single microphone source selector and clear observed routing status.

## ADDED Requirements

### Requirement: Off is a source
The source selector SHALL include Off and SHALL remove the separate enabled row. Off SHALL disconnect all managed mic and AUX-return sends to A1–A5/B1–B3 without changing computer audio or recorder transport. Old disabled choices SHALL normalize to Off; selecting a physical mic SHALL enable it and preserve other preferences.

#### Scenario: Select Off and restore
- **WHEN** the user chooses Off then Lav
- **THEN** Off clears managed mic sends and Lav resumes current configured processing/monitor/recording preferences

#### Scenario: Disconnected or ambiguous mic
- **WHEN** Off is selected while Volt disconnects, reconnects or fallback matches ambiguously
- **THEN** no fallback mic is activated and effective source stays Off

#### Scenario: Failure
- **WHEN** saving or applying Off fails
- **THEN** the UI reports the error and does not claim verified mic disconnection

### Requirement: Responsive dashboard and controls
The TUI SHALL group microphone, playback and recording controls with a visible selected row, clear mode/status and contextual keyboard help. Enter and Space SHALL change non-transport controls. The normal TUI SHALL omit transport and restart action rows and the Actions heading. An externally requested disruptive restart SHALL retain an explicit Enter-only confirmation under System. Source Off SHALL have a visible direct keyboard shortcut. External text SHALL be sanitized before styling, and NO_COLOR SHALL disable styling.

#### Scenario: Small window
- **WHEN** the terminal shrinks
- **THEN** content remains bounded and the selected control stays visible or a terminal-too-small message appears

#### Scenario: Preview mode
- **WHEN** Off is selected in dry-run
- **THEN** its saved choice is visible with a preview/no-mixer-writes indication rather than an applied claim

#### Scenario: Reconnect
- **WHEN** recorder or mixer reconnects
- **THEN** fresh observed status replaces unknown status without automatically starting capture
