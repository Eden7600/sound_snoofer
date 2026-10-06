# Hue lighting
## Why
Control Philips Hue scenes, room lighting and the Hue Sync PC app from Snoofer's Stream Deck and GUI without Elgato software, as two halves of one lighting feature.
## What Changes
- One optional `hue` plugin with two halves:
  - **Room:** a local Hue Bridge (CLIP v2) with multi-interface LAN discovery, link-button pairing, scene recall and a brightness dial for one configured room or zone. (Revision 3 removed the color-temperature dial after use.)
  - **Sync:** the Hue Sync PC app's local third-party control socket, with Sync on/off, mode and intensity.
- The halves coordinate: the brightness dial follows sync while syncing, and recalling a scene stops sync first.
- Stable room scene slots for deck bindings, and scene controls usable with `auto_controls` prefixes.
- A dedicated Lights GUI screen. The Plugins page becomes enable/disable only.
- New code-drawn deck icons, blank rendering for empty slots, and UI contract vocabulary. The contract no longer implies Home dial index 2 is reserved.
- Hue bindings in the user's Home page block (rightmost four columns, dial 3).
## Impact
One plugin, disabled by default, with a `no_hue` build exclusion. `github.com/gorilla/websocket` and `golang.org/x/net` (`dns/dnsmessage`, `ipv4`) become direct dependencies; both already exist in go.sum through Wails, so nothing new is downloaded. The bridge application key is stored in the ignored snoofer.json. Third-party plugin fallback forms leave the Plugins page. No audio routing or recorder changes.
