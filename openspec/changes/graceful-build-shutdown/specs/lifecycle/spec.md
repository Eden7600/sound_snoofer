## ADDED Requirements
### Requirement: Graceful build shutdown
The canonical build SHALL request normal shutdown of this repository's running Snoofer and wait for exit before replacing binaries. It SHALL NOT forcibly terminate the audio owner.
#### Scenario: Running host
- **WHEN** a build begins with the repository host running
- **THEN** tray cancellation and plugin cleanup run before executable replacement
#### Scenario: Hung shutdown
- **WHEN** shutdown exceeds the deadline or normal close is unavailable
- **THEN** the build fails without terminating the host or replacing its executable
#### Scenario: Other processes
- **WHEN** another executable or another repo's Snoofer is running
- **THEN** it receives no shutdown request
### Requirement: Test process cleanup
GUI smoke tests SHALL close their private input pipes and await child exit even on assertion failures. Forced fallback SHALL be restricted to their audio-free controls children and reported as test failure.
#### Scenario: Failed GUI assertion
- **WHEN** the test exits early
- **THEN** its child receives EOF and is reaped before the script exits
### Requirement: Native close ordering
Normal audio cleanup SHALL stop monitoring before logging out of Voicemeeter and releasing its DLL, once per successful connection.
#### Scenario: Normal close
- **WHEN** the audio owner closes
- **THEN** monitoring shutdown precedes logout and DLL release; repeated close does not repeat them
