# Design

## 1. Meters and knobs
### Stream Deck + XL dials
- **Panel layout:** each 200×100 touch-strip panel shows:
  - label (cyan, top-left);
  - large value (right-aligned on the label row when it fits, otherwise below the label);
  - a thin **knob-position track** for numeric controls with a known range: gain is −60…+12 dB and soundboard volume is the same, and the zero-dB point is marked;
  - for metered controls, a **smooth meter bar** with a per-pixel gradient (green below −12 dBFS, amber to −3, red above), a **peak-hold tick** and a compact scale.
- **Non-metered controls** (for example Hue brightness) show the position track when their value parses as a percentage.
- **VU ballistics** live in the Stream Deck plugin as per-dial state, never in the renderer:
  - instant attack;
  - release at 24 dB/s;
  - peak hold for 1.5 s, then falling at 18 dB/s.
  - Expired or unknown readings still show `LEVEL N/A`, never silence. Bindings and the 500 ms expiry are unchanged.
- **Responsiveness:**
  - the audio worker samples levels every 50 ms (was 100 ms);
  - the deck frame ticker runs at 60 ms while any displayed dial has a meter, and 150 ms otherwise;
  - only changed key images are re-sent (existing caching), and the touch strip re-renders only when its presentation changes.
- **Tile fields:** a new `Tile.PeakDB` (with `PeakKnown`) and `Tile.Position` (0–1, with `PositionKnown`) carry the data. The renderer stays pure.

### GUI mixer strips
- **Meter:** a gradient bar with a smooth width transition (≈120 ms), a peak-hold marker computed client-side with the same timings, and scale ticks (−60/−30/−12/−3/0).
- **Position track:** under the readout, for gain values.
- **Refresh:** the poll interval drops from 200 ms to 100 ms while the Audio or Soundboard screen is shown; other screens keep 200 ms.

## 2. Mic controls hidden on the Stream Deck
- **When:** `Intent.Enabled` (mic stack) is false.
- **Controls marked `Hidden`:**
  - `audio.mic-mute`, `audio.source`, `audio.mode`, `audio.monitor`, `audio.gain-mic`;
  - `audio.record-mic`, `audio.record-tap`;
  - the profile variants `audio.normal-source|mode|monitor` and `audio.vr-profile-source|mode|monitor`.
- **What stays visible:** `audio.mic-stack` itself, so the stack can be turned back on.
- **Availability:** unchanged, so the GUI keeps these controls usable.
- **Contract change for `Control.Hidden`:** it means "no useful place on control surfaces right now".
  - The Stream Deck renders a blank key or dial, keeps the binding and ignores input on it; the dispatch path checks `Hidden` as well as `Available`.
  - GUI screens are not required to omit hidden controls. The Lights screen keeps hiding the sync Mode and Intensity rows by its own choice.
  - Hue sync controls keep publishing `Available: false` with `Hidden`.

## 3. Soundboard overlap
- **Setting:** `overlap` (bool, optional, default false) in soundboard settings, persisted with `Services.SaveSettings` on the plugin goroutine.
- **Toggle control:** `soundboard.overlap`, a toggle (press) with icon `soundboard-overlap`, short label "Overlap" and value On/Off. It shows on the Soundboard screen's Playback card and is bindable on the deck. A new code-drawn deck icon shows stacked play triangles.
- **Overlap off:** today's behavior; any press stops all voices, then plays.
- **Overlap on:** a press starts a new voice.
  - The plugin keeps up to **8** `ClipPlayer` instances, created lazily and reused, all on the existing locked plugin thread.
  - When 8 voices are active, the oldest is stopped and its player reused.
  - Each voice is polled every 200 ms; finished voices free their player.
  - A clip shows Playing while any of its voices plays.
- **Stop:** stops every voice.
- **Shutdown:** closes every player.
- **Preparation:** normalization still prepares one clip at a time; a newer press replaces a pending preparation, as today, and prepared clips start immediately.
- **Routing readiness:** checked per press, as today.
- **Mixing:** DirectSound mixes concurrent graphs on the same renderer, so the native companion is unchanged.

## 4. GUI chrome
- **Sidebar:** the "Connected / Plugin host" footer is removed. A failed state poll still raises the existing error notice, so loss of the host connection stays visible.
- **Header:** no longer shows audio health on any screen. It still shows transient "Sending" and "VR active" badges.
- **Health location:** an "Engine" status row with tone at the top of the Audio screen's Live controls card, and the existing row on Diagnostics.
