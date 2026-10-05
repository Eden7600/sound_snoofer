## ADDED Requirements
### Requirement: Live knob levels
Bound audio gain dials SHALL retain their gain readout and display a segmented digital level bar derived from the corresponding output bus or active microphone input.

#### Scenario: Gain dial signal activity
- **WHEN** a gain dial has a fresh successful level reading
- **THEN** its bar displays that signal on a -60 to 0 dBFS scale with green, amber and red ranges without changing the gain setting.

#### Scenario: Unavailable or stale signal
- **WHEN** a level read fails, its source is unavailable, or telemetry is older than 500 ms
- **THEN** the dial shows an unavailable meter rather than retaining a live bar or claiming silence.

### Requirement: Nonintrusive telemetry
Meter polling SHALL remain on the serialized native worker, preserve routing polling and command revisions, and avoid rerendering unchanged key graphics.

#### Scenario: Meter-only update
- **WHEN** levels change while controls and bindings remain the same
- **THEN** pending knob input remains valid and unchanged key images are reused.

#### Scenario: Plugin shutdown
- **WHEN** audio or Stream Deck is disabled or stopped
- **THEN** its polling or display work respectively ends with its existing lifecycle.
