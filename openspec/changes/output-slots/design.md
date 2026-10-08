# Design
## Configuration
```json
"outputs": [
  {"id": "music", "name": "Music", "device": "Speakers (SteelSeries Arena 7)", "sources": ["virtual:1"]}
]
```
- **`id`:** `^[a-z0-9-]{1,24}$`, unique.
- **`name`:** non-empty, at most 16 characters (it is a deck label).
- **`device`:** an exact WDM output name, matched only when available, and unique across slots. A slot device that also matches a playback pattern belongs to the slot: playback selection, options and ownership skip it, so the two never compete.
- **`sources`:** each entry is a configured `playback_sources` value, `monitor`, `soundboard` or `tape`.
- **Count:** at most three slots. Planning also caps slots by the edition's A buses.

Saved choices gain `outputs: {<id>: {<source>: bool}}`.
- A slot without saved choices uses its configured `sources`.
- Choices for removed slots or sources are dropped on normalization.

## Bus allocation (deterministic, in `buildStudio`)
1. **Held buses.** A bus whose current device name equals a slot device is held by that slot. Only the first such bus counts; a duplicate on a later bus is cleared. A held bus stays reserved even while the device is unavailable, so a reconnect does not reshuffle outputs.
2. **Playback.** Playback devices exclude slot devices: they are marked unavailable for playback selection, options and ownership. Playback takes the lowest bus that is free and not held. If no bus is free, Playback takes the bus held by the last slot in config order, and that slot becomes "No output".
3. **Slots.** In config order, a slot with an available device keeps its held bus. Otherwise it takes the lowest free bus. A free bus is empty, or holds an owned playback device that is not the target. A1 is never free while ASIO is active. A slot with no free bus is "No output".
4. **Status.** `Topology.Outputs` reports each slot's ID, name, device, bus and state (`ok`, `missing`, `no-output`). Slot states never add `Unresolved` entries, so a missing slot device never blocks applying or recovery.

## Sends
For each slot placed on bus X:
- `Strip[virtual].X` follows each Computer source switch.
- The monitor strip computed by `addVoice` (mic, or AUX for Post) sends to X when the Monitor switch is on and monitoring is active. Managed mic strips' A sends stay owned on all buses, as today.
- `Strip[7].X` follows the Soundboard switch while the soundboard input is reserved.
- `Recorder.X` follows the Tape switch during listen-back. `routeTape` writes the Playback bus and every slot bus with Tape on, and leaves other tape buses at 0.

Sends from other strips to X are not planned.

## Controls
- **`audio.slot-N`** (N = 1..3): Kind status. Label is the slot name; Value is the device's bus state (`In use`, `Missing`, `No output`). Hidden when slot N is not configured.
- **`audio.slot-N:<source>`:** Kind toggle with `press` and `set`. It is Hidden when the slot or source does not apply, for example Soundboard without the soundboard input, or Tape without a recording profile.
- **Persistence:** route switches go through the worker's rule edits (`slot:<id>:<source>`), so they are saved before they are applied.
- **`audio.output-edit`** (Kind text, SurfaceOnly): JSON `{op: add|remove|rename|device|sources, id, value}`. It uses the priority editor's validate, save and reload path.

## GUI
The Routing screen's first card is Outputs.
- **Layout:** a table with a column per destination (Playback, then the slots) and a row per source. Each cell is an icon toggle with an On/Off label; inapplicable cells are empty.
- **Column headers:** each shows the name, the device and a state badge. Slot headers have a device select (connected WDM outputs not already used by Playback or another slot), Rename and Remove.
- **Add output:** takes a name and a device.

## Deck
The default layout gains a `routing` page:
- **Row 1:** `audio.playback-device`, `audio.playback:virtual:1`, `audio.monitor`, blank, blank.
- **Rows 2–4:** `audio.slot-N`, then `audio.slot-N:virtual:1`, `audio.slot-N:monitor`, `audio.slot-N:soundboard`, `audio.slot-N:tape`.
- **Dials:** Playback and Mic.

Default Home gets a go-to Routing key. The icons reuse existing primitives: speaker, monitor, soundboard and tape.
