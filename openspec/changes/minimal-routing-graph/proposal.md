## Why
Routing, Devices and Events expose implementation detail and clutter the passive application's controls.
## What Changes
Replace those three tabs with a single read-only ASCII Graph tab alongside Controls. Remove their renderers, obsolete text-controls renderer and event-history collection/IPC field. Retain device discovery, routing plans and immediate errors needed by active controls and the audio worker.
## Impact
TUI navigation, graph rendering, worker presentation state, tests and current user documentation. No audio-routing policy or CLI diagnostic command changes.
