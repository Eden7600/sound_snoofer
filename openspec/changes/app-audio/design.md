# Design

## 1. Session adapter (`internal/windowsaudio`)
This builds on the existing pure-Go COM approach (`defaults_windows.go`): vtable calls through `syscall.SyscallN` on a locked OS thread, with no new dependencies. All of it runs on one worker goroutine owned by the plugin.

### Interfaces used
| Interface | Methods |
|---|---|
| `IMMDeviceEnumerator` | `EnumAudioEndpoints(eRender, DEVICE_STATE_ACTIVE)` |
| `IMMDevice` | `Activate(IAudioSessionManager2)`, plus the device's friendly name for diagnostics |
| `IAudioSessionManager2` | `GetSessionEnumerator` |
| `IAudioSessionEnumerator` | `GetCount`, `GetSession` |
| `IAudioSessionControl2` (QI) | `GetState`, `GetDisplayName`, `GetProcessId`, `IsSystemSoundsSession`, `GetSessionInstanceIdentifier` |
| `ISimpleAudioVolume` (QI) | `Get/SetMasterVolume`, `Get/SetMute` |
| `IAudioMeterInformation` (QI) | `GetPeakValue` |

- **Float arguments:** `SetMasterVolume` takes a float in the second position. The Windows amd64 `asmstdcall` mirrors the first four integer argument slots into XMM0–3, so the float's bits are passed as a uintptr. A native-probe test checks the read-back.
- **Process identity:** `OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION)` and `QueryFullProcessImageName` give the executable path. The system sounds session (PID 0) becomes path `system`.
- **Name:** the executable's version resource `FileDescription`, otherwise the base name without `.exe`. The session display name is used only when it is a plain string (not `@…` or `ms-resource:`).
- **Icon:** the executable's large icon (`PrivateExtractIcons` at 64 px, then `GetIconInfo`/`GetDIBits`), encoded as a 64 px PNG `Artwork`. Names and icons are cached by path and modification time.
- **Snapshot:** one session per (device, instance ID): path, PID, name, state, volume, mute and peak.
- **Polling:** sessions are listed every second; peaks are read every 100 ms from the open sessions.
- **No callbacks:** COM session-notification callbacks are not implemented. Polling avoids COM servers written in Go.

## 2. Apps from sessions (`plugins/appaudio`)
### Rules
`Settings.Rules` is an ordered list of `Rule{Match, Name string; Hide bool}`:
- `Match` is a case-insensitive Go regexp against the executable path, or `system` for system sounds.
- The first matching rule wins:
  - `Hide` drops the session;
  - `Name` sets the display name;
  - a rule with neither is an explicit *show*, which stops later rules (it is how a default is undone).
- **Built-in defaults** apply after the user's rules and hide Snoofer, Voicemeeter (`voicemeeter*.exe`) and `audiodg.exe`. Element is not hidden by default.
- **Grouping:** sessions with the same final display name, compared case-insensitively, form one app. That covers several processes of one program (Chrome, Discord), several sessions per process, and the same app on several devices. Combining two programs is a rule that gives both the same name.
- **App ID:** `appaudio.app-` plus a short hash of the lowercase name. It stays stable across restarts and changes only when the app is renamed.

### Volume and mute across sessions
- **Volume:** the shown volume is the highest volume among the app's sessions. Setting it writes the same value to every session.
- **Mute:** Muted when every session is muted, Mixed when only some are, otherwise the volume. A press mutes every session unless all are already muted, in which case it unmutes them all.
- **Pending:** a write is requested, then Pending until a read-back matches within 0.5 % on every session. If it is not observed within 2 s, the status is **Ignored by app**. This is how programs that reset their own session volume become visible instead of silently failing.
- **Turning:** a dial turn moves 2 % per detent from the requested value while Pending, otherwise from the observed value.
- **Persistence:** Snoofer never stores or reapplies volumes. Windows remembers per-app volume itself.

### Visibility and order
- **Picked apps** (`Settings.Picked`, ordered names) always appear first in that order. A picked app with no sessions shows `Closed` and is unavailable.
- **Others** appear while heard: some session's peak exceeded −60 dBFS (0.001) within `Settings.RecentMinutes` (default 5). They are ordered by name, so positions do not reshuffle with every sound.
- **Hearing:** it is tracked in memory only. After a restart, an app reappears once it plays again.
- **Hidden apps** never appear, but they are listed in the GUI while they have sessions.

### Controls
- **App control (`appaudio.app-<key>`):**
  - Kind `numeric`; operations `press` (mute), `adjust` (volume) and `set` (percent, from the GUI slider).
  - Value `42%`, `Muted` or `Mixed`; Status `Pending`, `Ignored by app` or empty.
  - Carries `Meter` (the highest peak, in dBFS), `Artwork` (the icon), ShortLabel (the name) and Collection `appaudio.apps` (label "Apps").
  - Carries `Order`, the app's position. The registry snapshot does not keep publish order, so a new optional `snoofer.Control.Order` field records it, and regions fill by Order before label. Other collections leave it at zero, so their order is unchanged.
