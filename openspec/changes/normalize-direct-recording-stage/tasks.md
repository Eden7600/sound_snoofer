## Current behavior and supersession

The element-process-fallback and sound-snoofer-default-launch changes supersede historical Direct/Post normalization, no-dry-bypass, and preview-default statements below. Current behavior preserves Post preference, uses effective Pre in Direct, automatically falls back to Direct when Element is unavailable, and defaults to live. Historical checkboxes are evidence only, not instructions to restore superseded behavior. Physical acceptance remains pending. The harden-application-consistency change adds reset/discard confirmation without replaying transport.

## Implementation
- [x] 1.1 Normalize Direct/Post across config, persistence and TUI and add 100 ms routing-only debounce; update affected expectations and docs.
- [x] 1.2 Test normalization, persistence and control behavior; run tests, vet, build, strict validation and preview interaction smoke.
