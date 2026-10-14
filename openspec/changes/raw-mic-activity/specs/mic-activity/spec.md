## ADDED Requirements
### Requirement: Activity metering ignores insert processing
Activity metering SHALL measure microphone signal before any Voicemeeter input-insert stage, such as echo cancellation, whenever that insert is registered and delivering audio.

#### Scenario: Echo cancellation between sentences
- **WHEN** echo cancellation processes the desk mic, nobody talks, and the raw desk signal stays above the silence threshold
- **THEN** the desk mic is not latched Silent, and Auto keeps it

#### Scenario: Dead battery while processed
- **WHEN** echo cancellation processes the lav and its raw signal stays below the silence threshold for the configured time
- **THEN** the lav is latched Silent, and Auto moves to the next microphone

#### Scenario: No input insert
- **WHEN** no input insert is registered, or no input-insert audio arrived since the last sample
- **THEN** metering reads Voicemeeter's pre-fader input levels
