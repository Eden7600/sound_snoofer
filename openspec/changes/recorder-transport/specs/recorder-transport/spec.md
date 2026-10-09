## ADDED Requirements
### Requirement: Tape-deck transport
The Record, Play/Pause, Stop, Rewind and Fast-forward commands SHALL act according to the recorder's state, using only the recorder commands valid in that state. Every write SHALL be one-shot and never retried.

#### Scenario: Pause playback
- **WHEN** the tape is playing and Play/Pause is pressed
- **THEN** Snoofer writes `Recorder.stop` once, verifies Stopped, and shows the tape as Paused

#### Scenario: Return to start after pausing
- **WHEN** the tape is paused by Snoofer's stop and Stop is pressed
- **THEN** Snoofer writes `Recorder.stop` once more, and the tape no longer shows as Paused

#### Scenario: Stop while playing
- **WHEN** the tape is playing and Stop is pressed
- **THEN** Snoofer writes `Recorder.stop`, verifies Stopped, and writes `Recorder.stop` again

#### Scenario: Pause and resume a recording
- **WHEN** a recording is in progress and Play/Pause is pressed
- **THEN** Snoofer writes `Recorder.pause` and verifies Paused, and pressing Play/Pause again writes `Recorder.record` and verifies Recording

#### Scenario: End a recording with Stop
- **WHEN** a recording is in progress or paused and Stop is pressed
- **THEN** the recording stops through the verified Stop path

#### Scenario: Winding during a recording
- **WHEN** a recording is in progress or paused
- **THEN** Rewind and Fast-forward are unavailable and refused

#### Scenario: Record during playback
- **WHEN** the tape is playing and Record is pressed
- **THEN** Snoofer stops playback and verifies Stopped before preparing and starting the recording, and a failed stop aborts the Start without retrying

#### Scenario: Write fails
- **WHEN** a transport write returns an error
- **THEN** the command reports that its outcome is unknown, and nothing is retried

#### Scenario: State changed elsewhere
- **WHEN** the tape is paused by Snoofer's stop and the recorder is then observed playing or recording
- **THEN** the tape no longer shows as Paused
