> **Superseded behavior:** For Direct/Post and Element availability, the current element-process-fallback requirements and AGENTS.md take precedence: retain Post as a preference, use effective Pre in Direct, and restore Post with Element. Live is the default; explicit --dry-run selects preview under sound-snoofer-default-launch. Historical contradictory scenarios below are retained for context, not current acceptance.

# Voice routing

## Purpose

Manage selectable microphone, Element processing, monitoring and playback paths as visible persistent rules without mixing dry and processed voice or creating feedback.

## ADDED Requirements

### Requirement: Explicit profile ownership
The voice profile SHALL be opt-in and require Potato. It SHALL reserve inputs 1/2 for Volt, input 3 for the configured webcam, B2 for the Element send, AUX for its return and B3 for app capture. Configurations without this profile SHALL retain existing behavior. Conflicting playback use of AUX SHALL be rejected before writes.

#### Scenario: Banana or conflicting configuration
- **WHEN** the profile is used on Banana or includes AUX as an app-playback source
- **THEN** validation reports the conflict and no mixer settings are written

#### Scenario: Reserved input occupied
- **WHEN** input 3 contains an endpoint outside the configured webcam patterns
- **THEN** the profile reports the ownership conflict without replacing that endpoint

### Requirement: Preferred and effective microphone
The system SHALL offer desk, lav and webcam selection. Desk and lav SHALL use Volt channels 1 and 2 respectively, each patched to both sides of its corresponding strip. If Volt is absent, either selection SHALL fall back to a uniquely matched webcam on input 3, retaining the preference. Silence SHALL NOT trigger source changes.

#### Scenario: Dead lav battery
- **WHEN** lav is selected and Volt remains present but the lav signal is silent
- **THEN** lav remains selected until the user changes source; desk is not selected automatically

#### Scenario: Disconnect and reconnect
- **WHEN** Volt disappears and later returns while lav is preferred
- **THEN** the effective source becomes webcam if eligible, then returns to lav after stable observations

#### Scenario: Ambiguous or missing webcam
- **WHEN** no eligible unique webcam candidate exists and Volt is absent
- **THEN** voice and monitoring sends are disabled, the preference is retained, and source availability is reported unresolved

### Requirement: Exclusive voice delivery
When voice is enabled, Direct SHALL route only the effective microphone to B3. Element SHALL route only that microphone to B2 and AUX to B3, with direct mic-to-B3 disabled. When voice is disabled, all managed B2/B3 voice sends and monitoring sends SHALL be off. Unselected microphones SHALL never feed these buses.

#### Scenario: Direct mode
- **WHEN** desk is effective and Direct is selected
- **THEN** input 1 sends to B3, no microphone sends to B2, and AUX does not send to B3

#### Scenario: Element mode
- **WHEN** desk is effective and Element is selected
- **THEN** input 1 sends to B2, AUX sends to B3, and physical mic sends to B3 are off

### Requirement: Protected processing and capture buses
The profile SHALL keep AUX-to-B2 off and exclude all other strips from B2/B3 except the routes selected by voice mode. These constraints SHALL remain enforced while optional rules are disabled. B1, gains, mutes, effects and insert settings SHALL remain outside profile ownership.

#### Scenario: Accidental loop or desktop audio capture
- **WHEN** AUX-to-B2 or primary VAIO-to-B3 is manually enabled during live enforcement
- **THEN** reconciliation disables it and does not mix desktop playback into the voice capture bus

### Requirement: Selectable monitoring
Monitoring SHALL offer Off, Pre and Post. Pre SHALL route the effective mic to the logical playback output; Post SHALL route AUX there only in enabled Element mode. Off, disabled voice, unavailable mic, and Post in Direct mode SHALL disable all managed mic/AUX hardware sends. The TUI SHALL explain inactive monitoring. Only one monitoring source SHALL be enabled.

#### Scenario: Playback moves
- **WHEN** monitoring is active and playback moves from A2 to A1
- **THEN** the old monitor send turns off before the selected source is enabled on A1

#### Scenario: Post selected in Direct mode
- **WHEN** the user selects Direct while Post monitoring is preferred
- **THEN** monitoring becomes inactive with an explanation and the preference remains Post

### Requirement: Optional app-playback rules
Each configured app-playback source SHALL have an enabled state. Enabled sources SHALL continuously follow the logical playback output; disabled sources SHALL have managed playback sends removed. Primary VAIO SHALL be enabled by default. Explicit rule state SHALL override migrated sends.

#### Scenario: Disable primary playback
- **WHEN** primary VAIO playback is disabled in live mode
- **THEN** its managed playback sends turn off and device migration does not reactivate them

### Requirement: Independent capture and playback availability
A missing playback device SHALL suppress monitoring and report playback unavailable without suppressing valid voice-to-app routing. Invalid snapshots SHALL cause no writes and SHALL NOT be treated as missing devices.

#### Scenario: Headphones disconnect with no playback fallback
- **WHEN** no playback candidate remains but the mic and mixer are available
- **THEN** the voice path remains managed and monitoring is off

### Requirement: Verified mode transitions
Source, voice-mode and monitoring transitions SHALL disable incompatible owned routes and verify those writes before enabling replacement routes. Failed writes or unexpected state changes SHALL stop the pass, report partial application and replan from fresh state. Stable state SHALL produce no redundant writes.

#### Scenario: Disable fails
- **WHEN** disabling direct mic-to-B3 times out while switching to Element
- **THEN** AUX-to-B3 is not enabled and the TUI reports the failed transition

#### Scenario: Interrupted transition
- **WHEN** the application restarts after old sends were disabled but before new sends were enabled
- **THEN** a new live session reconciles fresh observed state to saved intent without enabling competing paths

### Requirement: Truthful TUI rule controls
The persistent TUI SHALL expose source, voice enablement, Direct/Element, Off/Pre/Post and individual app-playback toggles. It SHALL show preferred and effective source, desired and observed routes, and disabled, inactive, pending, applied or error state with reasons. Dry-run controls SHALL update the preview without mixer writes. Live permission SHALL remain explicit.

#### Scenario: Toggle in dry-run
- **WHEN** a rule is changed in a dry-run TUI
- **THEN** its saved intent and preview change but the mixer remains untouched and the UI does not label the route applied

### Requirement: Persistent rule intent
Explicit selections SHALL persist per configuration independently of device patterns. Missing saved state SHALL use profile defaults. Invalid or incompatible saved state SHALL block live writes until repaired or explicitly reset. Failed saves SHALL leave prior active intent intact. Reload SHALL validate configuration and saved intent together before replacing active state.

#### Scenario: Restart
- **WHEN** the TUI restarts after selecting lav and Direct
- **THEN** those preferences are restored while the session defaults to dry-run unless explicitly started live

#### Scenario: Unwritable state
- **WHEN** saving a new selection fails
- **THEN** the TUI reports the failure and does not apply that selection to the mixer

### Requirement: Host health is not inferred from silence
The system SHALL distinguish mixer route verification from end-to-end audio verification. Element mode SHALL NOT imply a healthy host or automatically fall back to dry voice based on silence or process presence. Direct mode SHALL remain available for explicit recovery.

#### Scenario: Element closes
- **WHEN** Element stops while Element mode remains selected
- **THEN** no automatic dry mic route is enabled and the UI identifies the external processing path as not audio-verified
