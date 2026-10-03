# Studio routing

## Purpose

Continuously enforce logical microphone and playback rules as an ASIO interface and playback devices connect, disappear, or move between output buses.

## ADDED Requirements

### Requirement: ASIO hardware presence and ownership
Studio mode SHALL reserve A1 for a unique configured ASIO driver only while a uniquely matching WDM input companion provides presence evidence. Installed ASIO drivers alone SHALL NOT establish hardware presence. Unrelated A1 occupants SHALL block the ASIO transition.

#### Scenario: Volt becomes available
- **WHEN** the Volt companion input and unique ASIO driver are present
- **THEN** A1 is assigned Volt ASIO and playback is allocated elsewhere

#### Scenario: Driver remains after unplug
- **WHEN** the ASIO driver remains enumerated but the companion endpoint is absent
- **THEN** Volt is treated as unavailable and playback can reclaim A1

### Requirement: Independent mono input channels
With Volt active, the application SHALL clear direct device selections on Stereo Inputs 1 and 2 and set their L/R ASIO patches to 1/1 and 2/2 respectively. On Volt departure it SHALL disable those four patches, select the configured webcam fallback on input 1 if available, and leave input 2 without a direct device.

#### Scenario: Lav on input 2
- **WHEN** Volt is active with a lav plugged into physical input 2
- **THEN** its planned channel mapping feeds both sides of Stereo Input 2 independently of Stereo Input 1

### Requirement: Lowest free playback output
The selected playback device SHALL occupy the lowest unreserved hardware output. Existing assignments matching configured playback patterns are owned and movable; unrelated occupied outputs SHALL be preserved. If no playback device is eligible, the topology SHALL remain unchanged and report unresolved.

#### Scenario: Normal A1 and A2 transition
- **WHEN** Volt connects while playback occupies A1 and A2 is free
- **THEN** playback moves to A2; on Volt departure it returns to A1 and the former A2 assignment is cleared

#### Scenario: Another output is occupied
- **WHEN** Volt owns A1 and an unrelated device occupies A2
- **THEN** playback uses A3 if free, or reports no free output

### Requirement: Migrate playback sends
When enabled, the application SHALL transfer enabled strip playback sends to the new output and disable sends to former playback outputs. If multiple matched playback outputs exist, enabled sends SHALL be combined using logical OR. Other sends SHALL be preserved except where an explicit rule overrides them.

#### Scenario: Playback changes buses
- **WHEN** playback moves from A1 to A2
- **THEN** enabled strip sends follow playback and former playback sends turn off

### Requirement: Persistent semantic source rules
Configured playback_sources SHALL always route to the selected playback output, including when device assignments remain unchanged. virtual:1 SHALL resolve to primary VAIO for the running edition. Rules SHALL override migrated button state and prevent primary playback from continuing to the managed ASIO A1 output.

#### Scenario: Manual routing drift
- **WHEN** VAIO's send to the playback output is manually disabled
- **THEN** live watch restores it after the stable-observation interval

#### Scenario: Edition migration
- **WHEN** the running edition changes from Banana to Potato
- **THEN** virtual:1 resolves from strip index 3 to index 5 without changing the configured rule

### Requirement: Verified bounded topology changes
Each topology operation SHALL be verified before proceeding. Changed inventory or unexpected concurrent setting changes SHALL stop the current pass. The operation sequence SHALL use only allowlisted numeric parameters and report partial application on failure.

#### Scenario: Patch write fails
- **WHEN** an ASIO patch operation fails
- **THEN** later operations stop and the failure reports how many operations were verified
