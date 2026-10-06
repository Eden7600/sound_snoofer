# Design

Positions are given as row/column on the 4×9 key grid (r1c1 is top-left). Dials are numbered 1–6; dial 6 stays reserved for pagination.

## 1. Hue motion toggle
**Bridge facts.** These come from a read-only probe of the user's bridge on 2026-10-06:
- Each sensor exposes a `motion` service with `owner` (its device) and a writable `enabled` flag. This is the flag behind the Hue app's per-sensor on/off switch.
- Sensor devices are children of their room. The bridge has three: Guest Room, Cody Bathroom and Cody Office.
- Motion automations are `behavior_instance`s that reference the `motion` service.

**Association.** A room's motion sensors are the `motion` services whose owner device is in `room.children`. Zones have none. `grouped_motion` is not written; its writability is undocumented.

**Control `hue.motion`.**
- Label `Hue motion sensors`, ShortLabel `Motion`, Kind `toggle`, Icon `hue-motion`.
- **Value:** `On` when every sensor is enabled, `Off` when none is, and `Mixed` otherwise.
- **Hidden** (a blank deck key) while the selected group has no sensors. When the bridge isn't connected, the key shows `N/A`.
- **Press:** if the value is `On`, disable every sensor; otherwise enable every sensor. The bias favors restoring automations.
- **Writes:** `PUT /resource/motion/{id} {"enabled": bool}` per sensor, serialized on the Hue worker.

**State discipline.**
- A successful PUT is only *requested*. The value comes from observed resources: the SSE `motion` update or the next reload.
- Until each sensor is observed, the Status is `Pending`.
- A failed PUT sets the control Status to the error: the deck shows `Error`, and the GUI shows the reason. A partial failure leaves the observed `Mixed`.
- The bridge owns this state. Snoofer never persists it and never reapplies it on launch or reconnect.

**Model.**
- `motion` joins `usedTypes`.
- `resource` gains `Owner *reference` and `Enabled *bool`, and `merge` copies both.

**GUI.** The Lights room card shows a `Motion` row with the value, the sensor count and an On/Off button. It appears only while the room has sensors.

## 2. Control collections
- **New fields:** `snoofer.Control` gains `Collection` (a stable ID) and `CollectionLabel` (an editor label). Both are optional and affect presentation only. Behavior never depends on them.
- **Published collections:**

  | Collection | Label | Members |
  | --- | --- | --- |
  | `soundboard.clips` | Soundboard clips | every `soundboard.clip-*` |
  | `hue.room-scenes` | Scenes · selected room | `hue.room-scene-N` slots |
  | `hue.scenes.<roomID>` | Scenes · `<room name>` | `hue.scene-<room>-*` |

- **Room scene slots:** the Hue plugin now publishes `max(12, scenes in room)` slots, so a region can show every scene. Unused slots are `Hidden` (blank), as before.

## 3. Page regions
```go
// Region fills a rectangle of keys from a control collection.
type Region struct {
	Source string `json:"source"` // Collection ID, or a legacy ID prefix ending in "-".
	First  int    `json:"first"`  // Zero-based key indexes of opposite corners.
	Last   int    `json:"last"`
}
// Page gains: Regions []Region `json:"regions,omitempty"`
```
**Cells.** A region's cells are the keys in the rectangle, in row-major order. Cells with a manual binding on the page, or a shared binding, are skipped: manual wins.

**Candidates.**
- **Membership:** controls whose `Collection` equals `Source`. If `Source` ends in `-`, legacy prefix matching applies instead.
- **Eligibility:** members must support `press` and not be `Hidden`. Anything bound manually or shared on the page is excluded.
- **Order:** by Label, then ID.
- **Duplicate sources:** regions on the same page with the same source fill in sequence: the second continues where the first ends.

**Overflow.**
- **Page count:** a page expands to `max(ceil(candidates/cells))` pages over its regions, with a minimum of 1.
- **Overflow pages:** page *k* shows chunk *k* of each region; exhausted regions leave blank cells. Overflow pages keep the existing `~auto~N` IDs and the `Name N` names, and repeat every manual binding.

**Migration.**
- A page with `auto_controls` and no `regions` behaves exactly as before: one region covering every key, with the prefix as its source.
- The editor shows it as "Whole page · `<prefix>`".
- Editing that page's regions in the editor first converts the prefix into an explicit whole-page region and clears `auto_controls`. Saving an untouched page keeps it as is.

**Validation.**
- Indexes must be in range.
- Regions on a page must not overlap.
- A region needs at least one free cell.
- The source must be non-empty and contain `.`.
- The existing rule that a page needs a free key applies to legacy pages only.

**Editor view.** `EditorSlot.Source` keeps `Auto` and `Shared`. `EditorView` gains `Regions []{Source, Label, First, Last}` and a per-key `Region` index (-1 for none) so the grid can tint region cells.

## 4. Lights page and the personal layout
The personal layout is owned by the user, who asked for this change. It is edited once, after the build, with the running editor's Save path (or an equivalent JSON edit while Snoofer is stopped).

**Home.**
- **r1:** unchanged — Mic stack, Mute, Playback mute, Monitor, Mode, then Sync, Mode, Intensity and Brightness in c6–c9.
- **r2c9:** Motion, beneath Brightness.
- **Removed:** all twelve room-scene slots.
- **r4:** c8 holds the Soundboard go-to key and c9 the Lights go-to key.
- **Unchanged:** everything else, and the dials (Playback, Mic, —, —, Brightness).

