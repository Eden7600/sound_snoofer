## ADDED Requirements
### Requirement: Queue cleanup on neural restart
Every neural timeline restart SHALL discard previously published output on the consumer thread before pre-roll, without resetting producer-owned indices or replaying obsolete audio.
#### Scenario: Repeated warm-up resets
- **WHEN** lifecycle notifications, explicit retries or callback gaps repeatedly restart neural processing before output consumption begins
- **THEN** obsolete results do not accumulate to queue overflow and sustained valid pairs resume Active
#### Scenario: Concurrent old result
- **WHEN** an old worker result is published during reset
- **THEN** generation validation prevents it from being rendered as current audio
