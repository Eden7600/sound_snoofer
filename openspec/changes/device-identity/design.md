# Design
## Configuration
- `Candidate` gains `id` and `name`. An entry has either `pattern` or `id`; an `id` entry requires `name`, which is a label kept for display while the device is disconnected.
- `ASIOInterface` gains `asio_id`/`asio_name` and `presence_id`/`presence_name` as alternatives to its patterns.
- `Output` gains `device_id`. `device` stays the label and the bus-ownership name.

## Endpoint inventory
- The Windows audio worker enumerates active render and capture endpoints every 5 s, and immediately when its request changes, whether or not default protection is on.
- `Result` carries the latest `Endpoints []Endpoint{ID, Name, Flow}`. A failed enumeration keeps the previous list and reports the error in diagnostics.
- The control worker stores the latest list in its state.

## Resolution
`config.Config.Resolve(names map[string]string) Config` returns a copy. It never mutates the saved base. The `names` map holds endpoint IDs plus ASIO CLSIDs (from Voicemeeter's inventory) mapped to current names. For each identity entry:
- **Resolved:** the runtime regex is `^` + QuoteMeta(name) + `$`.
- **Unresolved:** the regex matches the stored label, unless an available inventory device currently has that label, in which case it matches nothing.

For a slot with `device_id`, `Device` is replaced by the resolved name for planning. The worker resolves before every plan: the controller step, the UI plan and the priority view. Routing code is unchanged.

## Editor
- Suggestions gain `ID` when exactly one active endpoint of the right flow has that name, or the ASIO CLSID for drivers.
- **Add** saves `{driver, id, name}`.
- **Pattern** offers the existing Exact and Device regex forms.
- Identity entries render their name and a Device badge instead of a pattern field.
- Interface additions and the slot device picker save identity when available.
