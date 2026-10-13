# Design
## 1. Models
`internal/streamdeck` owns a table of supported models. Hardware facts live only there; the plugin reads geometry from it.

| Model ID | Name | PIDs | Keys (cols × rows) | Key px | Key image | Dials | Strip / screen | Paging |
|---|---|---|---|---|---|---|---|---|
| `plus-xl` | Stream Deck + XL | 00C6 | 9 × 4 | 112 | rotated 90° CCW (as today) | 6 | 1200 × 100, cmd 0x0B, rotated (as today) | dial 6 |
| `plus` | Stream Deck + | 0084 | 4 × 2 | 120 | upright | 4 | 800 × 100, cmd 0x0C | dial 4 |
| `xl` | Stream Deck XL | 006C, 008F | 8 × 4 | 96 | rotated 180° | 0 | none | Pages key |
| `mk2` | Stream Deck MK.2 | 006D (Original v2), 0080, 00A5 | 5 × 3 | 72 | rotated 180° | 0 | none | Pages key |
| `neo` | Stream Deck Neo | 009A | 4 × 2 | 96 | rotated 180° | 0 | info screen 248 × 58, cmd 0x0B, rotated 180° | touch buttons |

Original v2 shares the MK.2 geometry and protocol, so it uses the `mk2` layout. Model names are display text; model IDs are persisted.

**Protocol (all models):** input report `01 <cmd> <len16>`. Keys `cmd 0`: one byte per key, then for the Neo two touch-button bytes (indexes 8 and 9). Dials `cmd 3`: `00` press states or `01` signed deltas, one byte per dial. Touch `cmd 2` stays unbound. A payload must hold at least the model's count; extra bytes are ignored. Key images are JPEG in output reports `02 07 <key> <last> <len16> <chunk16>`. The + XL strip and Neo screen use `02 0B 00 <last> <len16> <chunk16>`; the + strip uses the 16-byte header `02 0C <x16> <y16> <w16> <h16> <last> <chunk16> <len16> 00` with the full 800 × 100 region. Brightness is feature report `03 08 <percent>` on every model. Report lengths still come from the HID descriptor (the + reports input 512, output 1024, feature 32).

**Rendering:** keys are still drawn on the 112 px design canvas (4× supersampled), so the key typography, icons and visual baseline are unchanged. The averaged image is resampled to the model's key size (CatmullRom) only when it differs from 112, then oriented and JPEG-encoded. The + XL output is byte-identical. Text on 72 px keys is about 64% of + XL size; the design accepts this rather than maintaining a second key typography. Strip panels stay 200 × 100, one per dial, with the last being the page dial. The Neo screen shows the current page name on the page dial's styling.

## 2. Several decks
`StartSurface` returns a surface whose discovery loop (every 2 s) enumerates supported PIDs. It opens each HID path not already in a session and starts one session goroutine per deck. Sessions are keyed by device ID: the serial, or `path:<hid path>` when a deck reports none. A deck without a serial works with its model layout but cannot be remembered.
- `Show(id, Frame)` keeps the latest frame per deck (bounded, latest wins); the plugin addresses decks by ID.
- Events carry the device ID, serial and model ID. `Connected` reports a new session; `Error` with a device ID ends that deck's session (unplug, read or write failure), and discovery may reopen it. An error without a device ID is a discovery failure.
- The surface owns its sessions and waits for all of them; `Done` closes after every session has cleared its display and closed its handle. Each session keeps today's bounded clear-on-shutdown.

## 3. Layouts
`Page.Keys`, `Page.Dials`, `Layout.SharedKeys` and `Layout.SharedDials` become slices sized by the layout's geometry: keys = columns × rows; user dials = physical dials − 1, since the last dial pages. On dial-less decks the bottom-right key is reserved: it can hold no binding, and no region may cover it. Saved arrays of the + XL decode unchanged. Shorter lists are padded with empty bindings on load; longer ones, out-of-range regions and bindings on the reserved key are rejected. Regions and the region editor use the layout's column count instead of 9. Legacy `auto_controls` covers every non-reserved key.

**Where layouts live (settings, additive):**
- `layout`: the + XL layout (legacy key, unchanged).
- `models`: `{model ID: layout}` for the other models. A missing entry means the shipped default for that model, which is used until the user saves an edit.
- `serials`: `{serial: layout}` per-deck overrides (unchanged shape). An override's model is its device record's model; a legacy override without one is the + XL.
- `devices`: `{serial: {model, name}}` remembered decks.
- `paging`: `independent` (default) or `mirrored`.

A deck uses its serial override if it has one, otherwise its model layout.

