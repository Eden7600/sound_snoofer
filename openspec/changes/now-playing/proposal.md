# Now playing
## Why
Today the deck's media keys send blind key presses. Windows routes them to one "current" session it chooses, and nothing shows what is playing. Windows media sessions expose every player's title, artwork, position and controls. Chromium browsers, though, publish only one session for the whole browser, so two playing tabs cannot be told apart or controlled separately.
## What Changes
- **`nowplaying` plugin** with two sources:
  - **Windows media sessions:** every app's media session through a small C++/WinRT companion DLL, `snoofer-media.dll`. Each session has app, title, artist, album, cover art, play state, position, length and the controls it allows.
  - **Browser bridge:** a bundled Manifest V3 extension for Brave, Chrome and Edge. It reports every tab and frame that is playing media and executes play, pause, next, previous, seek and tab mute. It connects to Snoofer over a local WebSocket with a per-install token. While a browser's extension is connected, that browser's single Windows session is replaced by its tabs.
- **Focus:** one session has focus.
  - **What sets it:** focus follows the session that most recently started playing, or the last one you pressed.
  - **What follows it:** the media dial and transport keys act on the focused session.
- **Controls:**
  - **Session keys:** a key per session (a collection for deck regions) shows its artwork and state; pressing it plays or pauses that session and focuses it.
  - **Media dial:** shows artwork, title and progress; turning seeks 5 s per detent and pressing plays or pauses.
  - **Transport keys:** Previous, Play/Pause, Next and Mute (tabs only).
- **Deck:**
  - **Dial artwork:** dials can show artwork on the touch strip.
  - **Progress:** a `m:ss / m:ss` value draws a progress track.
  - **Media page:** the default layout gains a Media page; Home gains a go-to key and the media dial.
- **GUI:** a Media screen with session cards (artwork, title, artist, source, progress, transport, seek and focus) and a Browser extension card (status, Save extension files, instructions).
## Impact
- **Code:** new `internal/mediasessions` (Go wrapper plus C++ companion), `plugins/nowplaying`, `extension/` (embedded), the Stream Deck dial renderer and defaults, the GUI and the build script.
- **Native code:** the companion follows the soundboard DLL pattern (C ABI, owned thread). It sits outside the Voicemeeter adapter, as the Windows audio session code already does.
- **Unchanged:** the `media` plugin's key presses, which remain available.
