## ADDED Requirements
### Requirement: Pausing device and send writes
Snoofer SHALL let the user separately disable its device-assignment writes and its routing-parameter (send) writes, persist both choices, and keep planning and reporting while paused.

#### Scenario: Devices paused
- **WHEN** device manipulation is disabled and the plan wants a different device on a slot
- **THEN** no device assignment is written, sends are not gated for that change, and the toggle shows the held count

#### Scenario: Sends paused
- **WHEN** send manipulation is disabled and a strip send, patch, routing mute or recorder routing value differs from the plan
- **THEN** none of those parameters is written, tape-playback protection does not run, and the toggle shows the held count

#### Scenario: User actions still apply
- **WHEN** either switch is on and the user changes a gain, Playback or Mic mute, or starts/stops the recorder
- **THEN** that explicit action is applied as usual, except that recorder Start fails with "Sends paused" if the recorder's routing would need changing

#### Scenario: Resume
- **WHEN** the user turns a switch off
- **THEN** the next reconciliation applies the held changes normally, including the voice transition

#### Scenario: Restart
- **WHEN** Snoofer restarts with a switch on
- **THEN** the switch stays on and the header shows the paused badge
