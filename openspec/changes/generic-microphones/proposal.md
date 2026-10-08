# Generic microphones
## Why
The microphone model is one person's studio:
- **Hardcoded roles.** "desk" and "lav" are the first two channels of an ASIO interface, and "webcam" is a Windows device on input 3. These roles and their labels are written into the code.
- **Edits needed.** Another setup (one USB mic, three interface mics, a stereo mic pair) needs code changes.

## What Changes
- **Named microphones.** `studio.microphones` defines any number of microphones by ID and name, up to the hardware inputs Voicemeeter provides. Each is either:
  - **Interface microphone:** each interface entry maps microphone IDs to one channel (mono, fed to both sides of the strip) or two channels (stereo L/R).
  - **Device microphone:** a priority list of Windows input devices (by identity or pattern, see `device-identity`).
- **Fixed inputs.** Each microphone feeds the hardware input matching its position: the first microphone uses input 1, the second input 2, and so on. Inputs beyond the microphones stay unmanaged, and the VR input is excluded. Positions are stable, so a microphone's gain never moves between strips.
- **Everywhere by ID and name.** Priorities, activity metering, options, labels, the GUI and the deck use the user's microphone IDs and names. No role is special.
- **Migration.** A configuration without `microphones` keeps working unchanged: it behaves as desk and lav (interface channels from the old `inputs: [desk, lav]` array) and webcam (from `fallback_mic`). The Routing screen writes the explicit form on first edit.
- **Editing.** A Microphones card on the Routing screen adds, renames, reorders and removes microphones, chooses a device microphone's devices, and sets each interface's channel (and optional right channel) per microphone.

## Impact
- **Code:** `internal/config`, `internal/routing` (strips, patches, device inputs, options, managed strips), `internal/voicemeeter` (patch allowlist `Patch.asio[0-9]`), `internal/control` (activity strips, edits), `plugins/audio` (labels, view, edits), `app/web`, `docs/ui-contract.md`, AGENTS.md invariants.
- **Invariants amended:**
  - "Desk/lav channel mappings feed stereo inputs 1/2; zero means unavailable" becomes per-microphone channel mappings feeding each microphone's own input; an unmapped microphone is unavailable on that interface.
  - "Input 3 is reserved for the webcam" becomes: each device microphone owns its own input.

  Unchanged:
  - mic stack disable semantics;
  - the B2/AUX/B3 chain;
  - the VR input;
  - ASIO A1;
  - presence, absence and ambiguity rules.
