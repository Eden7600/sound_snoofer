# Design

## Root cause and existing paths

`startWithSurface` validates every stored layout against the current control registry.
`Layout.Validate` treats a key as actionable only, although the + XL and XL default Routing
pages intentionally include destination status keys. A missing provider passes validation,
so the same saved layout starts or fails depending on publication timing. Saving another
binding can fail for the same reason once audio is present.

Compatibility is repeated in `Layout.Validate`, the `streamdeck.binding` options assembled
by `plugin_windows.go`, and `compatible` in `app/web/src/model.ts`. The GUI independently
builds its binding list, so changing only the Go options will not expose status choices.

The hardware event handler creates a `press` request by default. The registry refuses a
status control with no operations and that error becomes the layout editor status. Status
input must be stopped before it creates or queues a request.

## Compatibility rule

Use a small package-local Go predicate for the two Go consumers, and mirror its cases in
the existing TypeScript `compatible` helper:

| Control | Key | Dial |
| --- | --- | --- |
| `Kind: text` | Reject | Reject |
| `Kind: status` | Accept for display only | Reject |
| Other control with `press` or `set` | Accept | Only if it also has `adjust` |
| Other control with only `adjust` | Reject | Accept |
| Other control with no supported operation, including connection reports | Reject | Reject |

Compatibility does not depend on availability: an unavailable or hidden status binding
remains valid and retains its ID. Missing providers remain valid under the existing layout
rules. Preserve the provider and GUI filters that exclude Stream Deck editor internals;
go-to, scroll and Reset Focus controls keep their existing exceptions.

Keep full-layout validation at startup, assignment and Save. Do not defer validation,
reorder plugin startup, add dummy operations, remove the default status keys, or migrate
saved layouts. Automatic key/dial regions retain their existing operation-based filters.

## Inert hardware input

Recognize a displayed `Kind: status` control before building a request or recording a
pending hold. Key down, release and a long hold must dispatch nothing and must not produce
an unsupported-operation error. Keep current generation, hidden, inactive and unavailable
guards, plus the normal command, selection, hold and navigation paths.

Use the existing status rendering unchanged: visible keys continue to show the provider's
label/value/status, while hidden bindings stay blank. Clarify this behavior in the UI
contract without changing positions, icons, palette or labels.

## Regression coverage and fixture

- Table-test compatibility in Go and the TypeScript model, including status keys/dials,
  text with operations, existing actionable controls, connection reports and missing IDs.
- Validate shipped layouts for all supported models against the fixed generated audio
  controls. Include the + XL and XL Routing status keys explicitly.
- Start the real Stream Deck plugin through its existing fake surface with audio controls
  published first; require startup and editor publication to succeed. Retain coverage for
  providers appearing after startup. Use fixed fixture data and no live integrations.
- Through the fake hardware surface, show a status key and an actionable key. Exercise
  status down/release/hold with no provider calls or editor error, verify state refresh still
  reaches rendering, and verify the actionable neighbor still dispatches.
- Check the provider's binding options and GUI list include status controls for a key,
  omit them for a dial, and still omit text/connection controls. Assign and save a status
  key without disturbing other bindings.
- Move audio publication before startup in `TestGUIFixture`, remove its workaround comment,
  regenerate the fixture, and review the compatibility-driven option changes.

Run focused Go tests, fixture drift checks, model/deck browser tests and the existing GUI
suite. Inspect a native-size rendered status key using the existing renderer/test surface;
leave physical hardware acceptance unclaimed unless it is performed. No new dependency or
general binding framework is needed.
