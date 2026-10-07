# Design

## 1. Windows media sessions (`internal/mediasessions`)
C++/WinRT in a companion DLL, `bin/snoofer-media.dll`, built by `scripts/build-media.ps1` from `internal/mediasessions/native/media.cpp`. It is called from `scripts/build.ps1` like the soundboard DLL and is locked-file checked the same way. The ABI is Windows x64 C with UTF-8 JSON payloads, which keeps the Go side free of WinRT vtables.

| Export | Purpose |
|---|---|
| `MSOpen(MS**)` | Initializes a multithreaded apartment on the calling thread and requests the session manager. |
| `MSSnapshot(MS*, char* buf, uint32 cap, uint32* needed)` | JSON array of sessions. Each has `id`, `app` (AppUserModelID), `title`, `artist`, `album`, `status` (playing, paused, stopped, changing, closed), `positionMs`, `durationMs`, `updatedMs` (Unix ms of the timeline sample), `rate`, `canPlay`, `canPause`, `canNext`, `canPrev`, `canSeek`, `current` and `artKey`. |
| `MSArt(MS*, const char* id, char* buf, uint32 cap, uint32* needed)` | The session's raw thumbnail bytes. |
| `MSCommand(MS*, const char* id, int op, int64 value)` | Ops: 1 play, 2 pause, 3 toggle, 4 next, 5 previous, 6 seek (absolute ms). |
| `MSClose(MS*)` | Releases the manager and leaves the apartment. |

- **IDs:** a session's ID is its AppUserModelID plus `#n` when several sessions share one; `n` is the order within that app.
- **Artwork key:** `artKey` hashes title, artist and album, so Go fetches artwork only when it changes. Go scales it to the 64 px `Artwork` contract.
- **Calls:** the DLL blocks on WinRT async calls with `.get()`, which is allowed in a multithreaded apartment. All calls come from the plugin's locked OS thread.
- **Sizing:** a result that does not fit `cap` returns `ERROR_INSUFFICIENT_BUFFER` with `needed` set, and Go retries once with that size.
- **Status:** command success means the request was accepted. Pending lasts until a snapshot shows the change.

## 2. Browser bridge
Revised after review: the extension is a standalone codebase published to the Chrome Web Store and Firefox Add-ons. Nothing is exported from Snoofer and no token or pairing is used.

### Extension project (`extension/`)
**Layout:** `src/` holds the shared code: `page.js`, `bridge.js`, `background.js`, `logic.js`, `popup.html`, `popup.js` and the icons. `manifest.json` is generated per browser by `build.mjs`.

**Build:** `npm run build` writes `dist/chrome` and `dist/firefox` (load these unpacked while developing). `npm run package` zips each into `dist/snoofer-media-<browser>-<version>.zip` for store upload, using Windows `tar` with no new dependencies. `npm test` runs the Node tests.

**Browsers:**

| | Chrome, Brave, Edge | Firefox (≥ 128) |
|---|---|---|
| Background | Manifest V3 `background.service_worker` (module) | Manifest V3 `background.scripts` (module) |
| Extension ID | — | `browser_specific_settings.gecko.id` |
| Host permission | granted on install | optional; the popup asks with `permissions.request` |

- **Page-world script:** both use `world: "MAIN"` content scripts.
- **API namespace:** code uses `globalThis.browser ?? chrome` with promises.
- **Policy:** `content_security_policy.extension_pages` is `script-src 'self'; object-src 'self'`, so the local `ws://` connection is never upgraded.

**Permissions:** `tabs` (titles, mute), `storage` (the port setting), `scripting` (open tabs at install) and host access to `<all_urls>` (content scripts and artwork fetches). There is no remote code, and data goes only to `127.0.0.1`.

**Existing tabs:** browsers inject content scripts only into pages loaded after install. On `runtime.onInstalled` and `runtime.onStartup`, the background therefore injects `page.js` (main world) and `bridge.js` into every open http(s) tab and frame, using the `scripting` permission. Both scripts guard against running twice.

