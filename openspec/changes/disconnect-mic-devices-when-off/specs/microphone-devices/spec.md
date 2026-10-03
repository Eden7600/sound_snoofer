## ADDED Requirements

### Requirement: Off disconnects microphone inputs
The system SHALL clear managed microphone device assignments and ASIO input patches when Source is Off (including legacy disabled intent), in addition to disconnecting all managed microphone sends.

#### Scenario: Volt and webcam are assigned
- **WHEN** Source changes to Off
- **THEN** input 1/2 assignments and the owned input 3 webcam assignment are empty and ASIO patches 0 through 3 are zero
- **AND** Volt remains assigned to A1 and playback retains its normal output and sends
- **AND** computer capture preferences and recorder transport remain unchanged

#### Scenario: Devices disconnect or inventory is ambiguous
- **WHEN** Source is Off and webcam devices disappear, reconnect, or match ambiguously
- **THEN** managed microphone inputs remain disconnected without webcam-selection errors

#### Scenario: Playback is unavailable
- **WHEN** Source is Off and no playback candidate is available
- **THEN** owned microphone input devices and patches are still cleared and playback remains unresolved
- **AND** unrelated output assignments are preserved

#### Scenario: Unknown webcam assignment
- **WHEN** input 3 contains an unmanaged device
- **THEN** planning reports the ownership conflict rather than clearing that device

#### Scenario: Device release fails
- **WHEN** a device setter fails or readback does not confirm disconnection
- **THEN** reconciliation reports failure rather than claiming success and a fresh plan can recover

### Requirement: Microphone selection restores hardware
The system SHALL restore normal hardware selection when a microphone source is selected after Off, preserving mode, monitor and recording preferences.

#### Scenario: Return to Volt
- **WHEN** Desk or Lav is selected and Volt is available
- **THEN** Volt remains on A1, its channels map to stereo inputs 1 and 2 again, playback retains its output and the selected microphone routes are restored

#### Scenario: Remain Off
- **WHEN** a completed Off plan is reconciled again
- **THEN** no redundant device or patch writes are required
