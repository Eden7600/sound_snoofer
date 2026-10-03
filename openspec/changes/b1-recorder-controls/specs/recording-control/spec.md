# Recording control

## Purpose

Choose a B1 recording mix and operate Voicemeeter's integrated recorder from the persistent TUI while preserving voice delivery and playback rules.

## ADDED Requirements

### Requirement: Dedicated recording mix
An optional recording profile SHALL require Potato voice routing and own all strip-to-B1 sends. Mic and computer inclusion SHALL be independent toggles, initially off. Unselected strips SHALL not feed B1. Without this profile, B1 and recorder settings SHALL remain unmanaged.

#### Scenario: Computer only
- **WHEN** computer recording is enabled and mic recording is disabled
- **THEN** the configured computer sources feed B1 and every mic and AUX return B1 send is off

### Requirement: Selected microphone tap
Mic recording SHALL use the effective selected microphone for Pre and AUX for Post. Pre/Post SHALL mean before/after Element, not pre/post Voicemeeter fader. Post SHALL require enabled Element voice mode; otherwise its preference is retained with an inactive reason and no dry substitution. Disabled voice or unavailable microphone SHALL suppress mic recording.

#### Scenario: Switch to Direct
- **WHEN** Post recording is selected and voice changes to Direct
- **THEN** AUX-to-B1 turns off, no dry B1 route is substituted and computer capture can continue

#### Scenario: Mic disconnect and ambiguity
- **WHEN** the preferred interface disappears and no unique eligible fallback exists
- **THEN** mic B1 sends turn off without treating ambiguous devices as selected

#### Scenario: Reconnect
- **WHEN** the preferred microphone returns after webcam fallback
- **THEN** recording follows the effective source after stable observations and the old mic send is disabled first

### Requirement: Independent computer capture
Computer recording SHALL use an explicit configured set of virtual input sources, defaulting to primary VAIO. AUX SHALL be rejected. Capture inclusion SHALL be independent of speaker-playback toggles and playback-device availability.

#### Scenario: Silent speakers
- **WHEN** app playback is disabled or all configured speakers/headphones are absent
- **THEN** enabled computer sources still feed B1

### Requirement: Recorder preparation
Before Start, the system SHALL verify bus recording mode, B1 as the sole armed bus, stereo non-multitrack recording and the selected route matrix. Recorder playback sends to B1/B2/B3 SHALL be off. Configuration changes SHALL occur only while stopped; active incompatible recorder configuration SHALL block preparation and be reported. Recording location and format SHALL remain managed in Voicemeeter.

#### Scenario: Recorder uses another source
- **WHEN** the recorder is active with an incompatible armed source
- **THEN** the application reports the conflict and does not silently reconfigure or restart it

### Requirement: Explicit transport
Start and Stop SHALL be explicit one-shot live actions serialized under writer ownership. Start SHALL require a fresh valid recorder snapshot, verified routing, at least one eligible enabled source and no pending transition. Stop SHALL remain available despite routing conflicts. Dry-run SHALL make no transport writes. Repeated Start while recording SHALL not toggle recording or pause it.

#### Scenario: No recording sources
- **WHEN** Start is requested with both sources disabled or unavailable
- **THEN** no record command is sent and the reason is displayed

#### Scenario: Source changes during capture
- **WHEN** inclusion or tap changes while recording
- **THEN** the selected mix changes through verified transitions without stop/start commands or an implied new file

### Requirement: Uncertain outcomes and reconnect
Transport SHALL be verified through fresh recorder state. An uncertain or failed Start SHALL not be retried automatically. Reconnection SHALL observe existing state without auto-starting or replaying queued commands. Exiting or switching to dry-run SHALL leave recorder transport untouched and explain that an active recording continues in Voicemeeter.

#### Scenario: API disconnect after Start
- **WHEN** the record command is submitted but confirmation cannot be read
- **THEN** status becomes unknown, the command is not replayed, and reconnection observes the recorder

### Requirement: Visible controls and durable choices
The TUI SHALL show mic inclusion, Pre/Post tap, computer inclusion, recorder status, Start/Stop and blocked reasons. Source choices SHALL persist with backward-compatible loading of existing voice state. Transport state and commands SHALL never be persisted as intent. A failed save SHALL not change routes.

#### Scenario: Existing sidecar
- **WHEN** a valid older voice sidecar is loaded after adding the recording profile
- **THEN** existing voice choices remain and recording inclusion defaults off

#### Scenario: Restart while recorder is active
- **WHEN** Voice Snooter restarts
- **THEN** it restores source preferences, defaults to dry-run and displays the observed recorder state without starting or stopping it

### Requirement: Preserve other mixes during source changes
Recording-only changes SHALL preserve B2/B3 delivery and monitoring, except protected recorder playback sends. Old B1 mic sends SHALL be disabled and verified before replacement B1 sends are enabled. Failure SHALL stop later operations and expose partial state without claiming a valid audio file.

#### Scenario: Tap change fails
- **WHEN** disabling Pre-to-B1 fails during a switch to Post
- **THEN** AUX-to-B1 is not enabled and Discord's route is not unnecessarily gated
