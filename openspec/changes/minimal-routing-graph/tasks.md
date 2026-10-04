## Implementation
- [x] Replace three tabs with Graph and remove obsolete renderers/history.
- [x] Render observed strips, buses, hardware, Element and recorder paths.
- [x] Update current user documentation.
## Verification
- [x] Graph and navigation regression tests; full tests and vet.
- [x] Strict OpenSpec validation and Windows build.
- [x] Isolated preview keyboard smoke test and automated resize/bounds checks.

Evidence: full Go suite and vet passed; strict OpenSpec validation passed. Built bin/sound-snoofer-graph.exe. An isolated PTY preview showed the current VAIO -> A2 speakers and B1 recording-mix branches, plus armed B1 -> stopped recorder. Tested tab cycling, picker cancellation, scrolling and Q. Width/height and sanitization cases passed automatically. A separate preview tray opened a controls child using the simplified private command; both were cleaned up. Live audio was not changed and the user's running app was preserved.
