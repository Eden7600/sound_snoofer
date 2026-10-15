# Display-only Stream Deck bindings

## Why

[Repository audit finding 32](../../../docs/repo-audit-minor.md) describes a startup-order
failure: the default Routing pages bind `audio.playback-device` and `audio.slot-*` status
controls, but layout validation requires a key to support `press` or `set`. If audio has
already published, Stream Deck rejects its shipped layout. The GUI fixture currently avoids
the failure by publishing audio only after the deck starts.

The provider's binding options and the GUI's compatibility filter apply the same restrictive
rule. The hardware input path also assumes a key can act, so accepting status bindings in
validation alone would make a press produce an unsupported-operation error.

## What Changes

- Treat `Kind: status` as a display-only key binding. Status controls cannot occupy dials;
  text controls remain incompatible with keys and dials.
- Apply the same compatibility rule in layout validation, provider editor options and the
  GUI binding list. Keep the existing rules for actionable controls and missing providers.
- Ignore hardware input on status bindings while retaining their displayed state, saved
  identity and existing hidden/unavailable rendering.
- Cover startup with audio already published, inert status input, and editor agreement.
  Generate the GUI fixture with fixed audio controls present before Stream Deck starts.

## Impact

- Code: `plugins/streamdeck/layout.go`, `plugin_windows.go`, their focused tests and
  `guifixture_windows_test.go`; `app/web/src/model.ts`, model/deck tests and the generated
  Stream Deck fixture.
- Clarify display-only keys in `docs/ui-contract.md` during implementation.
- Preserve default positions, user layouts, control IDs, saved settings, rendering style,
  navigation, automatic regions, save-before-apply and stale-input rejection.
- Implement after `gui-typescript-modules` lands; this proposal adds planning artifacts only.
