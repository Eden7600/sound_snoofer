# Output slots and audio routing
## Why
Today there is exactly one listening destination, Playback.
- **Second devices get overwritten.** A second device that matches a playback pattern counts as a former playback output, so Snoofer clears it and moves its sends.
- **Manual sends get erased.** Every send from the mic, soundboard and tape strips to a bus other than Playback is forced to 0.

Wanting speakers for music and headphones for monitoring therefore means pausing Snoofer's writes. Routing is configured as buses (A1, B2) rather than intent, and the GUI and the deck have no place to see or change it.

## What Changes
- **Named output slots.** Config gains up to three `studio.outputs`. Each has an ID, a name such as "Music" or "Monitor output", and one static device chosen by exact name (no priority list).
  - A slot's device gets its own A bus: the bus it already holds, otherwise the lowest free one.
  - A slot's device is never treated as a playback candidate. Snoofer never clears it, moves its sends or picks it as Playback.
  - Playback keeps its priority list and still takes the lowest free output, but buses held by slot devices are not free.
- **Intent routing.** Sources are Computer (each configured playback source, `virtual:N`), Mic monitor, Soundboard and Tape.
  - Each slot has per-source switches, saved as choices (save before apply). Defaults come from the slot's configured `sources`.
  - **Monitor:** the slot receives the same tap as Playback (Pre/Post).
  - **Soundboard:** the soundboard strip is sent to the slot.
  - **Tape:** listen-back plays to the slot as well as to Playback.
  - Snoofer owns exactly these sends on a slot's bus. Everything else on that bus is left alone.
- **Routing screen.** The Routing screen (see `priority-list-editor`) gains an Outputs card at the top: a matrix with sources as rows and Playback plus the slots as columns.
  - Each cell is a toggle, with an icon and On/Off.
  - Each column header shows the destination's device and its state: In use, Missing or No output.
  - Slots can be added, renamed, assigned a connected device or removed.
  - The Playback column reuses the existing controls: the Computer source toggles, Monitor, and Soundboard monitor.
- **Stream Deck Routing page** in the default layout:
  - **Row 1:** Playback.
  - **Rows 2–4:** slots 1–3.
  - **Column 1:** the destination (name and device).
  - **Columns 2–5:** Computer, Monitor, Soundboard and Tape switches.
  - Keys for missing slots are blank.
  - Home gains a go-to Routing key in the default layout.
  - The personal layout is user-owned, so the user adds the page there.
- **Removing a slot** leaves its device and sends as they are. They become unmanaged.

## Impact
- **Code:**
  - `internal/config` (outputs schema, intent);
  - `internal/routing` (bus allocation, slot sends, tape buses, playback exclusion);
  - `internal/controller/tape.go`;
  - `internal/control` (route edits);
  - `plugins/audio` (controls, output edits);
  - `plugins/streamdeck/default.go`, `internal/streamdeck` (icons);
  - `app/web`;
  - `docs/ui-contract.md`.
- **Invariants:** CLAUDE.md's "Playback takes the lowest free output and its routing follows that output" is amended: buses held by slot devices are not free. Unchanged:
  - mic stack disable semantics;
  - B1/B2/B3 and AUX ownership;
  - ASIO A1 reservation;
  - pause semantics;
  - save-before-apply.
