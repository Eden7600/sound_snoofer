## ADDED Requirements
### Requirement: Reset uses the running configuration
Resetting saved choices SHALL derive defaults from the configuration the audio worker is running, and SHALL NOT read any other file as configuration.

#### Scenario: Plugin with inline configuration
- **WHEN** the audio plugin runs with an inline configuration and no configuration file exists at its state path
- **THEN** reset choices succeeds and restores the inline configuration's voice defaults

#### Scenario: Corrupt saved choices
- **WHEN** saved choices are corrupt and the user resets them
- **THEN** reset succeeds from the running configuration and saves valid choices before any mixer write

### Requirement: Visible defaults
Behavior the user can change SHALL be stated in the shipped default configuration, or SHALL come from a single named default documented at the point of use.

#### Scenario: Factory configuration
- **WHEN** the embedded default configuration is decoded
- **THEN** it validates and states the Normal microphone priority explicitly

#### Scenario: Legacy configuration without profiles
- **WHEN** a configuration omits `profiles`
- **THEN** the named default microphone priority applies, and behavior matches previous releases

### Requirement: No unread schema
The audio configuration SHALL NOT accept settings that no component reads.

#### Scenario: Legacy deck profiles in audio config
- **WHEN** the audio configuration contains `stream_deck`
- **THEN** decoding fails with an unknown-field error, and the Stream Deck plugin settings remain the only deck layout

### Requirement: Policy names and steps are configurable
Process names used for detection and dial step sizes SHALL be optional settings that default to the previous built-in values.

#### Scenario: Custom processor executable
- **WHEN** `studio.voice.processor_process` names a different executable
- **THEN** Element-mode availability follows that process, and an absent process falls back to Direct exactly as before

#### Scenario: Invalid step
- **WHEN** a step setting is outside its allowed range
- **THEN** validation rejects the settings and the previous settings stay active

#### Scenario: Omitted settings
- **WHEN** none of these settings are present
- **THEN** behavior matches previous releases