- **`appaudio.status`:** a status control whose ViewData lists every app (visible and hidden) with diagnostics: executables, PIDs, devices, session count, last heard and the matching rule.
- **`appaudio.edit`:** a text control that receives GUI edits as JSON: `{"op":"pick|unpick|move|hide|unhide|rename|combine","app":"…","value":"…"}`.
  - GUI rules match the app's executable file names exactly (`(?i)(^|\\)(discord\.exe)$`) and are inserted first.
  - **Unhide** removes the app's GUI hide rule. For a default-hidden app it inserts a *show* rule instead.
  - **Rename** and **Combine** keep picks pointing at the new name.
- **Preview mode:** sessions are read but never written, and app controls report `Preview`.

## 3. Deck
### Dial regions
`Page.DialRegions []Region` fills dials from a collection, using dial indices 0–4 as First/Last. The page dial is never part of a region.
- **Expansion:** dial streams join the overflow calculation, so the set count is the largest across key and dial streams.
- **Paging:** each set shows the next chunk of every stream. A five-key region and a five-dial region of the same collection therefore page in step, keeping each key above its dial's app.
- **Editor:** dial regions are listed with key regions and created by Shift-selecting dials. Validation rejects overlap with bound dials or other dial regions.

### Apps page (defaults and personal layout)
- **Dials 1–5:** a dial region of `appaudio.apps`. The strip shows name, value and meter, and its position track shows the percentage.
- **Keys 1–5 (r1):** a key region of `appaudio.apps`, showing icon, name and value; Muted uses the critical colour. A press mutes.
- **Up and Down** (keys 18 and 27) page through apps beyond five.
- **Home** gains a go-to key for Apps at r4c7, beside Soundboard and Lights.

## 4. GUI: App audio screen
- **Sidebar:** **App audio** sits after Lights. The existing "Third-party apps" screen keeps its name.
- **Visible apps:** each gets a strip with icon, name, a volume slider (sends `set` with a percent), a mute button, a meter, a pin toggle (pick/unpick) and up/down order for picked apps.
- **More menu:** Rename, Combine into… (a list of other apps) and Hide.
- **Hidden card:** hidden apps that have sessions, with the rule that hid them and an Unhide button.
- **Details:** an expander per app with executables, PIDs, devices, sessions, last heard and the matching rule.
- **Rebuilds:** the screen rebuilds only when `appaudio.status` ViewData changes; meters update in place at the existing 100 ms poll.

## 5. Scenarios considered
| Scenario | Handling |
|---|---|
| Chrome with a renderer process per tab | Grouped by name. One strip; volume and mute apply to all of them. |
| Discord helper processes | Grouped by name. |
| Game keeps a silent Active session | Not shown unless picked, because hearing uses the meter. |
| Store app with `@{…}` display name | Version description or file name is used instead. |
| App resets its own volume | Shows **Ignored by app**; Snoofer does not fight it. |
| App plays on two devices | One app; both sessions are set. |
| Two programs that belong together (game + launcher) | Combine into…. |
| Plumbing (audiodg, Voicemeeter, Snoofer's soundboard) | Hidden by default; Unhide is available. |
| Exclusive-mode or ASIO app | Has no shared session, so it never appears. The GUI notes this. |
| Many apps | Overflow pages with Up/Down; pick the important ones to pin their positions. |

## 6. Revision: exclusion list
Review asked for a visible exclusion list, with Hue Sync excluded by default. Hue Sync listens to system audio and reacts to it, so its session behaves erratically.
- **Setting:** `Settings.Exclude []string` holds program file names, case-insensitive, with `*` wildcards (for example `voicemeeter*.exe`). The entry `system` covers System sounds.
- **Defaults:** when `exclude` is absent (JSON null), the defaults apply: `snoofer.exe`, `voicemeeter*.exe`, `audiodg.exe` and `huesync.exe`. The first edit stores the explicit list, so you can remove any default. An empty list means nothing is excluded.
- **Precedence:** exclusion is checked before rules, so an excluded program stays hidden even if a rule names it. The built-in hide rules are removed. `Rule.Hide` remains for regex rules written in JSON.
- **Edits:**
  - `exclude` adds a pattern and `include` removes one; neither needs the program to be running.
  - **Hide** adds the app's program file names.
  - **Unhide** removes every pattern matching the app's programs, plus any exact-match hide rule for them.
- **GUI:** an **Excluded** card lists the patterns. Each shows the running apps it currently hides and has a Remove button. An Add field takes a file name. Apps hidden by JSON regex rules are listed under the card with their rule.

## 7. Revision: configurable recent window
Review asked for the 5-minute window to be configurable in the app.
- **Setting:** `Settings.RecentMinutes` already exists (0 means the default of 5; the maximum is 1440).
- **Edit:** a new `recent` op takes a whole number of minutes from 1 to 1440 and saves it.
- **GUI:** the App audio screen's intro line becomes "Pinned apps first, then apps heard in the last [N] minutes", with N in a number field that saves on Enter or when the field loses focus.
- **Effect:** the change applies on the next refresh. Memory of when apps were heard is kept, so lengthening the window can bring recently heard apps back.
