## ADDED Requirements
### Requirement: Hold a clip to loop it
The soundboard SHALL play a clip that is held, rather than tapped, as a loop that repeats until it is stopped.

#### Scenario: Hold on the deck
- **WHEN** a clip key is held for 500 ms
- **THEN** the clip plays and restarts from the beginning each time it ends, and the key shows Loop

#### Scenario: Tap on the deck
- **WHEN** a clip key is released before 500 ms
- **THEN** the clip plays once, starting on release

#### Scenario: GUI hold and click
- **WHEN** a GUI clip button is held with the mouse or touch for 500 ms
- **THEN** the clip loops; a shorter click or Enter plays it once

#### Scenario: Stop
- **WHEN** Stop is pressed while a clip loops
- **THEN** the loop ends and no further iteration starts

#### Scenario: Replacement and overlap
- **WHEN** another clip is pressed while a clip loops
- **THEN** with Overlap off the loop is replaced; with Overlap on it keeps looping until Stop or until it is cut as the oldest of eight

#### Scenario: Routing lost or disconnected
- **WHEN** soundboard routing becomes unready or audio disconnects while a clip loops
- **THEN** the loop stops and does not restart when routing returns

#### Scenario: Restart failure
- **WHEN** restarting an iteration fails natively
- **THEN** the loop ends, the error stays on the clip and nothing is retried

#### Scenario: Preview, quit and relaunch
- **WHEN** preview is active, the plugin stops, or Snoofer relaunches
- **THEN** no loop plays and none resumes
