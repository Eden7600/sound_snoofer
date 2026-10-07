## ADDED Requirements
### Requirement: Tape routing survives reconciliation
While tape playback started by Snoofer's Play is in effect, every routing plan SHALL keep the tape send on the Playback destination's bus, regardless of profile resolution.

#### Scenario: Play with profiles configured
- **WHEN** Normal microphone profiles are configured and the user presses Play
- **THEN** the tape send is set, and later reconciliations plan no change to it while the tape plays or is paused

#### Scenario: Convergence during playback
- **WHEN** a routing apply completes while the tape plays
- **THEN** the convergence check includes the tape send and does not report "routing has not converged"

#### Scenario: Stop
- **WHEN** the tape stops
- **THEN** the next reconciliation clears the tape send
