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

## Revision 2 (2026-10-05)
- Root cause of "No bridge": the single mDNS query used the OS default multicast route. On this host (Ethernet plus Tailscale, the WSL Hyper-V switch and four Wi‑Fi adapters) it did not reach the bridge network. A scratch diagnostic got the bridge's answer (172.16.102.3) when the query was sent explicitly on Ethernet. After the fix, `SNOOFER_HUE_LIVE=1 go test ./plugins/hue -run TestLiveDiscovery` finds `[172.16.102.3]`, and `/api/0/config` reports a BSB002 bridge.
- `go vet ./...` and `go test ./...`: pass. The hue package also passed `-count=3` after the merge. gofmt is clean for all changed files; `internal/voicemeeter/{client,studio}_test.go` were already unformatted at HEAD and were not touched.
- `node --test app/web/model.test.mjs`: pass. `node scripts/check-gui.cjs`: pass (Lights setup and Pair, scene grouping, dial buttons and slider ticks, sync disabled while not syncing, enable-only Plugins, narrow-width overflow). Reference image: `docs/design/gui-lights.png`.
- `node scripts/check-desktop.cjs` against the new binary: pass. The native checks now drive the Lights screen's Start sync toggle instead of the removed Plugins fallback form.
- `go test -race`: not run (no C compiler; toolchain unchanged). OpenSpec strict validation: valid. `scripts/build.ps1`: built; Snoofer exited gracefully.
- Personal config (`bin/snoofer.json`, not in git; backup kept in the session scratchpad): applied the Home Hue block (keys 5–8: Sync, Mode, Intensity, Brightness; rows 1–3 of columns 5–8: room scene slots 1–12) and dials 3–4 (Brightness, Temp), and removed the obsolete `huesync` entry. The target positions were empty before the change, no other settings changed, and `snoofer.exe --check` passes. Snoofer was relaunched without the automation-only NO_COLOR.
- Still requires hardware: pressing the bridge link button to pair, and every item in tasks 8–11.

## Decoding fix (2026-10-05)
- Pairing succeeded on the real bridge, but loading failed with "cannot unmarshal string into ... sceneStatus": `zigbee_connectivity` and `entertainment_configuration` use a string `status`. The fake bridge had only scene statuses, so the tests missed it.
- After the fix, `SNOOFER_HUE_CONFIG=bin/snoofer.json go test ./plugins/hue -run TestLiveResources` loads the paired bridge (172.16.102.3): 36 resource types, 6 rooms and 2 zones, 36 lights, 48 scenes. It prints resource counts only. New tests cover string statuses in loads and in the event stream, skipping of malformed unused types, and rejection of malformed used types.

## Revision 3: temperature removed (2026-10-05)
- The user found the temperature dial worked once the lights were in color-temperature mode, but decided to remove it. `go vet ./...`, `go test ./...`, the GUI model tests, `check-gui.cjs` and strict OpenSpec validation pass. The reference images `docs/design/hue.png` and `docs/design/gui-lights.png` were regenerated.
- Personal config, edited while Snoofer was stopped for the canonical build (backup in the session scratchpad): removed `hue.neutral_kelvin` and unbound Home dial 4 (`hue.temperature`). `snoofer.exe --check` passes; Snoofer was relaunched.
