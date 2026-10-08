# Device identity
## Why
Devices are matched by regular expressions over names, and names are a poor identity:
- Windows renames endpoints, and it inserts `2-` / `3-` prefixes that change when a device is re-plugged.
- Writing a pattern is expert work, even with generated suggestions.
- Voicemeeter's reported hardware ID cannot replace names. It is the enumerator class: the Insta360 and the SteelSeries both report `USB\Class_01`. ASIO entries are the exception, because they report a unique driver CLSID.

## What Changes
- **Device entries.** A priority entry (playback, webcam/device mics), an interface's driver and presence input, and an output slot's device can each name a device by identity: `{"driver": "wdm", "id": "<Windows endpoint ID>", "name": "<label>"}`. ASIO drivers use their CLSID.
- **Resolution.** On every planning cycle, Snoofer resolves each ID to the device's current name:
  - Windows endpoints come from the Windows audio worker, which now always enumerates active endpoints.
  - ASIO drivers come from Voicemeeter's inventory.
- **Matching.** A resolved entry matches exactly that name. An unresolved entry (disconnected) matches nothing for selection. It keeps ownership of a bus still showing its stored name, unless a connected device now has that name.
- **Patterns.** They remain as an *Advanced* entry type for rules such as "any AirPods".
- **Editor.** Suggestions add devices by identity (Add). The pattern forms move under Pattern. Entries by identity show the device's current name, or its stored label with *Disconnected*.
- **Slots** are assigned by identity too.

## Impact
- **Code:**
  - `internal/windowsaudio` (endpoint inventory in results);
  - `internal/config` (entry identity, resolution);
  - `internal/control` (resolution before planning, endpoints in state);
  - `plugins/audio` (view, edits);
  - `app/web`;
  - `docs/ui-contract.md`.
- **Invariants:**
  - The first entry with exactly one available match wins.
  - Ambiguity is skipped.
  - Installed ASIO drivers never prove presence.
  - An ID shared by two active endpoints with the same name is ambiguous.
  - Without the Windows audio worker (CLI), WDM identity entries are unresolved; patterns still work.