**Scripts:**
- **`page.js`** (all frames, from `document_start`, or injected later):
  - At start it adopts `<audio>`/`<video>` already in the document and reports at once if one is playing.
  - `timeupdate` from an unknown element adopts it too, so media that was already playing is found.
  - It wraps `navigator.mediaSession.setActionHandler` to remember page handlers such as YouTube's `nexttrack`.
  - It tracks `<audio>` and `<video>` elements through captured media events and reads `navigator.mediaSession` metadata and state.
  - It reports through a `CustomEvent` on changes, and every second while playing.
  - Commands use a page handler first, otherwise the element.
- **`bridge.js`** (isolated world) relays to a runtime port, opened only once the frame has media.
- **`background.js`:**
  - It keeps one WebSocket to `ws://127.0.0.1:<port>/nowplaying`, reconnecting with backoff and pinging every 20 s.
  - It merges frame reports per tab and adds the tab title, site and mute state.
  - It turns artwork into 64 px PNGs with `OffscreenCanvas`.
  - It runs commands and tab mute.
  - It answers the popup's status requests.
- **Popup:** shows "Connected to Snoofer" or "Snoofer is not running", the number of playing tabs and the port field (default 47815). On Firefox it also shows **Allow on all sites** while host access is missing.

### Server (in the plugin)
- **Listener and origin:** `127.0.0.1:<port>` (default 47815) serves only `/nowplaying`. The origin must start with `chrome-extension://` or `moz-extension://`, so web pages, which cannot forge an origin, are refused. Off-machine clients cannot reach a loopback listener.
- **Remaining access:** local programs and other installed extensions can reach the bridge. Local programs already run as the user; another extension could at most read media titles or press play. No secret is needed.
- **Handshake:** the first message is `hello{protocol, version, browser}`. Protocol 1 is supported; any other value is refused with "Update Snoofer" when newer, or "Update the extension" when older.
- **Messages from the extension:** `sessions{[…]}`. Each session has `id` (`tab:frame`), `tab`, `site`, `title`, `artist`, `album`, `art` (base64 PNG, sent only when it changes, otherwise `artKey`), `state`, `positionMs`, `durationMs`, `updatedMs`, `rate`, `canNext`, `canPrev`, `canSeek`, `muted` and `audible`.
- **Messages to the extension:** `command{id, op, value}`.
- **Bounds:** messages are limited to 1 MiB and sessions to 64 per browser. One connection per browser name; a newer connection replaces the older one.
- **Legacy setting:** `Settings.Token` from the first build is accepted and ignored, so saved configurations still load.

### De-duplication
A Windows session is hidden only when its non-empty title equals the title of a tab that a connected extension reports. It is never hidden by app ID alone, so a session cannot vanish when the extension cannot see its tab.

