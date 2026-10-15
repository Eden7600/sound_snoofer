# Snoofer Media

A browser extension that lets [Snoofer](../README.md) show and control the media playing in each tab. Browsers give Windows a single media session for all their tabs. With this extension, every tab or embedded player that has media becomes its own session in Snoofer, on the Stream Deck and in its Media screen.

It talks only to Snoofer on the same computer (`ws://127.0.0.1:47815`), with no pairing or account. Nothing leaves the machine.

## Develop

```sh
npm test          # logic and build tests
npm run build     # dist/chrome and dist/firefox
npm run package   # also dist/snoofer-media-<browser>-<version>.zip
```

- **Chrome, Brave, Edge:** open `brave://extensions` (or `chrome://extensions`), turn on Developer mode, choose **Load unpacked** and pick `dist/chrome`. After rebuilding, press the extension's reload button.
- **Firefox (128 or later):** open `about:debugging#/runtime/this-firefox`, choose **Load Temporary Add-on** and pick `dist/firefox/manifest.json`. Then click the toolbar button and **Allow on all sites**; Firefox makes site access optional.

Protocol 2 requires a matching Snoofer version; the popup explains which side needs updating. After upgrading to 1.2.0, reload already-open media tabs once. Snoofer shows **Reload this tab** and leaves page playback and rating commands unavailable until then; tab mute remains available. The new page helper cannot recover the old helper's captured handlers. Reloading lets the page register them again without duplicate commands.

On YouTube Music, bind **Like Track** and **Dislike Track** in the Deck editor. They follow Snoofer's selected media session. Press an inactive rating to select it, or the active rating to clear it. The keys show the observed page state, including changes while paused; unknown or unsupported player state cannot be rated. This uses the signed-in page's current player controls, so website changes can make ratings unavailable. No Google API setup is needed.

`../scripts/check-extension.cjs` checks the page script in Chrome. It also runs the built extension end to end in Brave against a stand-in bridge.

## Layout

| File | Role |
|---|---|
| `src/page.js` | Page-world content script: finds `<audio>`/`<video>` and the page's media session, reports state and runs commands. |
| `src/bridge.js` | Isolated-world content script: relays between `page.js` and the background. |
| `src/background.js` | Keeps the WebSocket to Snoofer, merges reports into sessions, sends artwork and runs commands. |
| `src/logic.js` | Pure helpers shared by the background and popup, tested with Node. |
| `src/popup.*` | Toolbar popup: connection status, site access (Firefox) and the Snoofer port. |
| `build.mjs` | Generates each browser's manifest and packages the zips. |

## Publish

- **Version:** set it in `package.json`, then run `npm run package`.
- **Chrome Web Store:** upload `dist/snoofer-media-chrome-<version>.zip`.
  - **Single purpose:** show and control tab media from the Snoofer desktop app.
  - **Permissions:**
    - `tabs` reads titles and mutes tabs;
    - `storage` keeps the port;
    - `scripting` reaches tabs already open at install;
    - host access runs the media script on pages and fetches artwork.
- **Firefox Add-ons:** upload `dist/snoofer-media-firefox-<version>.zip`.
  - **Add-on ID:** `snoofer-media@sound-snoofer` (`GECKO_ID` in `build.mjs`). It is permanent after the first upload; change it before then if you want another.
  - **Data collection:** none is declared, because data goes only to the local Snoofer app.
- **Protocol:** the message format is `PROTOCOL` in `src/logic.js` and `protocol` in `plugins/nowplaying/bridge.go`. Change both together. Snoofer refuses a mismatch, and the popup says which side to update.
