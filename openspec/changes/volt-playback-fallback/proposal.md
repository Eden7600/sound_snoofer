## Why
Volt should be selectable for playback and serve as the lowest-priority automatic output.
## What Changes
Add studio.asio_playback, enabled in shipped voice defaults and the active config. Reuse connected Volt ASIO A1 for playback after all eligible WDM candidates, with explicit selection supported.
## Capabilities
### New Capabilities
- `volt-playback`: ASIO playback fallback and explicit selection.
### Modified Capabilities
None.
## Impact
Config, planner, playback options, shipped defaults, active config and documentation.
