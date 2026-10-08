# Design
## Ready
`studio.microphones[].ready` (bool, omitted when false).
- **Metering:** the worker samples only wired microphones that are not ready. A ready microphone never enters the latched silent set, and leaves it at once when it becomes ready.
- **Selection:** unchanged. Auto's silence skip uses the silent set, so a ready microphone is chosen whenever it is an option. Options still require the device to be present or the interface to map the microphone.
- **Legacy configurations:** the first microphone edit writes the explicit microphone list, as today.

## Edits
All through `audio.priority-edit` (save, validate, then apply).

| List | Op | Field | Value |
|---|---|---|---|
| `mics` | set | `ready` | `true` / `false` |
| `mics` | set | `source` | `interface`, or a device candidate as JSON |
| `interfaces` | set | `input:<id>` | unchanged: `c`, `l,r` or empty |
| `activity` | set | `check` | `0` removes `profiles.activity`; 1–4 sets it |
| `activity` | set | `silence_db` / `silent_after_s` | number; needs activity on |

- **Source change to a device:** the microphone's `devices` become that one candidate, and every interface drops its channels for it.
- **Source change to interface:** `devices` are removed. It is unavailable until an interface maps channels.
- **Activity** needs `profiles` (Auto); without it the edit is refused.
- **Interface add:** the GUI sends `inputs: {}`.

## View
`micView` gains `Ready` and `Silent` (in the latched set). The priority view gains `Activity` (nil when off) with `Check`, `SilenceDB` and `SilentAfterS`, and `Profiles` (activity editable).

## GUI
- **Microphone row:** name, source select (Interface channels, the current device, connected input devices), Ready toggle, status badge (In use, Ready, Silent, Unavailable), order buttons.
- **Interface channels:** per interface microphone, a select (Off, Mono, Stereo) and one or two channel number inputs. Changing a value sends `input:<id>`.
- **Activity card:** a Meter select (Off, 1–4 options), threshold (dBFS) and delay (s) fields that apply on change.
