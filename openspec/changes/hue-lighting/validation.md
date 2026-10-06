# Validation

## Automated (2026-10-05)
- `gofmt -l` on all changed Go files: clean.
- `go vet ./...`, plus `go vet -tags no_hue` and `-tags no_huesync` for `./cmd/snoofer`: clean.
- `go test ./... -timeout 30s`: pass. `plugins/hue` and `plugins/huesync` also passed five consecutive runs (`-count=5`).
- `go test -race ./...`: not run. The race detector needs cgo, and no C compiler (gcc) is installed; the toolchain was left unchanged.
- `openspec validate hue-lighting --strict`: valid.
- `scripts/build.ps1`: built `bin/snoofer.exe`. The running Snoofer (tray and controls processes) exited gracefully, and the tray process was relaunched without the automation-only NO_COLOR value.

## Coverage notes
- Hue tests use an in-process TLS fake bridge (CLIP v2 resources, PUT, SSE) and cover: discovery outcomes (none, one, many, paired-ID match, address mismatch), certificate pin mismatch, link-button pairing and timeout, knob clamping and off-state rules, temperature Mixed/unknown and range intersection/union, coalescing of 10 ticks into at most 4 writes, 429 backoff with the latest value, unconfirmed-write errors, stream gap reload without replay, scene recall confirmation and timeout, scene add events, missing room, preview gating and stale revisions.
- Hue Sync tests use a loopback fake of the third-party control socket (protocol taken from the Hue Sync binary and the Elgato plugin source) and cover state events, toggling, mode/intensity only while syncing, inc_bri accumulation, confirmation timeout, bridge_disconnected, restart without replay, refused connections and preview.
- Renderer output was inspected at native size: key sheet `docs/design/hue.png`, plus a dial strip preview (`SNOOFER_HUE_DIAL_PREVIEW`). The inspection found and fixed a missing `%` glyph and icon identifiers printed on non-page dials.

## Not yet verified
Hardware acceptance tasks 8–11: real bridge pairing and mDNS discovery through the Windows firewall, scene recall, dial responsiveness at the 250 ms write gap against the bridge rate limit, and Hue Sync with Third-party control enabled (currently off in the local Hue Sync config).
