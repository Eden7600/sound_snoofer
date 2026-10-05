# VR, physical controls and audio recovery

Launch the replacement executable normally for tray mode. Open controls from the tray. Quit the previous Sound Snoofer instance before switching binaries; this build does not stop a running instance.

## Microphone and speaker mute

Mute Microphone uses native Voicemeeter mute on the active physical mic and Element return, including rehearsal return. It keeps the source, ASIO patch, device assignments and sends intact. Mute Speakers follows the current playback bus. Knob presses on A1/A2 add independent fixed-bus mute requests; clearing one request cannot defeat another. Pre-existing manual native mute is preserved, so removing Snoofer mute may still leave a manually muted strip silent.

Mute baseline ownership is stored beside the config in .mutes.json; do not remove it while Snoofer owns mute settings. Gain changes belong to Voicemeeter and are not saved/replayed at startup.

## SteamVR and headset profiles

Prefer Headset Mic and Prefer Headset Playback are independent persistent controls. Source Off still wins. Headset devices are eligible only while vrserver.exe is present in the same session; this does not establish whether a headset is worn or audio is healthy. Normal selections are preserved and return when the headset/runtime disappears.

Add a top-level vr object to config.json, adapting the example regexes to actual endpoint names. Names below are placeholders, not tested Index/Beyond presets. The ordered headsets list determines preference; each direction must match exactly one active WDM endpoint. Input 4 or 5 must be unreserved.

```json
"vr": {
  "input": 4,
  "headsets": [
    {"id": "my-headset", "label": "My headset",
     "microphone": "^REPLACE WITH MIC NAME$",
     "playback": "^REPLACE WITH PLAYBACK NAME$"}
  ]
}
```

Keep Windows on Voicemeeter is opt-in and works outside VR too. It protects playback and capture defaults for Console, Multimedia and Communications. Automatic targets are primary Voicemeeter Input and Out B3. Optional vr.playback_default and vr.capture_default accept exact active endpoint names or IDs for another installation. Ambiguous/missing targets are untouched. Repeated competing changes suspend protection; toggle it off/on to retry. Turning it off leaves current defaults in place. This does not alter per-application audio selections. The setter uses an isolated undocumented Windows policy interface and still needs live write acceptance on supported Windows versions.

## Stream Deck + XL

No Elgato software, plugin host, Python or replacement USB driver is needed. The supported product is Stream Deck + XL (USB 0FD9:00C6); original XL and smaller Plus are not supported by this build.

Encoders 1-3 adjust A1, A2 and active physical mic gain in 1 dB increments, clamped to -60..+12 dB. Press toggles native mute. Encoders 4-6 remain unassigned. The touch strip reports observed dB and native mute.

Default keys use four rows of nine, with empty space separating workflows:

1. Mic mute, output mute, monitor, processing, four blanks, Open controls.
2. Record/Stop, blank, record mic, record computer, mic stage, four blanks.
3. Reserved for future soundboard controls.
4. Rewind, Play/Pause, Forward, six blanks (zero-based keys 27, 28, 29).

Record/Stop uses observed recorder state and refuses to guess when unavailable or playing tape. Mic/output icons show LIVE/AUDIBLE or MUTED. Monitor and stage show PRE VST/POST VST; an asterisk marks a preference falling back to its effective state. Small amber accents identify pending, unavailable or failed controls without repainting unrelated keys.

The playback-output button and the knob for that bus toggle one combined Snoofer mute preference. Either can undo the other; other bus knobs remain independent. Preexisting native/manual mutes retain their ownership safeguards.

Media controls dispatch system media keys and do not report player state. Recorder/source actions obey the same guards as the TUI. Reconnect does not replay held keys or transport commands.

Optional top-level stream_deck.profiles entries contain serial and keys (up to 36 supported action names; empty string is unassigned). The list replaces the default key map for that serial. For example:

```json
"stream_deck": {
  "profiles": [{
    "serial": "REPLACE WITH DEVICE SERIAL",
    "keys": ["mic-mute", "speaker-mute", "record-toggle", "",
             "engine-restart", "vr-mic", "vr-playback", "defaults", "open-controls"]
  }]
}
```

Supported actions include the default bindings above plus a1-mute, a2-mute, engine-restart, vr-mic, vr-playback and defaults. Restart Snoofer after changing layouts. Only one connected + XL is managed in this first implementation.

## Engine recovery

Restart Audio Engine is available in controls and the tray; it can be mapped to a deck key. It restarts the Voicemeeter audio engine, not the process. Active/unknown recording or playback requires confirmation in controls. Snoofer does not automatically stop/start/resume the recorder. Current saved settings are reconciled after the engine responds.

Enable **Auto-recover Audio** in Controls to monitor the tested Universal Audio Volt on A1 with Potato. It is Off by default. In live mode, Snoofer registers a native output callback that passes samples through unchanged. After Volt returns, five seconds of transition grace and sustained missing callbacks qualify a restart, only with stable routing and stopped recorder transport. Preview never registers the callback. A missing companion DLL or another application owning the callback slot disables monitoring with a visible reason; Snoofer does not displace that application. Other A1 devices and editions do not qualify automatically.

Automatic attempts have a 60-second cooldown and a two-attempt limit per ten minutes, persisted across relaunch. An uncertain restart blocks automatic retries until explicit manual retry or observed processing recovery. Successful recovery requires advancing synchronized callback observations; audible output remains unverified. No level-meter or sample-rate heuristic triggers a restart. A stuck native call is reported as a stalled worker; Snoofer never creates a second concurrent DLL writer to force recovery.

## Verification so far

Automated tests and vet pass. A read-only Windows probe resolved both Voicemeeter endpoints and read all six default-role assignments. It made no changes. No + XL interface or VR audio endpoint was connected during acceptance checks. Direct hardware I/O, headset hotplug, default-setting writes and automatic-recovery listening acceptance remains unverified. The standalone probe reproduced Volt callback loss and resumption after manual restart; the production adapter passed native callback lifecycle validation. The race detector requires the unavailable CGO toolchain.

A TUI preview with an isolated config verified navigation, saved mic mute preference and Graph switching without mixer writes. Use --dry-run for your own preview; it also suppresses media-key injection and Windows default changes.