(Revised after review: hiding Brave's session as soon as its extension connected removed a video that was already playing in a tab opened before the extension was installed.)

The browser's single Windows session repeats one of its tabs, so title matching removes the duplicate for every browser, Firefox included.

## 3. Sessions and focus (`plugins/nowplaying`)
- **Session control:** each session gets `nowplaying.s-<hash of source and id>`.
  - **Labels:** Label is the title (falling back to the tab title or app name); ShortLabel is the title.
  - **Value:** Playing, Paused or Stopped. Status is Pending until a command is observed, or Failed.
  - **Artwork and grouping:** Artwork is the cover, or the favicon for tabs. Icon is `media-play`. Collection is `nowplaying.sessions` ("Media sessions"), with Order by most recently started first.
  - **Press:** toggles play/pause and focuses the session.
- **Focus rule:**
  - **Default:** focus goes to the session that most recently changed to Playing, unless you pressed a session or a transport control within the last 30 s.
  - **Single playing session:** when exactly one session is playing and the focused one is not, focus moves to the playing one at once, regardless of the 30 s hold (revised after review).
  - **Disappearing focus:** a focused session that disappears gives way to the next by the same rules.
- **`nowplaying.focus`:** a selection of session IDs with option labels, for GUI focus and the deck's cycling key. It is Hidden with fewer than two sessions.
- **`nowplaying.dial`:**
  - **Kind and operations:** numeric; `adjust` seeks ±5 s per detent from the interpolated position, and `press` toggles play/pause.
  - **Value:** `m:ss / m:ss`, or `m:ss` when the length is unknown.
  - **Labels and artwork:** Label is the title, ShortLabel the artist or site, and Artwork the focused session's art.
  - **Availability:** unavailable without a session; the strip then shows "Nothing playing".
- **Transport keys:** `nowplaying.prev`, `nowplaying.toggle`, `nowplaying.next` and `nowplaying.mute`.
  - Each is Hidden when the focused session cannot do it. Mute exists only for browser tabs.
  - Toggle shows Playing or Paused.
- **Position:** it is interpolated as `position + (now − updated) × rate` while playing, clamped to the length.
- **Status control:** `nowplaying.status` carries ViewData with sources (Windows: ok or error; browsers: connected, version and session count), sessions with full details (for the GUI), and the extension path and token state.
- **Polling:** Windows snapshots every 500 ms (artwork only when `artKey` changes); browser state arrives by push.

## 3a. Seeking (revised after review)
Review found seeking unreliable. The dial's value (`m:ss / m:ss`) changed every second, which gave the control a new revision each time. Deck turns that crossed an update were rejected as stale, and the new generation also dropped deck events. Each detent also sent its own seek, the display snapped back until the player confirmed, and slow confirmations flashed "No response".

- **Progress telemetry:** `snoofer.Control.Progress{Known, Playing, PositionMs, DurationMs, Rate, At}` is display-only, like `Meter`, and is excluded from revisions. Surfaces interpolate it: `position + (now − At) × Rate` while playing, clamped to the length.
  - The media dial carries it, and its `Value` changes only with the play state, so input is never rejected during playback.
  - Session keys keep showing Playing or Paused. The GUI's session cards get the same fields (position, rate, playing, sample time) in the status ViewData and interpolate them.
- **Scrubbing:**
  - Dial detents accumulate into one target (±5 s each, from the shown position), and the shown progress moves to it at once.
  - One seek goes out 250 ms after the last detent.
  - Further turns while a seek is pending start from the pending target.
  - The dial's status never shows Pending for seeks. The optimistic progress already shows the request, and a changing status would change the revision and reject the next turns. Only failures appear.
- **Optimistic progress:** while a seek is pending, progress shows the requested position, advancing if playing. The pending seek clears when the source reports a position within 3 s of the target. If that does not happen within 3 s, Snoofer quietly shows what the source reports instead of an error.
- **GUI:** session cards interpolate `Progress` between polls. The seek slider is not overwritten while dragged, and after release it shows the optimistic position.
- **Deck rendering:** with `Progress` known, the dial shows `m:ss / m:ss` and its track from the interpolated position at render time, at the idle refresh (150 ms).

## 4. Deck
- **Dial artwork:** when a dial's tile carries Artwork, the touch-strip panel draws it at 56 px on the left. The label, value and position track then move to the right column (x 72–192).
- **Progress:** `withPosition` accepts `m:ss / m:ss` values and turns them into a 0…1 progress track with no zero mark.
- **Default Media page:**
  - a session region on r1–r3 c1–c8;
  - Previous, Play/Pause, Next, Mute and Focus on the bottom row, keys 28–32 (r4 c1–c5; revised after review);
  - Up/Down at 18 and 27 for more sessions;
  - the media dial on dial 1 and Playback gain on dial 2.
- **Home:** a go-to key for Media and the media dial on dial 3.
- **Personal layout:** the same additions, at Home key 33 (r4c6) and dial 3 (both free). Home's existing media keys stay.

## 4a. Combined Media page (revised after review)
The Apps page (change app-audio) and the Media page were both sparse, so they are one deck page. The GUI keeps its separate App audio and Media screens.

| Row | Keys |
|---|---|
| r1 c1–c8 | media sessions (region) |
| r2 c2–c5 (keys 11–14) | app keys (region), above their dials. Superseded by §4b: one stacked region on r1–r3 shares rows between sessions and apps |
| r2c9, r3c9 | Up, Down |
| r4 c1–c5 (keys 28–32) | Previous, Play/Pause, Next, Mute, Focus |
| r4 c7 (key 34) | **Apps** filter |
| r4 c8 (key 35) | **Media** filter |

**Dials:** dial 1 is the media dial; dials 2–5 are a dial region of apps, paging in step with the app keys.

**Filters** are deck filters owned by their providers and saved in their settings. Filtered members are published Hidden, so deck regions skip them, bound keys go blank, and the GUI screens, which list every member, are unaffected.
- **`appaudio.deck-apps`:** a selection cycled by the key: All, Pinned or Off (setting `deck_apps`).
  - Pinned keeps only picked apps; Off hides every app, leaving the app keys and dials blank.
- **`nowplaying.deck-media`:** a toggle, On or Off (setting `deck_media_off`, so On is the default).
  - Off hides the session keys, the media dial and the transport keys on every page, Home's media dial included.
  - The toggle itself stays visible so it can be turned back on.

**Home:** the go-to key for Apps is removed, and Media's moves to key 34 (r4c7), so Media, Soundboard and Lights sit together. The personal layout is migrated the same way.

## 4b. Filters free space (revised after review)
Review found that filters only blanked keys. The space should go to whatever is still shown.

- **Stacked regions:** `Region.Sources` lists further sources that share the region's rows after `Source`. In each overflow set, rows are allotted in order:
  1. every source with remaining members gets one row, while rows last;
  2. leftover rows go to sources in order, up to what each needs (`ceil(remaining / row width)`).

  Each source fills its rows left to right, top to bottom. A source with no visible members takes no rows, so a filtered category's rows go to the others. Set counts follow the stacked sources until all are placed.
- **Hidden bindings yield:** a key or dial bound to a control that is published Hidden is free for a region covering it. Stream Deck's own controls (go-to and scroll keys) never yield, because their visibility depends on the expansion itself.
- **Media page:**
  - **Keys:** the combined page uses one stacked region on r1–r3 c1–c8 with sessions then apps.
  - **Dials:** an app dial region covers dials 1–5. Dial 1's media dial binding yields when media is off.
  - **Effect:** Apps Off gives sessions all 24 keys; Media Off gives apps all 24 keys and all five dials.
- **Keys and dials:** app keys and app dials are independent; logos identify apps. (Review confirmed that key-to-dial alignment does not matter.)
- **Editor:** a stacked region is labelled with all its sources ("Media sessions + Apps"). The source picker changes the first source; further sources are edited in the configuration.

## 5. GUI
- **Media screen** (sidebar, after App audio):
  - **Session cards:** artwork, title, artist, source (app or browser · site), a progress bar with a seek slider, Previous/Play/Next, Mute for tabs and a Focus marker (click to focus).
  - **Browser extension card:**
    - connection status per browser, with "update" warnings from the handshake;
    - bridge or Windows errors;
    - install steps: the store listings, or for development `npm run build` in `extension/` and load `dist/chrome` unpacked or `dist/firefox` as a temporary add-on.
- **Rebuilds:** the screen rebuilds when the session set changes. Progress updates in place at the 100 ms poll.

## 6. Scenarios
| Scenario | Handling |
|---|---|
| Two Brave tabs playing (YouTube and a podcast) | With the extension: two sessions, each pausable; the dial follows the last started. Without it: one Brave session, as Windows shows. |
| Spotify and a tab | Spotify comes from Windows sessions and the tab from the extension; both appear. |
| YouTube Next | Uses the page's `nexttrack` handler, captured by `page.js`. |
| Plain `<video>` page with no media session | The element is controlled directly; Next/Previous are hidden. |
| Extension not installed | The Windows path still works. The GUI card explains how to install it. |
| Extension or Snoofer out of date | The handshake is refused, and the card and popup say which side to update. |
| Browser closed | The connection drops, its sessions disappear, and its Windows session (if any) returns. |
| Long-running service worker | Kept alive by WebSocket traffic; it reconnects if Chrome restarts it. |
