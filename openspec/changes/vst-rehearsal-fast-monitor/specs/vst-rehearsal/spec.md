## ADDED Requirements
### Requirement: Monitor labels identify VST stage
The TUI SHALL show Off, Pre-VST and Post-VST for Monitor without changing saved values or recording stage labels.
#### Scenario: Existing profile
- **WHEN** a profile contains monitor post
- **THEN** the control displays Post-VST
### Requirement: Rehearse a recording through Element
The TUI SHALL provide Recording to VST, Loop, Play Snippet and Stop Playback controls for the loaded native tape.
#### Scenario: Rehearsal enabled
- **WHEN** rehearsal is enabled in Element mode with an active source
- **THEN** the tape feeds B2, live mic sends and AUX-to-B3 are disabled, and monitoring selects dry tape or processed AUX without a feedback loop
#### Scenario: Off or Direct
- **WHEN** source Off or Direct is selected
- **THEN** rehearsal is disabled and its tape sends are disconnected
#### Scenario: Transport safety
- **WHEN** playback is requested in preview, while recording, before routes converge, or without rehearsal
- **THEN** it is rejected without starting transport
#### Scenario: Explicit transport
- **WHEN** Play is requested with verified rehearsal routes and stopped transport
- **THEN** the loaded tape starts at zero and native Loop follows the saved setting
- **AND** Stop stops playback without automatically starting recording
#### Scenario: Uncertain outcome
- **WHEN** a transport write fails or cannot be confirmed
- **THEN** its outcome is reported and it is not retried automatically
### Requirement: Send changes avoid device enumeration loops
Send-only changes SHALL debounce for 20 ms and verify changed operations using fresh parameters without repeated device enumeration when supported.
#### Scenario: Monitor switch
- **WHEN** only monitor sends change
- **THEN** unchanged operations are skipped and changed sends preserve break-before-make and readback verification
#### Scenario: Hardware changes or disconnects
- **WHEN** device assignment changes are needed or full boundary observations detect changed inventory
- **THEN** device debounce or replan applies and a stale topology is not reported converged
#### Scenario: Drift or ambiguity
- **WHEN** owned cells drift or hardware matching is ambiguous
- **THEN** the controller stops or rejects the affected plan rather than guessing
