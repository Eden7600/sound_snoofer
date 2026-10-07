# Meetings design

## Sources
Protocol facts were read from the official Insta360 Webcam Stream Deck plugin 1.1.1.3 and the Elgato Discord plugin 2.4.0.123 (downloaded by the user), plus Discord's RPC documentation. Nothing was copied; both integrations are reimplemented for interoperability. Nothing below is hardware-verified yet.

## Insta360 Link 2 (`insta360` plugin)
**Device.** Vendor 0x2E1A, product 0x4C04, exposed by Windows as a UVC camera (interface 0) and a microphone. There is no vendor HID interface. Other Link models are out of scope and shown as unsupported.

**Transport.** A companion `snoofer-camera.dll` (C++, MSVC, built like the media DLL) enumerates DirectShow video input devices, matches the device path `vid_2e1a&pid_4c04`, binds the filter and uses `IKsControl::KsProperty` on the camera's extension units. It never opens a video stream, so it coexists with Discord, browsers and the Insta360 Link Controller.
- Extension units are kernel-streaming property sets: XU1 `{FAF1672D-B71B-4793-8C91-7B1C9B7F95F8}` and XU2 `{E307E649-4618-A3FF-82FC-2D8B5F216773}`. The node for each set is found by probing the filter's nodes, not by assuming node numbers.
- Writes read the control's length first (a GET with no buffer reports the size), then SET a zero-padded payload of that length. Reads use the reported length. Firmware varies these lengths, so fields are parsed only when present.
- Calls run on the plugin's single worker thread, which owns COM for the DLL. A call that blocks marks the camera unavailable; no second caller is started.

**Commands.**

| Control | Write | Readback |
| --- | --- | --- |
| Privacy | XU2 selector 0x0F, 1 byte (1 on, 0 off) | status flags bit 0 |
| Tracking | XU1 selector 0x02 (video mode): Off = Normal (0), Single = AutoComposition (1), Group = AutoFraming (7) | status byte 0 |
| Framing | XU1 selector 0x13, 1 byte (Head 1, Half body 2, Full body 3); the AI-zoom bit (XU1 0x1B bit 0) is set first if off | XU1 0x13 |
| Reset position | Normal: XU1 selector 0x1A absolute pan/tilt (0, 0). Tracking: re-send the current mode | status pan and tilt |

The video-mode write is the status layout with only the mode set and the "leave unchanged" pose (pan, tilt and roll 3610, zoom 0, host pitch −450), as the official plugin sends.

**Status.** The worker reads XU1 0x02 every two seconds and after each write: byte 0 is the mode, byte 1 its state (detecting, working, lost), and the flags word at 0x36 holds privacy in bit 0. Framing is read from XU1 0x13. Device presence is re-enumerated every five seconds.

**Rules.**
- While privacy is on, tracking, framing and reset are unavailable (the camera ignores them) with the status Privacy.
- Full body is refused while Group tracking is active, matching the camera.
- A write is verified by status reads within three seconds; until then the control shows Pending, and a mismatch shows Failed. No automatic retry.
- Absent camera: controls are unavailable with "No camera". A connection report `insta360.app-camera` (Required: No) describes presence and the last error.

## Discord (`discord` plugin)
**Application.** The user creates a Discord application in the Developer Portal. As its owner they can use RPC without Discord's approval (Discord allows owners and up to 50 testers). They add `http://127.0.0.1` as an OAuth2 redirect and put `client_id` and `client_secret` in the plugin's settings in snoofer.json; docs/plugins.md lists the steps. Without them the Discord card shows "Setup needed".

**Transport.** The Windows named pipe `\\.\pipe\discord-ipc-0` to `-9`, first that connects. Frames are `[opcode u32 LE][length u32 LE][JSON]`: 0 handshake `{"v":1,"client_id":…}`, 1 frame, 2 close, 3 ping, 4 pong. Requests carry a nonce and responses are matched to it; a request without a response within five seconds fails.

