# Interface and playback

The audio plugin resolves `studio.asio` in priority order before microphone and playback. The first uniquely present interface owns A1 and the clock. Installed ASIO drivers alone do not establish hardware presence.

Each entry contains `asio_pattern` (driver name), `presence_pattern` (physical WDM input name), and `inputs`: the desk and lav mono channels, each patched to both stereo sides. Zero means that microphone is unavailable on this interface. These are the existing two managed ASIO microphone strips.

```json
"asio": [
  {
    "asio_pattern": "(?i)^Universal Audio Volt$",
    "presence_pattern": "(?i)^INPUT 1/2 \\(Volt 2\\)$",
    "inputs": [1, 2]
  }
],
"playback": [
  {"driver": "wdm", "pattern": "(?i)steelseries.*arena"},
  {"driver": "asio", "pattern": "(?i)^Universal Audio Volt$"}
]
```

An empty ASIO list disables interface selection. Ambiguous matches stop routing rather than guessing. Unselected interfaces supply neither ASIO microphones nor ASIO playback, even if connected. Give a speaker-only interface `[0, 0]`; microphone priorities then fall through to webcam or the next eligible source. Normal and VR retain separate downstream priorities and runtime overrides. Interface priority is global.

Playback gain/meter and mute follow the selected listening destination, whichever bus it occupies. Routing never copies or resets gain. Playback mute is saved in Snoofer and imposed on Voicemeeter, including unmute. Native readback confirms or disputes that request; it never changes the preference.

This is a configuration changeover: remove the former top-level studio `asio_pattern`, `presence_pattern` and `asio_playback`; place ASIO in the ordinary playback list at the desired priority. Remove saved `bus_muted` and retain the intended effective playback mute as `playback_muted`. Replace fixed output dial bindings with `audio.gain-playback` and clear redundant bindings. Keep `audio.speaker-mute` as the semantic Playback mute binding.
