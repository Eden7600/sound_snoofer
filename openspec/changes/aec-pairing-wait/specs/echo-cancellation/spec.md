## ADDED Requirements
### Requirement: Live callback pairing diagnosis
The paired probe SHALL report input and output formats and pairing mismatches without retaining audio.
#### Scenario: Alternating callbacks
- **WHEN** the live host delivers alternating input and output callbacks
- **THEN** diagnostics distinguish missing callbacks from format mismatches


### Requirement: Background callback timing
The monitor SHALL retain an honored 1 ms timer request while its callback registration exists and restore prior process policy and timer ownership after confirmed removal.
#### Scenario: Background audio host
- **WHEN** Snoofer is running in the background with registered audio callbacks
- **THEN** background timer throttling does not discard its timer resolution request
#### Scenario: Callback registration ends
- **WHEN** callback removal is confirmed or registration fails
- **THEN** the monitor releases its timer request and restores the prior process power policy

