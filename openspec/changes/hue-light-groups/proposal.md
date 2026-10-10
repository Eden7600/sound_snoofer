# Hue light-group dials
## Why
The Lights page has one Brightness dial for the whole room. The user wants to set decorative and mixed lighting separately, and to adjust colour saturation, without per-light controls or scene editing (the Hue app does those).

The bridge already reports each light's function (`metadata.function`: functional, decorative, mixed or unknown, set per light in the Hue app), its colour and colour mode, and gradient points; the plugin downloads but ignores them. Hue has no group for "the decorative lights in a room" and no saturation setting, so these dials write to each matching light, and the bridge rate-limits light writes. Turning therefore only stages a target, and a press applies it.

## What Changes
New dials for the selected room:

| Dial | Lights | Turn | Press |
|---|---|---|---|
| **Decor** (`hue.decor`) | Decorative | Stages a brightness | Applies the staged brightness, turning those lights on; with nothing staged, toggles them on and off |
| **Mixed** (`hue.mixed`) | Mixed | Stages a brightness | Same as Decor |
| **Saturation** (`hue.saturation`) | Every colour light showing a colour (not white-only lamps, not lights in white-temperature mode) | Stages a saturation | Applies it; does nothing with nothing staged |

- **Staging:** a staged target shows as the dial's value with an arrow (`→ 60%`). Nothing is sent until a press, and the stage is dropped after 10 s without input.
- **Hidden dials:** a dial with no matching lights in the room is blank, like Motion.
- **During Hue Sync:** the dials are inert while Hue Sync streams, because the stream owns the lights.
- **Saturation keeps hue:** each light's colour moves along its own line toward or away from white, so it keeps its hue; gradient strips change every point.
- **Layout:**
  - The default Lights page gains Decor on dial 3, Mixed on dial 4, and Saturation on dial 2 in place of Mic.
  - The user's layout is changed the same way, as requested.
- **Icons:** the dials reuse the sun (brightness) symbol; Saturation gets a drop symbol.

## Impact
- **Code:** `plugins/hue` (model fields, light groups, saturation, staged dials, paced per-light writes), `internal/streamdeck` (saturation icon, lights area), `plugins/streamdeck` (default Lights dials), `docs/ui-contract.md`, `docs/plugins.md`.
- **Unchanged:**
  - The room Brightness dial and its immediate writes.
  - Scenes, motion and Hue Sync behaviour.
  - The GUI's Lights screen; the dials are deck controls.
