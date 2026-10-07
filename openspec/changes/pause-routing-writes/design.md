# Design
## Classification
Routing plan operations are already typed:
- **Device:** an operation with a `Device` (`Target` is an output bus or an input slot).
- **Send:** an operation with a `Parameter`. That covers `Strip[n].A1–A5/B1–B3`, `Strip[n].Mute`, `Bus[n].Mute`, `Patch.asio[n]` and `Recorder.*`.

## Holding operations in the plan
- **Where:** `routing.addVoice` applies the intent's pauses before building the voice transition. A held operation keeps its desired value and `BeforeValue`, but gets `Change = false`. Topology records `HeldDevices` and `HeldSends` counts.
- **Effect on the rest of the plan:**
  - `HasChanges` ignores held operations, so the debounce, Apply and the final verification behave as if nothing were pending.
  - Drift checks still compare observed values with the plan's `BeforeValue`/`BeforeName`.
  - The plan key keeps the desired state, so a held change does not churn the key.
- **Transitions:**
  - **Sends paused:** no transition is built; device operations, if allowed, apply directly. The transition consists only of send writes (gating sends off and back on), and those are exactly what is paused.
  - **Devices paused:** device operations are held before the transition is built, so they never gate sends. Otherwise sends would be switched off for a device change that never happens.
- **Scope:** only studio voice plans carry an intent; the legacy route list (no voice profile) is unaffected.

## Controller paths outside the plan
- **Tape-playback protection** (`protectRecorder`, Recorder.B1–B3) is skipped while sends are paused.
- **Recorder Start preparation:** writes the recorder's routing parameters. While sends are paused, Start fails with "Sends paused" if a write is needed, and proceeds if the recorder is already prepared.
- **Unchanged:** gain and mute reconciliation (`Mixer`), recorder transport, snippet play and engine restart.

## Persistence and controls
- **Storage:** `Intent.PauseDevices` and `Intent.PauseSends` (`pause_devices`, `pause_sends`) are edited through the existing toggle rows `pause-devices` and `pause-sends`. That means save-before-apply and stale-revision rejection, like `auto-recover`.
- **Controls:** the audio plugin publishes the toggles as "Disable device manipulation" and "Disable send manipulation" (On means paused). Their status is the held count, such as "2 held".
- **GUI:** they fall into Routing & recovery with the other audio rows. The header shows an attention badge, Devices paused or Sends paused, while either is on.
