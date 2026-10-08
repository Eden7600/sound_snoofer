# Design
## Configuration
```json
"studio": {
  "microphones": [
    {"id": "desk", "name": "Desk"},
    {"id": "lav", "name": "Lavalier"},
    {"id": "webcam", "name": "Webcam", "devices": [{"driver": "wdm", "id": "...", "name": "Microphone (Insta360 Link 2)"}]}
  ],
  "asio": [{"asio_pattern": "...", "presence_pattern": "...", "inputs": {"desk": [1], "lav": [2]}}]
}
```
- **Microphone ID:** `^[a-z0-9-]{1,24}$`, unique, and not `off`, `auto` or `normal`, nor starting with `vr`.
  - **Name:** 1–16 characters.
  - **Kind:** a microphone with `devices` is a device microphone. Without `devices`, it is an interface microphone.
- **Interface inputs:** each interface entry's `inputs` maps a microphone ID to `[channel]` or `[left, right]`, channels 1–64. The legacy array `[desk, lav]` decodes as `{"desk": [desk], "lav": [lav]}`, with zeros omitted. Every key must be an interface microphone.
- **Legacy synthesis:** when `microphones` is absent, Validate synthesizes desk and lav, plus webcam when `fallback_mic` is set, with `devices` = `fallback_mic`. `fallback_mic` is not allowed together with `microphones`.
- **Validated against microphone IDs:**
  - `voice.source` (default: `desk` if defined, else `auto`);
  - `profiles.microphones`;
  - recording and activity.
- **Input limits:** inputs available to microphones are the edition's physical strips minus the VR input, so Potato has 4 with VR and 5 without. `ValidateEdition` rejects more microphones than that.

## Planning
Microphone `n` (0-based) owns strip `n`, input `input:n+1` and ASIO patch cells `Patch.asio[2n]` and `[2n+1]`. Strips beyond the defined microphones are untouched.

**ASIO patches**, while ASIO is active and the mic stack is active:
- An interface microphone mapped on the selected interface and wired writes `[c, c]` (mono) or `[l, r]` (stereo).
- Every other microphone's cells are 0.
- With ASIO inactive or the stack inactive, all managed cells are 0.

**Inputs:**
- An interface microphone's input is cleared.
- A device microphone's input is assigned its selected device while the stack is active and it is wired, and cleared otherwise.
- An input occupied by a device that matches no candidate of its microphone is an error, as input 3 is today.

**Options:** an interface microphone is an option when the selected interface maps it. A device microphone is an option when a candidate has exactly one available match. Then VR headsets, then `off`.

**Source strip:** the effective microphone's strip. The webcam-fallback rule becomes: when the chosen microphone is unavailable, use the first available device microphone, then mark the source unresolved.

**Managed strips:** every microphone strip, AUX (strip 6) and the VR input. Playback-move skips exactly these.

**Activity metering:** wired microphones are read on their strips, from the configuration rather than a fixed map.

## Surfaces
- Labels come from microphone names, so `ChoiceLabels` and option labels use the configured names. The GUI drops its hardcoded name map in favor of labels from the view.
- The Routing screen's Microphones card lists microphones in input order, with Up/Down, Rename, Remove and Add (name, then interface or device).
  - A device microphone shows its device priority, using the same entry rows as Playback (Add by identity or Pattern).
  - Each interface row shows a channel field per interface microphone, plus an optional right-channel field.
- Edits use `audio.priority-edit` lists:
  - `microphones` for the priority;
  - `mics` for definitions, with ops add, remove, move and rename;
  - `mic-devices:<id>` for a device microphone's priority;
  - interface field `input:<id>` with value `c` or `l,r`.

  The first edit writes the synthesized legacy form explicitly.
