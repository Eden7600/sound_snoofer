# Priority routing

## Purpose

Select preferred microphones and playback endpoints independently using explicit user priorities and deterministic, explainable matching.

## ADDED Requirements

### Requirement: Independent ordered preferences
The system SHALL select the first eligible candidate in each managed slot's ordered preferences. Fixed-slot mode SHALL support WDM assignments. Studio mode SHALL coordinate ASIO and dynamic playback as specified by studio-routing.

#### Scenario: Preferred microphone connects
- **WHEN** Volt 2 becomes eligible ahead of the configured webcam microphone
- **THEN** the planned microphone selection becomes Volt 2 without changing playback preferences

#### Scenario: AirPods disconnect
- **WHEN** AirPods are unavailable and the next configured candidate is the available SteelSeries speakers
- **THEN** the output plan selects those speakers without changing microphone preferences

#### Scenario: Preferred device returns
- **WHEN** a higher-priority candidate becomes eligible again
- **THEN** the plan returns to that candidate rather than remaining on fallback

### Requirement: Unambiguous endpoint matching
Candidates SHALL specify direction, driver type, and a Go regular expression matched against the public device name. Patterns SHALL be compiled during configuration validation. A pattern matching multiple available endpoints SHALL be ineligible and produce a diagnostic. Candidate order SHALL determine priority.

#### Scenario: Duplicate names
- **WHEN** a configured regex matches two available WDM microphones
- **THEN** that candidate is skipped as ambiguous and the next eligible preference is considered

#### Scenario: Enumeration order changes
- **WHEN** endpoint indexes change without identity or availability changes
- **THEN** the same candidate is selected

#### Scenario: Flexible device naming
- **WHEN** a configured case-insensitive Volt regex uniquely matches the available WDM input name
- **THEN** it selects that endpoint even when the name includes driver-specific prefixes or suffixes

#### Scenario: Invalid regular expression
- **WHEN** a configured pattern cannot be compiled
- **THEN** configuration validation fails with the candidate location and regex error before any write

### Requirement: Configuration validation
Configuration SHALL name each managed slot exactly once, use nonempty ordered preferences, and match the running edition's hardware capabilities. Invalid regexes, unknown fields, conflicting targets, unsupported drivers, and invalid timing values SHALL be rejected before any write.

#### Scenario: Potato-only target on Banana
- **WHEN** configuration targets A4 while Banana is running
- **THEN** validation rejects the configuration and no assignments are changed

### Requirement: No eligible candidate
When no candidate is eligible, the system SHALL leave the slot unchanged and report an unresolved selection. It SHALL NOT choose an unconfigured device or describe the unchanged assignment as working audio.

#### Scenario: All microphones disconnected
- **WHEN** neither Volt 2 nor webcam microphone is eligible
- **THEN** the input assignment is left unchanged and the plan reports no available configured microphone
