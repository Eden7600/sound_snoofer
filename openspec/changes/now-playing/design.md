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
### Extension (`extension/`, embedded in `snoofer.exe`)
The extension is Manifest V3 with permissions `tabs` and host access to `<all_urls>` (for content scripts and artwork fetches).
- **`page.js`** runs in the page's main world, in all frames, from `document_start`:
  - It wraps `navigator.mediaSession.setActionHandler` to remember page handlers such as YouTube's `nexttrack`.
  - It tracks `<audio>` and `<video>` elements through captured `play`, `pause`, `timeupdate`, `durationchange` and `ended` events.
  - It reads `navigator.mediaSession.metadata` and `playbackState`.
  - It reports through a `CustomEvent` on state changes, and every second while playing.
- **Commands in the page:** a page handler is used first (`play`, `pause`, `nexttrack`, `previoustrack`, `seekto`). Otherwise the playing element is controlled directly (`play()`, `pause()`, `currentTime`).
- **`bridge.js`** runs in the isolated world and relays between page events and a `chrome.runtime` port.
- **`background.js`** (service worker):
  - It keeps one WebSocket to `ws://127.0.0.1:<port>/nowplaying`, reconnecting with backoff and pinging every 20 s, which also keeps the worker alive.
  - It merges frame reports per tab and adds the tab title, site and mute state.
  - It turns artwork URLs (or the favicon) into 64 px PNGs with `OffscreenCanvas`.
  - It sends snapshots, executes commands and mutes tabs through `chrome.tabs.update`.
- **`config.js`** holds `{port, token}` and is written when the files are saved.

### Server (in the plugin)
- **Listener:** `127.0.0.1:<port>` (default 47815) serves only `/nowplaying`, using `gorilla/websocket`, which is already a dependency.
- **Origin:** must start with `chrome-extension://`, so pages cannot connect.
- **Handshake:** the first message is `hello{token, version, browser}`. A wrong token closes the connection. `Settings.Token` is 32 random bytes in hex, created on first start.
- **Version:** an extension older than the embedded one is flagged "Update the extension" but still works.
- **Messages from the extension:** `sessions{[…]}`. Each session has `id` (`tab:frame`), `tab`, `site`, `title`, `artist`, `album`, `art` (base64 PNG, sent only when it changes, otherwise `artKey`), `state`, `positionMs`, `durationMs`, `updatedMs`, `rate`, `canNext`, `canPrev`, `canSeek`, `muted` and `audible`.
- **Messages to the extension:** `command{id, op, value}`.
- **Bounds:** messages are limited to 1 MiB and sessions to 64 per browser. One connection per browser name; a newer connection replaces the older one.
- **Saving the extension:** **Save extension files** writes the embedded files and `config.js` to `<config folder>/browser-extension`, next to `snoofer.json` (like `soundboard-cache`). The GUI shows that path and the steps: open `brave://extensions`, enable Developer mode, Load unpacked, pick the folder; reload after saving again.

### De-duplication
While a browser's extension is connected, Windows sessions from that browser are hidden. Matching is by AppUserModelID: Brave is `Brave` or `Brave.*`, Chrome is `Chrome` or `Chrome.*`, Edge is `MSEdge` or `MSEdge.*`. Without the extension, the browser's single Windows session appears as usual.

## 3. Sessions and focus (`plugins/nowplaying`)
- **Session control:** each session gets `nowplaying.s-<hash of source and id>`.
  - **Labels:** Label is the title (falling back to the tab title or app name); ShortLabel is the title.
  - **Value:** Playing, Paused or Stopped. Status is Pending until a command is observed, or Failed.
  - **Artwork and grouping:** Artwork is the cover, or the favicon for tabs. Icon is `media-play`. Collection is `nowplaying.sessions` ("Media sessions"), with Order by most recently started first.
  - **Press:** toggles play/pause and focuses the session.
- **Focus rule:** the session that most recently changed to Playing, unless you pressed a session or a transport control within the last 30 s. A focused session that disappears gives way to the next by the same rule.
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

## 4. Deck
- **Dial artwork:** when a dial's tile carries Artwork, the touch-strip panel draws it at 56 px on the left. The label, value and position track then move to the right column (x 72–192).
- **Progress:** `withPosition` accepts `m:ss / m:ss` values and turns them into a 0…1 progress track with no zero mark.
- **Default Media page:**
  - a session region on r1 c1–c8;
  - Previous, Play/Pause, Next and Mute at keys 10–13 (r2 c1–c4);
  - Focus at key 14;
  - Up/Down at 18 and 27 for more sessions;
  - the media dial on dial 1 and Playback gain on dial 2.
- **Home:** a go-to key for Media and the media dial on dial 3.
- **Personal layout:** the same additions, at Home key 33 (r4c6) and dial 3 (both free). Home's existing media keys stay.

## 5. GUI
- **Media screen** (sidebar, after App audio):
  - **Session cards:** artwork, title, artist, source (app or browser · site), a progress bar with a seek slider, Previous/Play/Next, Mute for tabs and a Focus marker (click to focus).
  - **Browser extension card:**
    - connection status per browser;
    - an "update" warning;
    - **Save extension files** with the saved path;
    - the install steps;
    - **Reset token**, which also requires saving and reloading again.
- **Rebuilds:** the screen rebuilds when the session set changes. Progress updates in place at the 100 ms poll.

## 6. Scenarios
| Scenario | Handling |
|---|---|
| Two Brave tabs playing (YouTube and a podcast) | With the extension: two sessions, each pausable; the dial follows the last started. Without it: one Brave session, as Windows shows. |
| Spotify and a tab | Spotify comes from Windows sessions and the tab from the extension; both appear. |
| YouTube Next | Uses the page's `nexttrack` handler, captured by `page.js`. |
| Plain `<video>` page with no media session | The element is controlled directly; Next/Previous are hidden. |
| Extension not installed | The Windows path still works. The GUI card explains how to install it. |
| Wrong or old token | The connection is refused, and the card says "Reconnect: save and reload the extension". |
| Browser closed | The connection drops, its sessions disappear, and its Windows session (if any) returns. |
| Long-running service worker | Kept alive by WebSocket traffic; it reconnects if Chrome restarts it. |
