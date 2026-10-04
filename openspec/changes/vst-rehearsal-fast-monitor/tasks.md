## Implementation
- [x] 1. Implement monitor labels and fast parameter-only verification with latency regression coverage.
- [x] 2. Implement persistent rehearsal/loop routing and guarded playback transport with tests.
- [x] 3. Run regression tests, vet, Windows build, strict validation and isolated TUI smoke; document results.
## Hardware acceptance
- [ ] 4. Verify actual snippet audio, hotplug recovery and live monitor-switch latency with user hardware.

Verification: go test ./... -timeout 30s, go vet ./..., Windows build and strict OpenSpec validation passed. Isolated preview TUI verified both monitor labels, rehearsal/loop persistence, action scrolling and rejection of Play/Stop in preview. Read-only native probe measured full inventory reads at 78.08/78.23/80.30 ms; parameter reads were below displayed timer resolution. No live mixer writes or listening acceptance performed. No new concurrency was introduced.
