## Why
SteamVR can leave headset endpoints enumerated while unavailable and can move Windows defaults away from Voicemeeter. Users need independent headset microphone and playback preferences without losing their normal settings.
## What Changes
Add SteamVR-gated headset profiles, independent Prefer headset microphone/playback controls, and an optional Keep Windows defaults on Voicemeeter control. Support configurable Go regex matching rather than model-specific routing code. Index and Bigscreen Beyond 2E are initial hardware acceptance targets.
## Impact
Configuration, device eligibility, effective routing, Controls/Graph status, and a Windows default-endpoint adapter. Preserve Volt A1 ownership, source Off, Element fallback and existing recording behavior. Planning only; no implementation authorized by this proposal.
## Dependencies
Can be implemented independently of Stream Deck. Use shared control actions from mixer-surface-controls when integrating hardware buttons.
