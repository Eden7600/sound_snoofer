# Meetings
## Why
Calls need the webcam and Discord at hand alongside Snoofer's mic controls, without Insta360's or Elgato's Stream Deck plugins.
## What Changes
- An optional `insta360` plugin for the Insta360 Link 2: privacy, AI tracking (off, single person, group), framing style (head, half body, full body) and reset position, with live state read from the camera.
- An optional `discord` plugin over Discord's local RPC pipe, using the user's own Discord application: mute, deafen, camera, screen share, current voice channel and Leave.
- Discord mute is linked to Snoofer's Mic mute: one preference, applied to Voicemeeter and Discord.
- A Meetings GUI screen with a Camera card and a Discord card, new deck icons and UI contract vocabulary.
## Impact
Two plugins, disabled by default, with `no_insta360` and `no_discord` build exclusions. A new companion `bin/snoofer-camera.dll` (DirectShow and kernel-streaming properties, MSVC) built by `scripts/build-camera.ps1`. Discord credentials and the refresh token are stored in the ignored snoofer.json. The audio plugin gains a small mic-mute API for the link. No new Go dependencies. No routing, recorder or Voicemeeter parameter changes beyond the existing mic mute.
