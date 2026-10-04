## Why
Monitor changes repeatedly enumerate hardware and verify unchanged operations, causing avoidable delays. Users need explicit Pre-VST/Post-VST labels and a recorded snippet feed to Element for tuning effects.
## What Changes
- Monitor labels Off, Pre-VST, Post-VST (saved values unchanged).
- Recording to VST rehearsal toggle, Play/Stop actions and Loop setting for the loaded native recording.
- Send-only debounce 20 ms, short parameter verification polling, no repeated hardware enumeration inside numeric-only transactions.
## Capabilities
### New Capabilities
- `vst-rehearsal`: Recorded snippet routing and responsive monitor updates.
### Modified Capabilities
None.
## Impact
Config intent, native snapshot/recorder allowlists, routing, controller, TUI, tests and documentation.
