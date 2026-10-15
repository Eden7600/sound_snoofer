# Tasks

- [x] 1. Review and commit the finding, proposal, design and requirements; strictly validate before implementation.
- [x] 2. Add failing provider-to-tile and Routing regressions, move device detail into ViewData and update all callers/fixtures.
- [x] 3. Run focused Go/browser tests and frontend gates, inspect actual offline native-size rendering, and update the UI contract and finding status.
- [x] 4. After the user permits running-host changes, perform canonical build and packaged/hardware acceptance. Do not claim this from offline rendering.

2026-10-15 partial task 4 physical acceptance: a temporary Audit Test slot had no device
and all sends Off. Bound as `audio.slot-1` on Plus XL Home key 8, it showed No Device
without Unavailable; the user confirmed pressing it did nothing. Key 8 was restored to
Empty/Saved and removing the slot returned Processed/Reloaded. The base configuration
fingerprint returned to its original value; the user's enabled Mic Stack choice was preserved.
At that point only the no-device/inert case was verified. Healthy and disconnected captions and
In Use physical checks remained open, so task 4 remained unchecked until the later check below.

Later task 4 packaged acceptance: after explicitly selecting SteelSeries for Normal Playback,
the temporary Audit Test slot used Headphones (Cody’s AirPods Pro). Native Routing displayed
that caption, selected the same device and showed In use; all four sends remained Off.
The display-only audio.slot-1 binding was saved to XL Home key 8. The user confirmed cyan
Audit Test / In use without Unavailable. Closing the case changed the physical key to
Disconnected with a neutral background and white text; native Routing showed Disconnected with the AirPods
caption/current selection. Reconnecting returned both surfaces to In use. Key 8 was cleared
and saved, the slot removed, and Normal Playback restored to Automatic. The base configuration
fingerprint matched its original value. Mic Stack On, both mutes Off, Normal Monitor Off,
Mic Stage Pre and the existing VR playback choice were retained; native Audio showed
Applied / No stall detected. Together with the earlier canonical build and no-device/inert
acceptance, this completes task 4. No additional application build was needed for this
documentation-only checkpoint.
