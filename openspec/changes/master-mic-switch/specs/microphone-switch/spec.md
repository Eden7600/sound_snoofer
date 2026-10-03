# Microphone switch

## Purpose

Provide a persistent master switch that disconnects managed microphone consumers while preserving computer playback and capture.

## ADDED Requirements

### Requirement: All managed microphone sends off
Mic Off SHALL desire zero A1–A5 and B1–B3 sends on desk, lav, webcam and Element return strips, even without recording configuration or with unknown/conflicting recorder state. It SHALL preserve computer playback/capture and recorder transport. Recorder conflicts SHALL never allow new mic sends to enable.

#### Scenario: Recording continues without microphone
- **WHEN** Mic becomes Off during combined recording
- **THEN** all managed microphone sends turn off and configured computer capture remains enabled without a transport command

#### Scenario: Recorder unavailable
- **WHEN** Mic is Off and recorder observation fails
- **THEN** microphone B1 zero operations remain permitted while recorder preparation and positive sends stay blocked

#### Scenario: Disconnect and reconnect
- **WHEN** Volt disconnects and reconnects while Mic is Off
- **THEN** no selected mic, fallback or processing return becomes routed

#### Scenario: Ambiguous fallback
- **WHEN** fallback matching is ambiguous while Mic is Off
- **THEN** no ambiguous mic is selected or enabled

#### Scenario: Setter failure
- **WHEN** a disconnect write fails
- **THEN** the UI reports failure/pending and does not claim verified disconnection

### Requirement: Persistent master switch and restore
The TUI SHALL expose Microphone On/Off using Space and preserve processing, source, monitoring and recording choices. On SHALL recompute configured routing using current devices. Existing enabled choices SHALL remain compatible. Dry-run SHALL save intent without mixer writes.

#### Scenario: Off then On
- **WHEN** the user switches Off and later On
- **THEN** the same configured preferences resume subject to source availability, without restoring arbitrary unmanaged sends or changing recorder transport
