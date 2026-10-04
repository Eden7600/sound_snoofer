## Implementation
- [x] 1. Add ASIO fallback selection and configuration with routing/eligibility tests.
- [x] 2. Update defaults and active config, validate, build and preview the picker.
## Hardware acceptance
- [ ] 3. Verify audible Volt playback and physical disconnect/reconnect.

Full tests, controller migration regression, vet, Windows build and strict validation passed. Isolated dry-run picker showed Universal Audio Volt beside SteelSeries. Active bin/config.json enables ASIO playback; saved choices untouched. No live mixer writes.
