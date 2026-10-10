# Design
## Model
`resource` gains the light fields this needs. They are merged from event updates like the others:
- `metadata.function`;
- `color.xy` and `color.gamut`, the gamut's red, green and blue corners;
- `color_temperature.mirek_valid`, which is true while the light shows a white temperature;
- `gradient.points` and `gradient.points_capable`.

`model.roomLights(groupID)` returns the room's lights, each with:
- function;
- on and brightness;
- whether it is colour-capable;
- whether it is in colour mode: colour-capable, `mirek_valid` false, and not a white-only lamp;
- its xy, gradient points and gamut. Gamut C is the fallback when the bridge omits it.

## Saturation
- **White point:** D65 (x 0.3127, y 0.3290).
- **Measuring:** a colour's saturation is its distance from white divided by the distance from white to the gamut's edge along the same ray, from 0 to 1.
- **Hue:** colours within 0.005 of white have no defined hue. They are left alone, and are not counted in the average.
- **Dial value:** the average saturation of the room's colour-mode lights, in 5% steps.
- **Applying a target:** every colour-mode light gets xy = white + (edge − white) × target along its own ray. A gradient light writes every point; other lights write `color.xy`.

## Staging
- **Per dial:** each dial keeps one staged target.
  - The Decor and Mixed dials stage a brightness, starting from the group's observed average and moving 5% per tick (1–100%).
  - The Saturation dial stages a fraction, starting from the observed average.
- **Expiry:** all stages expire together, 10 s after the last turn of any light-group dial.
- **Global apply:** a press on any light-group dial applies every staged target (Decor, Mixed and Saturation) at once. Only when nothing is staged does a press toggle Decor or Mixed (Saturation does nothing).
- **Presentation:** while staged, the dial shows the target with an arrow (`→ 60%`) and is subdued. A status note would render as Error on the deck, so none is used. Without a stage it shows the group's state: the average brightness of its lit lights, Off when none is lit, and N/A when unknown.

## Applying
- **Building the batch:** a press builds one write per light, for every staged dial:
  - Decor or Mixed with a stage: `{on: true, dimming: brightness}` for each light of that function.
  - Decor or Mixed without a stage: `{on: false}` when any light is lit, otherwise `{on: true}`.
  - Saturation: the colour write for each colour-mode light.
- **Pacing:** writes go out one at a time, at least 100 ms apart, about the bridge's limit for light commands. A throttled write doubles the gap, up to 2 s, and is retried.
  - A new press replaces any queued writes for the same dial; lights already written keep their values.
- **Confirmation:**
  - Each written light is confirmed when the bridge reports the requested on state, brightness within 2%, or xy within 0.01.
  - Lights unconfirmed 3 s after their write are reported in the dial's status as "Not applied: 2 lights".
  - A failed write reports its error; nothing is retried automatically.
- **While applying:** while the dial has writes queued or lights awaiting confirmation, it reports Pending (the deck shows Wait, like a pending key) and is subdued.
- **Disconnect:** the queue and stages are dropped.

## Availability
| Situation | Dial |
|---|---|
| Room has no light of the dial's kind (for Saturation: no colour-capable light) | Hidden |
| Lights exist, but none is in colour mode | Saturation shows White, and presses do nothing |
| Not live, not connected, or Hue Sync streaming | Unavailable |

## Deck
- **Icons:** Decor and Mixed use `hue-brightness`. Saturation uses a new `hue-saturation` drop icon, in the lights area colour.
- **Default Lights dials:** Playback (1), Saturation (2), Decor (3), Mixed (4), Brightness (5).
- **Personal layout:** the user's Lights page is updated the same way while Snoofer is stopped, with a backup.
