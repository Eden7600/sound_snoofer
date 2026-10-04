## Implementation
- [x] 1. Add process observation and effective intent routing with fallback and restoration tests.
- [x] 2. Add TUI status and transport guards; verify suite, vet, build, strict validation and preview.
## Hardware acceptance
- [ ] 3. Verify live Element close/reopen audio continuity.

Verification: full Go suite, vet, Windows build, strict OpenSpec validation and diff checks passed. Native process observation reported known=true/running=false with Element closed. Isolated preview confirmed red Element/Direct and Post/Pre overrides, yellow queued changes, selected-row readability, and persisted Post values after explicitly selecting Direct. Tests cover process errors/name matching, closure/restoration, Off, rehearsal detachment, no automatic transport, active-capture guard, transaction invalidation and row-specific status clearing. No live audio changes or close/reopen listening acceptance performed.
