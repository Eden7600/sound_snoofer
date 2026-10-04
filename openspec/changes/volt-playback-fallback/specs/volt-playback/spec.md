## ADDED Requirements
### Requirement: Volt is the lowest-priority playback fallback
When ASIO playback is configured the system SHALL choose connected Volt after eligible configured WDM outputs and reuse ASIO A1.
#### Scenario: Preferred outputs connected
- **WHEN** AirPods or SteelSeries are eligible in the default profile
- **THEN** automatic playback selects them ahead of Volt
#### Scenario: Only Volt connected
- **WHEN** no configured WDM playback output is eligible and Volt is present
- **THEN** playback and its sends use A1 without opening a second Volt device
### Requirement: Explicit Volt selection follows hardware presence
The playback picker SHALL offer the uniquely connected configured ASIO interface and retain its saved preference across disconnects.
#### Scenario: Manual override and reconnect
- **WHEN** Volt is selected explicitly
- **THEN** it overrides WDM priorities while connected, falls back on disconnect and regains priority on reconnect
#### Scenario: Driver-only or ambiguous presence
- **WHEN** only the ASIO driver remains or hardware matching is ambiguous
- **THEN** Volt is not offered as a playback choice
#### Scenario: Moving playback and disabling mic
- **WHEN** playback moves between Volt A1 and a WDM output or the mic is turned Off
- **THEN** playback sends follow the chosen output and connected Volt stays assigned to A1
