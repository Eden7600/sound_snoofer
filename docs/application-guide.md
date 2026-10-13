# Snoofer

Snoofer runs in the Windows tray and hosts optional compiled plugins. Open controls from the tray. Closing controls leaves Snoofer running; Quit releases its resources without stopping the recorder or resetting mixer routes.

The default build includes Audio, VR, Stream Deck and Windows media. Enable or disable them in the Plugins tab. Changes show dependent plugins together and restart the application after saving. Ordinary control and layout edits apply live. Missing hardware is shown as disconnected; failed plugin branches can be retried in Diagnostics.

Start at Login, at the top of Diagnostics, adds a per-user Windows Run entry that starts this executable with its configuration in the tray at sign-in, and removes it when turned off. It shows what Windows holds: Off with a note when Task Manager's Startup apps disabled it or another copy is registered (turning it on replaces that), and unavailable under `--dry-run`.

## Controls

Choose Audio, Soundboard, Stream Deck, Plugins or Diagnostics in the sidebar. Up/Down moves along the sidebar; Tab moves focus between controls. Text fields support native editing and paste; Apply/Add or Enter applies, and Escape cancels. Dropdowns and toggles apply their selected values. Closing the window leaves plugins running.

Normal microphone/playback and VR microphone/playback are separate sections. SteamVR running makes VR effective. Overridden Normal sections remain editable and explain that their edits apply outside VR. Mute, gain and recording preferences are shared. Unknown engine or device state is not reported as verified success.

Mic stack enablement is a master switch for both Normal and VR. Disabling disconnects managed mic inputs, ASIO input patches, monitoring and processing-return sends without stopping playback or recorder transport; Volt remains on A1 for playback. Mic stack target selects Automatic or a microphone and remains editable while disabled. Target edits and SteamVR changes never re-enable the stack. Enabling restores the active profile target through normal availability/fallback rules. Mic mute remains independent and does not disconnect devices or change the target. Recorder transport remains available as Stream Deck semantic bindings; the removed Actions section stays removed. An external engine-restart request that may interrupt recording exposes an explicit confirmation control in Audio.

## Stream Deck

Supported decks: Stream Deck + XL, +, XL, MK.2 (and Original v2) and Neo, several at once. The integrated editor shows the selected model’s grid and dials: Device lists a layout per model and each remembered deck (by serial, recorded on first connection) that has its own layout. Decks of one model share its layout unless Own Layout is on; Name, Own Layout and Forget apply to the selected deck. Click a position or use arrows then Enter to select it without triggering its action. The searchable binding inspector shows shared/automatic ownership. Save commits a validated draft live; Discard restores the saved layout. Key labels use zero-based physical indices. Device layouts can be edited while hardware is disconnected.

The last dial (sixth on the + XL, fourth on the +) rotates through pages with wraparound; press it for Home. The other dials are assignable. Decks without dials reserve the bottom-right key as Pages: press for the next page, hold for Home; the Neo uses its touch buttons (left previous, right next, hold for Home) and shows the page on its screen. Paging is Independent (each deck its own page) or Mirrored (every deck follows the last page change where it has that page). Shared positions are reserved on every page. Missing-plugin bindings remain visible as Unavailable and become usable again when their original provider returns.

See [plugin and configuration guide](plugins.md) for build composition, settings ownership and external plugin authoring.


Desktop controls use the existing mascot, dark neutral surfaces, cyan active state and focus outlines, amber pending/fallback indicators, and red error/mute states. Labels always accompany color. Audio displays live meters; Soundboard shows searchable artwork and playback state. The responsive interface supports windows from 800×600; larger windows expose the deck inspector beside its grid.

Stream Deck gain dials show live digital level bars below their gain readouts. A1/A2 show the loudest channel on that output bus; Mic shows the active input after mute. Green/amber/red runs from -60 to 0 dBFS. LEVEL N/A means readings are unavailable or stale. These sampled meters are not calibrated analog VU or guaranteed true-peak meters.
