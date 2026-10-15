# Tasks

- [x] 1. Review and commit the finding, proposal, design and requirements; strictly validate before implementation.
- [x] 2. Add failing provider-to-tile and Routing regressions, move device detail into ViewData and update all callers/fixtures.
- [x] 3. Run focused Go/browser tests and frontend gates, inspect actual offline native-size rendering, and update the UI contract and finding status.
- [ ] 4. After the user permits running-host changes, perform canonical build and packaged/hardware acceptance. Do not claim this from offline rendering.

2026-10-15 partial task 4 physical acceptance: a temporary Audit Test slot had no device
and all sends Off. Bound as `audio.slot-1` on Plus XL Home key 8, it showed No Device
without Unavailable; the user confirmed pressing it did nothing. Key 8 was restored to
Empty/Saved and removing the slot returned Processed/Reloaded. The base configuration
fingerprint returned to its original value; the user's enabled Mic Stack choice was preserved.
Only the no-device/inert case is verified. Healthy and disconnected device captions and
In Use physical checks remain open, so task 4 stays unchecked.