**Lights** (new page after Soundboard):
- **r1:** Room, Brightness, Motion, Sync, Mode, Intensity, —, —, Home (go-to).
- **r2–r4:** region `hue.room-scenes` (27 cells).
- **Dials:** Playback, Mic, —, —, Brightness. These match Home, so Brightness never moves.

**Soundboard** (existing page, converted):
- **r1–r4 c1–c8:** region `soundboard.clips` (32 cells).
- **Column 9:** Overlap at r1, Home (go-to) at r3, Stop at r4 (where it is today).
- **Dial 1:** Volume.

## 5. UX review
### Design language for the deck
1. **Pages by task:**
   - Home holds the live essentials.
   - Content pages (Soundboard, Lights) hold browsing.
   - Diagnostics never appear on keys.
2. **Fixed frame, dynamic body:** fixed controls sit on an edge row or column, and generated content fills a region. Overflow pages repeat the frame, so Stop and Home never move.
3. **Positional constancy:** a control keeps its position across pages where it appears. Brightness is always dial 5, Playback and Mic are dials 1–2, and Home sits at the same edge.
4. **Blank versus N/A:**
   - Blank means not applicable now: a hidden control, an empty region cell or a room without sensors.
   - N/A means expected but unknown or unreachable.
5. **One-step reach:** anything used mid-session is at most one press away from Home, through a go-to key or the page dial. Page cycling is for browsing, not reaching.
6. **State words and the semantic palette are unchanged.** New icons use the existing primitives:
   - `hue-motion`: a dot with emanating arcs, slashed when Off.
   - `deck-page`: a folder tab with the page name as the label.

### Scenarios
| # | Scenario | Friction today | Response |
| --- | --- | --- | --- |
| 1 | Streaming at the desk: mute, mode, record | Home is crowded by 12 scene keys | Adopt: Lights page; Home keeps live Hue essentials |
| 2 | Film or game with Hue Sync | Motion sensors switch lights on mid-scene | Adopt: Motion key on Home |
| 3 | Late night dimming or lights off | None: Brightness dial and press already work | Keep |
| 4 | Long soundboard session | Prefix-only auto page; Stop position depends on free keys | Adopt: clips region inside a fixed frame (Stop, Overlap, Home, Volume) |
| 5 | Switching rooms to set a scene elsewhere | The Room control is GUI-only | Adopt: Room key on Lights; scenes and Motion follow the room |
| 6 | Reaching Lights from page 3 of the soundboard | Several dial detents | Adopt: go-to keys; dial press still returns Home |
| 7 | Building a page in the editor | Must know and type `soundboard.clip-`; whole page only | Adopt: rectangle selection with a named source picker |
| 8 | Reading key positions | Grid badges are zero-based; slot names and inspector are one-based | Adopt: one-based everywhere in the UI |
| 9 | First launch for a new user | The default layout has Home only | Adopt: default Soundboard and Lights pages |
| 10 | Hue Sync app closed, or bridge offline | Handled: Sync blanks; bridge controls show N/A | Keep |
| 11 | No Hue at all (plugin disabled) | Lights page is all blank keys | Candidate: skip pages whose bindings all belong to disabled plugins |
| 12 | Favorite clips on Home | Manual binding works, but is tedious for many clips | Candidate: favorites collection |
| 13 | Entering VR | The deck stays on the current page | Candidate: optional page switch on VR active |
| 14 | A smaller Stream Deck (MK.2, Plus) | Layouts assume 36 keys | Candidate: rectangles map to other grids later |
| 15 | Accidental consequential presses | Recorder Stop and similar are one press | Candidate: hold-to-confirm for consequential keys |

Candidates are recorded for later changes and are not implemented here.

## 6. Go-to page keys
- **Controls:** the Stream Deck plugin publishes `streamdeck.goto-<pageID>` for each saved page of the active layout (overflow pages excluded).
  - Label `Go to <name>`, ShortLabel `<name>`, Kind `command`, Icon `deck-page`.
  - The value is `Here` while that page (or one of its overflow pages) is shown.
- **Press:** switches the device to that page. It is not persisted and doesn't change Home.
- **Binding:** these are the only `streamdeck.` controls offered as bindings. A binding to a deleted page is unavailable, like any missing control.

## 7. Editor
- **Range selection:**
  - **Anchor:** a click or arrow keys set the anchor.
  - **Extend:** Shift+click or Shift+arrows extend the rectangle.
  - **State:** the selection is GUI-local presentation state.
  - **Dispatch:** a selection never dispatches a binding.
- **Regions:**
  - **Page panel:** a Regions list. Each row shows the source label, the range (`Keys 10–36`) and a Remove button.
  - **Adding:** `Add region` uses the current rectangle and offers the published collections. It is disabled when the rectangle overlaps an existing region.
  - **Editing:** a region's source can be changed in place.
- **Plugin requests:** the editor sends `streamdeck.region-add` (value `first,last,source`), `streamdeck.region-source` (`index,source`) and `streamdeck.region-remove` (`index`). All are draft edits; Save applies them as today.
- **Grid:** region cells get the active background tint and a small region number (`R1`); the selection is a dashed outline and clears after a region is added. Auto-filled keys keep the `Auto` marker. Key badges are one-based.
- **Legacy pages:** the "Automatic prefix" field is removed. A legacy page shows its migrated whole-page region.

## 8. Default layout
New installs get Home (as today) plus the Soundboard and Lights pages from §4, without the personal Hue block on Home. Existing saved layouts are never rewritten.
