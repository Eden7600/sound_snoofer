## ADDED Requirements
### Requirement: Silence-aware automatic microphone selection
When `profiles.activity` is configured and the Normal source is Auto, Snoofer SHALL skip microphones latched Silent from Voicemeeter pre-fader levels. It SHALL return to a higher-priority microphone once that microphone is Active again.

#### Scenario: Dead lav battery
- **WHEN** the lav is first in priority, is in Auto, and stays below the silence threshold for the configured time
- **THEN** the desk mic becomes the effective source

#### Scenario: Lav returns
- **WHEN** the latched-silent lav delivers signal above the threshold for 300 ms
- **THEN** Auto returns to the lav

#### Scenario: Explicit choice
- **WHEN** the user explicitly selected the lav and it is silent
- **THEN** the lav stays selected

#### Scenario: Everything silent
- **WHEN** every option is latched Silent
- **THEN** the priority applies without regard to silence, and the source is not Off

#### Scenario: Muted mic
- **WHEN** the mic is muted
- **THEN** its pre-fader level is still measured, and mute alone never latches Silent

#### Scenario: Unreadable level
- **WHEN** a level read fails or Voicemeeter disconnects
- **THEN** the source is Unknown, not Silent, and selection treats it as eligible

#### Scenario: Activity not configured
- **WHEN** `profiles.activity` is absent
- **THEN** selection and wiring are unchanged

### Requirement: Wire only checked microphones
With activity configured, Snoofer SHALL wire only the first `check` available Normal priority options and the effective source. It SHALL clear other managed microphone patches and assignments.

#### Scenario: Check two of three
- **WHEN** the priority is lav, desk, webcam with `check` 2, and the lav is effective
- **THEN** the lav and desk ASIO pairs are patched, and the webcam input is cleared

#### Scenario: Fallback beyond the checked set
- **WHEN** the effective source is the webcam because the lav and desk are unavailable
- **THEN** the webcam is wired

#### Scenario: Mic stack disabled
- **WHEN** the mic stack is disabled
- **THEN** every managed mic patch and assignment is cleared as before, and activity is Unknown
