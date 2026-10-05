# Desktop controls design

## Platform and lifecycle
Wails v2 with the installed Windows WebView2 runtime; embedded HTML/CSS/JavaScript, no frontend framework or bundler. The same snoofer.exe launches its private __controls child using existing inherited pipes. Only the child owns the GUI; closing it releases its webview and leaves the host running. Existing tray/deck open controls focuses or opens the child. No HTTP listener, cloud service, or native audio access in the renderer. Preserve canonical build script/output; add Wails production tags there. No silent toolchain installation.

## Workspace
Dark neutral surfaces, cyan active states, amber pending/fallback, red errors/mutes, green-to-red meters. Visible keyboard focus is independent of state. Normal HTML controls supply tab navigation, text editing and paste. Sidebar uses click or Up/Down; grid arrows follow physical positions. Responsive workspace uses the full window, with stacked cards on narrow widths. Initial size 1280x820; usable minimum 800x600.

Audio: master stack and mutes, live mixer strips, Normal and VR cards with per-section override indication, recording configuration and advanced policy controls. Normal remains editable under VR.
Soundboard: artwork clip grid, search, playback state, stop and gain.
Stream Deck: page/device selectors, 9x4 physical key grid, five assignable dials plus reserved page dial, binding inspector with compatible searchable actions, shared/automatic markers and persistent save/discard bar. Select positions without triggering their actions. Page name/add/delete/reorder/Home remain available. Draft edits preserve existing plugin validation and atomic save behavior.
Plugins: lifecycle states, enable/disable confirmation and fallback forms for third-party controls.
Diagnostics: health, untruncated details, retry.

## Data and correctness
Keep control IDs, revisions, operation validation and provider ownership. Add one optional opaque JSON ViewData payload to a control for a plugin-owned structured presentation; core clones it and does not understand its schema. Stream Deck publishes its draft preview, selected position, dirty state and ownership markers on its existing preview control. GUI controls consume current snapshots; edits preserve their starting revision, never retry failed actions automatically. Read-only rows cannot dispatch. All text uses DOM textContent; artwork is restricted to embedded PNG data. Pending dispatch is distinct from observed state. Repeated updates must not steal focus, erase edits, or reset search/scroll. Expired meters show N/A.

## Validation and commits
1. Design and dependency feasibility.
2. GUI child bridge, lifecycle and structured deck snapshot, with focused tests.
3. Purpose-built screens and browser workflow checks.
4. Build, native lifecycle checks, live visual review, docs and removal of obsolete TUI code once coverage is established.
Do not claim physical/audio acceptance from mocked browser tests. Keep disabled plugins unloaded and preserve personal configuration.