**Authorization.**
1. If the settings hold a refresh token, refresh it at `https://discord.com/api/oauth2/token`. Otherwise, when the user presses Connect, send `AUTHORIZE` with the scopes `rpc`, `rpc.voice.read`, `rpc.voice.write`, `rpc.video.read`, `rpc.video.write`, `rpc.screenshare.read` and `rpc.screenshare.write`. Discord shows its approval popup; the code is exchanged at the token endpoint with the client secret and the redirect.
2. `AUTHENTICATE` with the access token. The refresh token is saved through `SaveSettings` (compare-and-swap) and never appears in reports or diagnostics.
3. `invalid_grant` discards the token and requires Connect again. Network failures retry with backoff (5 s doubling to 60 s). Authorization is never requested automatically, so the popup never appears unexpectedly.

**State.** `GET_VOICE_SETTINGS`, `GET_SELECTED_VOICE_CHANNEL`, and subscriptions to `VOICE_SETTINGS_UPDATE`, `VOICE_CHANNEL_SELECT`, `VIDEO_STATE_UPDATE` and `SCREENSHARE_STATE_UPDATE`. A reconnect re-reads everything; nothing is replayed.

**Controls.**

| Control | Command | Available |
| --- | --- | --- |
| Mute | linked, see below | connected |
| Deafen | `SET_VOICE_SETTINGS {deaf}` | connected |
| Camera | `TOGGLE_VIDEO` | in a voice channel |
| Screen share | `TOGGLE_SCREENSHARE` | in a voice channel |
| Channel | status: current voice channel, or "Not in a call" | connected |
| Leave | `SELECT_VOICE_CHANNEL {channel_id: null}` | in a voice channel |

Writes are verified by the following update within three seconds (Pending, then Failed), with no automatic retry. A connection report `discord.app-client` covers the pipe and authorization. Discord being closed is normal: controls are unavailable with "Discord closed" and the pipe is retried every five seconds.

## Mic mute link
Snoofer's Mic mute stays one persisted preference (`mic_muted`). The `discord` plugin requires `audio`, which gains a small API: the current Mic mute preference and a request to change it (the same edit as the Mic mute control).
- **Snoofer to Discord.** While Discord is authenticated and not deafened, the plugin imposes Discord mute = Mic mute: on connect, after each preference change, and when an update shows a mismatch it did not cause. At most one write is in flight.
- **Discord to Snoofer.** A Discord mute change that Snoofer did not request (Discord's button or keybind) is a user action on another surface: it becomes the Mic mute preference through the audio API, which saves it and applies it to Voicemeeter. This is a deliberate exception to "native readback never becomes preference": Discord's toggle is a user control, not a readback. A change matching Snoofer's pending write is only its acknowledgement.
- **Deafen.** Discord reports itself muted while deafened. Mute changes are neither adopted nor imposed while deafened; on undeafen, Snoofer re-imposes Mic mute.
- **Connect.** On connect, Snoofer's preference wins; Discord's earlier state is not adopted.
- **Discord mute control.** `discord.mute` toggles the same preference. It mirrors `audio.mic-mute`, so automatic deck layouts show only one of them, and it shows Pending until Voicemeeter and Discord agree.
- Mic stack Off does not change the link: the preference persists either way.

## Meetings screen
A new `meetings` screen between App audio and Media, with two cards.
- **Camera:** Privacy, Tracking, Framing, Reset position and the camera state (Working, Detecting, Lost).
- **Discord:** Channel, Mute, Deafen, Camera, Screen share and Leave; Connect while unauthorized; "Setup needed" without credentials.
- A disabled plugin shows its card with an Enable button, as on Lights and Media.

Deck icons, all code-drawn: `camera-privacy` (and `-off`), `tracking`, `framing`, `camera-reset`, `discord-deafen` (and `-off`), `discord-video`, `screen-share` and `call-leave`. Discord mute reuses the mic-mute icon. No default deck layout changes.
