## Why
Device changes still debounce for two seconds and force full enumeration for every send operation and every verification retry in the same transaction. Recorder transport polls at 100 ms despite using cheap parameter reads.
## What Changes
- Default device debounce becomes 1000 ms in parser, shipped profiles and active config.
- Mixed transactions use fresh parameters for numeric operations and verification polling, with full inventory before and after each actual device assignment and at transaction boundaries.
- Device verification polls at 20 ms; numeric and recorder verification at 5 ms.
## Capabilities
### New Capabilities
- `settings-latency`: Faster verified settings application.
### Modified Capabilities
None.
## Impact
Controller observation policy, config defaults, tests and performance documentation. Preserve saved choices, save-before-apply, failure backoff and hotplug checks.
