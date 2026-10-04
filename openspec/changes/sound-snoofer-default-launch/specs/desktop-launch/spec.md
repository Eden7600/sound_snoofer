## ADDED Requirements

### Requirement: Default launch opens live TUI
Sound Snoofer SHALL open its TUI in live mode when launched without arguments, and SHALL accept --dry-run to opt into preview.

#### Scenario: Double-click
- **WHEN** sound-snoofer.exe launches without arguments
- **THEN** the TUI opens in live mode using config.json beside the executable regardless of working directory
- **AND** live writer ownership is still required and recorder transport is unchanged

#### Scenario: Explicit preview
- **WHEN** --dry-run is supplied to the default launch, tui or watch
- **THEN** routing is preview-only

### Requirement: Default configuration is persistent
The application SHALL create missing default configuration from bundled defaults and SHALL preserve existing configuration and saved choices.

#### Scenario: First launch
- **WHEN** adjacent config.json is absent
- **THEN** a default config is created without overwriting a concurrent creator's file

#### Scenario: Existing or explicit config
- **WHEN** default config exists or --config names a file
- **THEN** that file and its saved sidecar are loaded without replacement

#### Scenario: Invalid or unwritable config
- **WHEN** configuration cannot be created or loaded
- **THEN** startup reports an error before native writes

### Requirement: Project identity is Sound Snoofer
Maintained project references SHALL use Sound Snoofer, sound-snoofer or sound_snoofer according to their naming convention.

#### Scenario: Project rename
- **WHEN** the renamed project is delivered
- **THEN** the folder is Documents/sound_snoofer, the executable is sound-snoofer.exe and Git history and personal configuration remain intact
