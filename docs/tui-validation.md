# TUI validation - 2026-10-03

The Windows PTY smoke test connected read-only to the current Potato instance (edition 3). The Routing view showed Volt ASIO on A1, Arena playback on A2, paired ASIO patches, and primary VAIO at strip index 5. Tab navigation displayed the device inventory and events view. Reload reported success while monitoring continued. Pressing q returned exit code 0 and emitted alternate-screen exit/cursor-restoration sequences.

No live toggle or mixer-setting writes were performed during this test. Worker tests separately exercise live ownership, conflict handling, config reload failures, connection retry, and cleanup. Model tests cover constrained layout, controls, and sanitizing terminal text. Noninteractive invocation returns an actionable error. Existing audio/hardware acceptance remains pending under automatic-device-routing.
