## Why
Element routing currently assumes the host is running. Closed or unobservable Element should not leave the microphone routed into an unavailable processor.
## What Changes
Observe element.exe through Windows process enumeration, retain requested settings and derive safe effective routing. Element preference falls back to Direct; Post monitor/capture effectively use Pre, and rehearsal pauses until the process returns. Expose the fallback in TUI. Preserve Post preferences in explicit Direct too; show per-control yellow pending and red override indicators.
## Capabilities
### New Capabilities
- `element-availability`: Process-aware effective processing mode and restoration.
### Modified Capabilities
None.
## Impact
Model observation, Windows adapter, planner, recorder guards, TUI and tests. No process launch/termination or new dependencies.
