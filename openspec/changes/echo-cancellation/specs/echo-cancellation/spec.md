## ADDED Requirements
### Requirement: Speaker echo cancellation
Snoofer SHALL cancel the speaker playback's echo from the managed microphone, using Voicemeeter's audio callback with no external application, driver or PATCH INSERT setup.

#### Scenario: Speakers in use
- **WHEN** playback goes to speakers, the mic stack is on and echo cancellation is Auto or On
- **THEN** the managed mic strip carries the echo-cancelled signal, and the status shows the echo removed and the estimated delay

#### Scenario: Headphones in Auto
- **WHEN** playback goes to headphones and the mode is Auto
- **THEN** the mic passes through unchanged, and the status says why

#### Scenario: Turned off
- **WHEN** the mode is Off
- **THEN** the input insert is unregistered, and the mic path is exactly as without the plugin

#### Scenario: Unsupported sample rate
- **WHEN** Voicemeeter runs at 44.1 kHz
- **THEN** audio passes through, and the status says echo cancellation needs 48 kHz

### Requirement: Real-time safety
The insert callback SHALL never block, lock or allocate after warm-up. It SHALL pass non-target channels through bit-identically, and SHALL fall back to pass-through on any engine error.

#### Scenario: Engine failure
- **WHEN** the engine reports an error during processing
- **THEN** the mic passes through unchanged from that buffer on, and the status shows the error

#### Scenario: New audio stream after a failure
- **WHEN** the engine failed and Voicemeeter then starts a new audio stream (engine restart, stream change or monitor restart)
- **THEN** the engine is reset once for that stream and resumes processing; a failure on the same stream stays latched with its reason shown