**Shipped defaults.** Same page IDs on every model so mirroring lines up; positions follow the + XL frame (fixed controls on an edge, content in regions, Playback and Mic on dials 1–2).
- **+ / Neo (4 × 2):** Home: Mic Mute, Speaker Mute, Monitor, Record / Go to Media, Soundboard, Lights, Meetings. Soundboard: clips in columns 1–3, with Stop and Down in column 4. Lights: Room, Brightness, Motion, Sort / scenes. Media: sessions and apps in columns 1–3, with Up and Down in column 4. Meetings: Mute, Deafen, Camera, Leave. + dials: Home has Playback, Mic and Media; Soundboard has Volume; Lights has Playback, Saturation and Brightness; Media has Playback, Media and Focused App; Meetings has Playback and Mic.
- **MK.2 (5 × 3):** Home: Mic Mute, Speaker Mute, Monitor, Mic Path, Record / Previous, Play/Pause, Next, Brightness, Motion / Go to Meetings, Media, Soundboard, Lights, Pages. Soundboard: clips in columns 1–4, with Stop and Down in column 5. Lights: Room, Brightness, Motion, Sort, Sync / scenes. Media: sessions and apps on rows 1–2 in columns 1–4, with Down at column 5; Previous, Play/Pause, Next and Mute on row 3. Meetings: Mute, Deafen, Camera, Screen Share, Leave.
- **XL (8 × 4):** the + XL pages without column 9. Home drops Reset Focus, Brightness and Motion (no dials), and row 4 is Previous, Play/Pause, Next, then Go to Meetings, Media, Soundboard and Lights. Soundboard has Stop, Overlap and Down in column 8. Media has Up and Down in column 8. Routing is unchanged.
- Routing ships only on the XL and + XL; its matrix needs five columns and four rows.

## 4. Per-deck state and paging
The plugin keeps one state per connected deck: device ID, model, its page, its displayed expanded layout and page, generation, held key, dial meters and connection link. The rules for input from a deck are unchanged: stale-generation rejection, holds, dark and inert behavior, and dispatching the displayed binding. They now apply against that deck's state.
- **Independent:** navigation changes only that deck.
- **Mirrored:** navigation sets the base page (ignoring overflow sets) on every connected deck whose layout has it; each of those decks shows the first set of that page. Decks without that page stay where they are. Home goes Home everywhere.
- **Page dial** (+ and + XL): turn steps pages; press goes Home (unchanged).
- **Pages key** (XL, MK.2): shows the current page name with the page icon. Press steps to the next page and wraps; hold for 500 ms goes Home. It is drawn and handled by the plugin and is never a binding.
- **Neo:** left touch button steps to the previous page and right to the next; holding either goes Home. The info screen shows the current page name.
- Go-to and scroll requests from the GUI act on the deck that was used most recently (or on every deck when mirrored). Their published `Here` and set values describe that deck.

## 5. Remembered decks and reports
When a deck with a serial connects and has no device record, the plugin saves `{model}` for it with the existing settings save. This one write is the only one connecting causes; a failed save is reported in layout status and retried on the next connection. Renaming, Own Layout and Forget are explicit editor commands:
- **Name:** text; empty clears it.
- **Own Layout:** On copies the deck's model layout into `serials`; Off removes the override after GUI confirmation.
- **Forget:** removes the record and override after GUI confirmation; a connected deck is recorded again on its next connection.

Each connected or remembered deck publishes a connection report `streamdeck.device-<serial>` (or the path-based ID). The report is labeled with the deck's name or model name and shows the model, serial, USB ID, reconnects and display state. With no decks known, a single `streamdeck.app-device` report shows Not connected, as today.

## 6. Editor
`streamdeck.profile` lists targets as `model:<id>` (every supported model) and `serial:<serial>` (remembered decks with Own Layout). Option labels are the model name, or "<Name or serial> (<model name>)". The editor validates against the target's geometry. Slots number keys, then user dials, for that geometry. `EditorView` adds `Columns`, `Model`, `Reserved` (Pages key slot or −1), `NavDial` and `Decks`: remembered and connected decks, each with ID, name, model, Connected and OwnLayout. The editor also adds controls `streamdeck.deck` (the deck its commands apply to), `deck-name`, `deck-own`, `deck-forget` and `streamdeck.paging` (Independent / Mirrored; saved at once, not part of the draft). The GUI builds its grid from `Columns`, the key count and the dial count, shows the reserved Pages key or Pages dial as reserved, and keeps arrow navigation following the geometry.

## 7. Risks
- MK.2, XL and Neo orientation and the Neo screen and touch-button details come from published protocol notes and are untested here. Wrong orientation shows as rotated keys and is fixed in the model table alone.
- A + XL user with no `devices` entries keeps today's behavior. The first connection writes a device record, which is the only settings change on upgrade.
