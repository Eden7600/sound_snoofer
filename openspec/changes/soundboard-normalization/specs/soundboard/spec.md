## ADDED Requirements
### Requirement: Peak-normalized playback
Soundboard SHALL play cached PCM copies normalized to -1 dBFS sample peak with constant gain and unchanged source files.
#### Scenario: Non-silent clip
- **WHEN** a clip plays
- **THEN** its normalized sample peak is -1 dBFS within quantization tolerance, before the user's volume gain
#### Scenario: Silence
- **WHEN** a clip contains no signal
- **THEN** it remains silent
#### Scenario: Invalid input
- **WHEN** decoding fails or exceeds supported bounds
- **THEN** playback reports an error without falling back to unnormalized audio

### Requirement: Responsive preparation
Soundboard SHALL keep normalization cancellable and limited to one background job.
#### Scenario: Stop during preparation
- **WHEN** Stop is pressed while normalization is pending
- **THEN** preparation is cancelled and its completion cannot start playback
#### Scenario: Replacement
- **WHEN** another clip is requested while preparation is pending
- **THEN** only the latest request may play after preparation completes
#### Scenario: Cached or changed clip
- **WHEN** an unchanged clip is replayed or the source changes
- **THEN** the matching completed cache is reused or a new normalized copy is prepared respectively
#### Scenario: Shutdown
- **WHEN** the plugin stops
- **THEN** background work is cancelled and joined, partial files removed and native resources released
