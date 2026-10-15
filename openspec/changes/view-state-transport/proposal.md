# View-state transport

## Why
Repository audit finding 14 identifies repeated full GUI snapshots, including every control's base64 artwork. The tray sends them every 200 ms and the GUI polls Wails every 100–200 ms, even when the presentation has not changed. Control revisions alone cannot identify these changes: meter and progress updates intentionally retain their control revision.

## What Changes
- Suppress unchanged presentation frames within each controls-child session, while retaining telemetry-only changes and explicit focus requests.
- Move still artwork into a complete map sent only when changed, following the existing animation side channel. Preserve explicit empty-map clears and pending maps when the one-slot queue replaces a frame.
- Return no state from unchanged Wails polls. Changed responses carry the latest complete stripped state and artwork only when the caller needs it. Reconstruct the existing frontend control shape centrally.
- Detect complete animation changes, including later frames and delays, and refresh affected animation playback through a small animation generation value.
- Keep local polling updates, reconnection snapshots, legacy native smoke frames and provider snapshot ownership intact.

## Impact
Implementation is confined to `app/transport.go`, controls-child publishing and desktop state, shared frontend state/host/widgets/polling, and focused tests. `snoofer.Control`, provider APIs, action revisions, dispatch and persistence behavior remain unchanged. No dependencies or new background workers are needed.

Configuration loading and login-startup observation remain at their current cadence. A reliable external-file invalidation mechanism is outside this transport change; caching only internal saves or file timestamps could hide external edits. GUI action outcomes and broader local-state preservation remain separate changes. The user-owned `scripts/check-desktop.cjs` is exercised without modification.
