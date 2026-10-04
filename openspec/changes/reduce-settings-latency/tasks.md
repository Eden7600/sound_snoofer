## Implementation
- [x] 1. Complete audit and implement debounce/verification changes with regression tests.
- [x] 2. Verify full suite, vet, build, strict validation and preview startup; document practical latency limits.
## Hardware acceptance
- [ ] 3. Measure live device-switch time and audible continuity.

Validation: full go test ./... -timeout 30s, go vet, Windows build, strict OpenSpec validation and connected preview startup passed. Regression tests cover 999/1000 ms debounce, four full inventory reads for a single playback assignment, 20 ms pending parameter polls, and inventory mutation rejection before playback sends resume. Active config updated to 1000 ms. No live mixer writes or audible timing acceptance performed.
