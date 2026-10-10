# Scene artwork brightness
## Why
Scene artwork already darkens each wedge by its light's brightness, but only within 55–100%, so a scene at 10% looks nearly as bright as one at 100%. The user wants the disc to tell bright scenes from moody, dark ones at a glance.

## What Changes
- **Range:** each wedge's level follows the brightness the scene sets on its lights, over a wider range: from 18% at the dimmest to full at 100% brightness.
- **Curve:** a gentle curve keeps mid-brightness scenes distinct.
- **Visibility:** the floor keeps the darkest wedges visible against the dark deck.
- **Defaults:** colours without a brightness stay at full level. Palettes (dynamic and brightness-only scenes) use their entries' brightness the same way.

## Impact
- **Code:** `plugins/hue/artwork.go` and its tests.
- **UI contract:** wording.
- **Scope:** affects scene keys and the GUI's Lights scene cards, which share the artwork.
