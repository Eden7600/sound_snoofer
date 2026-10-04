## Implementation
- [x] 1.1 Implement shared eligibility and persistent preferred playback routing.
- [x] 1.2 Add dynamic mic/output pickers with refresh and worker confirmation checks.
- [x] 1.3 Test persistence, fallback, ambiguity and picker interaction; run regression tests, vet, build, strict validation and preview smoke.

Verification: regression suite, go vet, Windows replacement build, and strict OpenSpec validation passed. Isolated dry-run TUI showed Webcam/Off with Volt absent; Playback Device showed Automatic/SteelSeries, accepted a selection, and persisted the preference. Tests cover disconnect/reconnect, ambiguous devices, stale confirmation, draft selection and persistence. Physical hotplug/listening acceptance remains unperformed.
