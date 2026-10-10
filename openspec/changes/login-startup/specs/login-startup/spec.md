## ADDED Requirements
### Requirement: Login startup toggle
Snoofer SHALL offer a Start at Login toggle that registers or removes a per-user Windows sign-in launch of the running executable with its absolute configuration path. The toggle SHALL show the registration Windows actually holds, read back after every change, and SHALL NOT be persisted in Snoofer's configuration.

#### Scenario: Turn on
- **WHEN** Start at Login is Off and the user presses it
- **THEN** a Run entry launching this executable with `--config` and the absolute configuration path is written, read back, and the toggle shows On

#### Scenario: Turn off
- **WHEN** Start at Login is On and the user presses it
- **THEN** the Run entry is removed, read back, and the toggle shows Off

#### Scenario: Sign-in
- **WHEN** the user signs in to Windows with Start at Login On
- **THEN** Snoofer starts in the tray with the registered configuration, without a controls window, in live mode

#### Scenario: Disabled in Windows
- **WHEN** Windows' startup-apps setting has disabled the entry
- **THEN** the toggle shows Off with the note "Disabled in Windows startup apps", and turning it on re-enables it

#### Scenario: Another copy registered
- **WHEN** the Run entry launches a different executable or configuration
- **THEN** the toggle shows Off with the registered command in its note, and turning it on replaces the entry with this copy

#### Scenario: Read failure
- **WHEN** the Run entry cannot be read
- **THEN** the toggle shows N/A, is unavailable, and its note gives the error

#### Scenario: Write not observed
- **WHEN** a write fails or the read-back does not match the request
- **THEN** the GUI notice reports the failure and the toggle keeps showing the observed state

#### Scenario: Preview
- **WHEN** Snoofer runs with `--dry-run`
- **THEN** the toggle shows the observed state, is unavailable, and login startup is never changed

#### Scenario: Stale press
- **WHEN** a press was generated against a toggle state that has since changed
- **THEN** the press is rejected and the registration is unchanged
