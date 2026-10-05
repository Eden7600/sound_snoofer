## Why
Sound Snoofer should provide physical audio controls while remaining in the tray, without running Elgato software.
## What Changes
Direct USB HID support for the Stream Deck + XL, with A1/A2/mic gain knobs and observed feedback, native mute, recording controls and Windows media keys.
## Impact
A Windows HID adapter, configurable device layout, a bounded renderer and shared control-service client. Depends on mixer-surface-controls. VR preference buttons depend on vr-audio-policy. No Elgato plugin, SDK host, service or Python runtime.
