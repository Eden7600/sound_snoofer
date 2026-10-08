# Dark Stream Deck while locked or asleep
## Why
When Windows locks or the monitors turn off, the Stream Deck stays lit and fully usable. Anyone at the desk can see the keys and use them to mute, unmute, start a recording, play clips or change lights.

## What Changes
- **Dark state.** While the session is locked or the monitors are off, the deck is dark:
  - every key and the touch strip show black;
  - the backlight is turned off where the hardware supports it.
- **Inert.** Every key, dial and touch is ignored while dark. Nothing is dispatched, and a held key does not fire its hold action.
- **Return.** When the session is unlocked and the monitors are on, the deck redraws the current page and restores its brightness. Dimmed monitors count as on.
- **Brightness setting.** The deck's restored brightness is a new optional `brightness` setting (1–100, default 100). Snoofer writes it only when the deck leaves the dark state.
- **Unknown state.** If Windows does not report state, the deck stays as it is today, lit and usable. A failure to observe never locks the user out.

## Impact
- **Code:**
  - `internal/presence` (new): Windows session lock and console display state, with a stub elsewhere;
  - `internal/streamdeck`: dark frames and the brightness feature report;
  - `plugins/streamdeck`: dark state, dropped input, setting;
  - `docs/ui-contract.md`.
- **Invariants:**
  - Bindings, layouts, pages and control state are unchanged.
  - A dark deck still owns its HID device and keeps reading input so the queue drains.
