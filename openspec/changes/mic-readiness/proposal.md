# Microphone readiness and editor
## Why
- **Silence is not always absence.** A lavalier with a hardware mute switch goes silent while muted. Activity metering then latches it silent, and Auto falls back to another microphone, although the lav is the one the user wants.
- **JSON still needed.** The Routing screen cannot set everything about a microphone:
  - Interface channels are a free-text field ("1" or "3,4"), so stereo is undiscoverable.
  - A microphone's kind (interface channels or a Windows device) cannot change after it is added.
  - Activity metering (how many options to meter, the silence threshold and delay) is configuration only.
  - Adding an interface writes the legacy `inputs: [1, 2]`, which silently maps channels to `desk` and `lav`.

## What Changes
- **Ready flag.** A microphone may be marked ready (`"ready": true`). Snoofer then never meters it or latches it silent, so Auto never skips it for silence. Anything that makes it unusable still applies: a missing device, an interface that does not map it, an unavailable interface.
- **Silent shown.** Microphone rows on the Routing screen show Silent while latched silent.
- **Editor:**
  - Each interface microphone's channels are Off, Mono (one channel) or Stereo (left and right channels).
  - A microphone's source can change between interface channels and a connected input device.
  - A Ready toggle per microphone.
  - An Activity card edits metering: Off or the number of options to meter, the silence threshold and the delay.
  - Adding an interface maps no channels.

## Impact
- **Code:** `internal/config` (field, edits), `internal/control` (metering skips ready microphones), `plugins/audio` (view), `app/web`, `scripts/check-gui.cjs`, `docs/ui-contract.md`.
- **Invariants:** none changed. Routing, patches and selection are unchanged except that ready microphones are never in the silent set.
