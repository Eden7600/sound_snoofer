# Design
## Presence
`presence.Watch(ctx)` returns a channel of `State{Known, Locked, DisplayOff}` and a done channel. Its goroutine is owned by the caller, cancelled by `ctx`, and bounded on shutdown.

**Startup.** The goroutine locks its OS thread and creates a hidden window, then registers that window for:
- `WTSRegisterSessionNotification(NOTIFY_FOR_THIS_SESSION)`:
  - `WM_WTSSESSION_CHANGE` with `WTS_SESSION_LOCK` (7) or `WTS_SESSION_UNLOCK` (8).
- `RegisterPowerSettingNotification(GUID_CONSOLE_DISPLAY_STATE)`:
  - `WM_POWERBROADCAST` / `PBT_POWERSETTINGCHANGE` with data 0 (off), 1 (on) or 2 (dimmed).
  - Windows sends the current value immediately after registration.

**Initial lock state.** It comes from `WTSQuerySessionInformation(WTSSessionInfoEx)` `SessionFlags`: lock is 0 and unlock is 1, as on Windows 8 and later. When that fails, the state is unknown, which is treated as unlocked.

**Shutdown.** On cancellation the goroutine posts `WM_CLOSE`, unregisters both notifications, destroys the window and returns.

**Failure.** If the window or the registrations fail, the watcher reports `Known: false` once and ends. The deck then behaves as today.

**Other platforms.** A stub reports unknown.

## Deck
- `Frame` gains `Dark bool` and `Brightness int`.
- **Dark frames.** While `Dark`, the surface draws the empty frame and, on entering dark, sends brightness 0. On leaving dark, it sends `Brightness` and redraws the frame.
- **Brightness report.** Brightness is the Stream Deck v2 feature report `[0x03, 0x08, percent]`, padded to the device's feature report length and sent with `HidD_SetFeature`.
  - A failed feature report is a diagnostic only, never a device failure. The black frame is the guaranteed part.
  - Hardware acceptance confirms the report on the Stream Deck + XL.
- **Plugin.**
  - `dark = Known && (Locked || DisplayOff)`.
  - While dark, events are dropped before any dispatch, local navigation or hold tracking, and a pending hold is cleared.
  - Publishing sends a dark frame (still `Generation`-stamped) instead of the page.
  - Reconnecting while dark stays dark.
- **Report.** The Stream Deck connection report gains a `Display` detail: `Dark (locked)`, `Dark (displays off)` or `On`.
