# Third-party apps
## Why
Most Snoofer plugins bridge to something outside the process: Voicemeeter, a companion DLL, a USB device, an OS service, a LAN bridge or another app. Their health is scattered across ad hoc status strings, mixed with unrelated messages (the Stream Deck shares one string with the layout editor), or not published at all (Windows default protection, Element detection, callback counters). Users need one place that shows, for every integration, whether it is working, since when, what it last did and why it failed.
## What Changes
- A core connection report: `snoofer.Control` gains an optional typed `Connection` (state, endpoint, since, last activity, last error, ordered details). Providers publish one status-only control of Kind `connection` per integration.
- A new **Third-party apps** GUI screen that renders every report generically, with a summary, per-app cards, relative times and Copy details (per card and for all).
- Reports for every external integration: Voicemeeter Remote API, Voicemeeter audio-callback monitor, Voicemeeter recorder, ASIO interface, Element VST host, Windows default-device protection, SteamVR, Stream Deck hardware, soundboard playback companion and renderer, Hue Bridge, Hue Sync and Windows media keys.
- Each plugin tracks the state its report needs (timestamps, identities, versions, counters), without changing control behavior.
## Impact
Additive core API field, a new GUI screen and a contract update. No routing, recorder or command behavior changes. Reports are read-only and are never bindable on the Stream Deck. Copy details excludes credentials.
