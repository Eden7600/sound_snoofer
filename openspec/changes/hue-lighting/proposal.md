# Hue lighting
## Why
Control Philips Hue scenes and room lighting, and the Hue Sync PC app, from Snoofer's Stream Deck and GUI without Elgato software.
## What Changes
- Optional `hue` plugin for a local Hue Bridge (CLIP v2): LAN discovery, link-button pairing, scene recall keys, and separate brightness and color-temperature dials for one configured room or zone.
- Optional `huesync` plugin for the Hue Sync PC app's local third-party control socket: Sync on/off, sync brightness dial, mode and intensity.
- Scene controls work with the existing `auto_controls` prefix to fill Stream Deck pages.
- New code-drawn deck icons and UI contract vocabulary for Hue controls.
## Impact
Two independent plugins with no audio or Stream Deck dependency, both disabled by default, with `no_hue` and `no_huesync` build exclusions. `github.com/gorilla/websocket` and `golang.org/x/net` (`dns/dnsmessage`) become direct dependencies; both already exist in go.sum through Wails, so no new module downloads. The bridge application key is stored in the ignored snoofer.json. No audio routing, recorder or existing layout changes.
