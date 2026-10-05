## Why
Voicemeeter devices can stop passing audio while still appearing assigned. Sound Snoofer should identify actionable faults and offer bounded recovery without mistaking silence for failure or repeatedly interrupting healthy audio.
## What Changes
Add audio-health observation, an optional automatic recovery policy, and a Restart audio engine action available through Controls, tray and an optional Stream Deck binding. Preserve current choices and reconcile managed state after recovery.
## Impact
Native adapter, serialized controller, health state and control surfaces. Recovery restarts the audio engine, not the Voicemeeter process. Planning only; no live restart or implementation in this change. Direct Stream Deck integration is optional and depends on its separate proposal.
