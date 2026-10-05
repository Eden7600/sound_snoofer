## Hardware and scope
The confirmed target is the single product Stream Deck + XL, not separate Plus and original XL units. Elgato documents PID 0x00C6 under VID 0x0FD9, 36 keys and six encoders. Its image rotation differs from the smaller Plus, so select an explicit product descriptor instead of guessing compatibility. Original XL and Plus are outside initial acceptance; unknown products are not opened.

Use Windows HID enumeration and asynchronous bounded I/O in an isolated platform adapter, preferably existing x/sys plus Windows APIs, preserving the no-CGO Windows build. Inspect descriptors and serial identity before opening the correct interface. A small framing layer validates report length, command, payload and indices. Use the official HID protocol directly, not the plugin SDK or stock application. No driver replacement. Access denied/busy becomes visible hardware unavailable; never terminate other software.

## Default layout
Provide a usable built-in layout and validated optional per-device configuration. No configuration file is required just to use the default mapping.

| Control | Default action |
| --- | --- |
| Encoder 1 | A1 gain; press toggles A1 native mute |
| Encoder 2 | A2 gain; press toggles A2 native mute |
| Encoder 3 | Active physical mic gain; press toggles microphone mute |
| Encoders 4-6 | Unassigned, visibly blank |
| Key row 1 | Record start, Record stop, Record computer, Record mic, Mic stage Pre/Post, Snippet play, Snippet stop, Loop, To VST |
| Key row 2 | Mic mute, Speaker mute, Monitor Off/Pre-VST/Post-VST, Processing Direct/Element, Media previous, Media play/pause, Media next, Media stop, Open controls |
| Key rows 3-4 | Unassigned; configurable supported actions |

Fixed-bus mute and playback-following mute compose as independent requests: a bus remains muted while either request requires it. Extend shared control actions accordingly before implementation; never let encoder unmute defeat speaker mute. Knob press on active mic never means Source Off. Display each gain target and observed dB on the touch strip with mute/pending/unavailable indicators. Unassigned controls do nothing. Touch gestures are not needed in v1. A1 may be the reserved Volt output even when A2 is the speaker target; label devices to avoid conflating these roles.

Buttons use the same recording/rehearsal guards as TUI. Source/stage/settings preferences persist through the shared service. Start/Stop/Play and media are edge-triggered one-shot actions, never saved/replayed or inferred from device connection. Post preferences retain existing effective Pre fallback. Recording feedback comes from native recorder observation, not last keypress. No generic scripts, macros or arbitrary executable bindings in initial scope.

## Ownership and responsiveness
Tray parent owns hardware lifetime; controls window may stay closed. One device worker owns each handle and reconnect loop; audio actions go to the shared actor, never directly to the DLL. Serial identity selects an optional profile. If serial identity is absent/ambiguous, use the safe default layout and report ambiguity rather than loading the wrong personal map. Closing controls does not stop hardware; application exit cancels reads/writes and releases handles with bounded shutdown.

Diff key state snapshots into press edges; initialize reconnect state without firing held keys. Validate and sign-extend encoder deltas. Coalesce gain ticks by target generation through the shared service, not by dropping input reports. Bound queues; overflow must be reported and invalidate uncertain commands rather than replaying transport. Held keys do not repeatedly start recording.

Rendering uses immutable observed state. Encode/cache only changed tiles, sanitize external labels and bound image sizes. Device output queues keep latest desired tile; complete or safely abort protocol transfers at defined boundaries before another image transfer. Slow image writes cannot block input handling, TUI or audio actor. Repaint on reconnect without replaying actions. Show pending yellow and unavailable/error red with text/icon distinctions; unknown native state must not look successfully applied. Gain/mute scheduling inherits the fast numeric path; image refresh targets 10 Hz with no guarantee hardware finishes inside that interval.

## Media controls
Emit Windows media virtual-key down/up pairs through an isolated SendInput adapter. Target the system's media handling, not a hard-coded player. Report dispatch failure/partial injection; do not claim playback state without a media-session observation API. No track artwork or player-specific integration in v1. Dry-run may read hardware and render Preview but must not inject media keys, change audio, operate recorder or change Windows defaults.

## References and acceptance
- [Elgato direct HID introduction](https://docs.elgato.com/streamdeck/hid/intro/)
- [Stream Deck + XL protocol](https://docs.elgato.com/streamdeck/hid/stream-deck-plus-xl/)
- [Windows SendInput](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-sendinput): input injection can fail, including integrity-level restrictions.

Protocol details come from official documentation; physical device interoperability remains unverified until acceptance. Tests must include report fixtures and malformed frames, not claim a physical deck was tested from mocks.

## Initial implementation refinements
The initial transport owns one connected + XL at a time, matching the user's single-device target. Per-serial profiles select a key action list; encoder defaults remain fixed. Multiple simultaneously connected decks and touch gestures are not initial acceptance requirements. Hardware actions are acknowledged serially, with adjacent relative ticks accumulated only for the same observed target. Image transfers are serialized and only changed tiles are uploaded. Hardware availability is reported in the tray; no separate device management tab is introduced. Profile layout changes take effect after restarting Snoofer.
