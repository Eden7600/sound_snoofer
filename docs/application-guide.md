# Snoofer

Snoofer runs in the Windows tray and hosts optional compiled plugins. Open controls from the tray. Closing controls leaves Snoofer running; Quit releases its resources without stopping the recorder or resetting mixer routes.

The default build includes Audio, VR, Stream Deck and Windows media. Enable or disable them in the Plugins tab. Changes show dependent plugins together and restart the application after saving. Ordinary control and layout edits apply live. Missing hardware is shown as disconnected; failed plugin branches can be retried in Diagnostics.

## Controls

Choose Audio, Soundboard, Stream Deck, Plugins or Diagnostics in the sidebar. Up/Down moves along the sidebar; Tab moves focus between controls. Text fields support native editing and paste; Enter applies and Escape cancels. Dropdowns and toggles apply their selected values. Closing the window leaves plugins running.

Normal microphone/playback and VR microphone/playback are separate sections. SteamVR running makes VR effective. Overridden Normal sections remain editable and explain that their edits apply outside VR. Mute, gain and recording preferences are shared. Unknown engine or device state is not reported as verified success.

Mic stack enablement is a master switch for both Normal and VR. Disabling disconnects managed mic inputs, ASIO input patches, monitoring and processing-return sends without stopping playback or recorder transport; Volt remains on A1 for playback. Mic stack target selects Automatic or a microphone and remains editable while disabled. Target edits and SteamVR changes never re-enable the stack. Enabling restores the active profile target through normal availability/fallback rules. Mic mute remains independent and does not disconnect devices or change the target. Recorder transport remains available as Stream Deck semantic bindings; the removed Actions section stays removed. An external engine-restart request that may interrupt recording exposes an explicit confirmation control in Audio.

## Stream Deck

The integrated editor shows a 9×4 key grid and five assignable dials plus the reserved page dial. Click a position or use arrows then Enter to select it without triggering its action. The searchable binding inspector shows shared/automatic ownership. Save commits a validated draft live; Discard restores the saved layout. Key labels use zero-based physical indices. Device layouts can be edited while hardware is disconnected.

The sixth dial rotates through pages with wraparound; press it for Home. The first five dials are assignable. Shared positions are reserved on every page. Missing-plugin bindings remain visible as Unavailable and become usable again when their original provider returns.

See [plugin and configuration guide](plugins.md) for build composition, settings ownership and external plugin authoring.


Desktop controls use the existing mascot, dark neutral surfaces, cyan active state and focus outlines, amber pending/fallback indicators, and red error/mute states. Labels always accompany color. Audio displays live meters; Soundboard shows searchable artwork and playback state. The responsive interface supports windows from 800×600; larger windows expose the deck inspector beside its grid.

Stream Deck gain dials show live digital level bars below their gain readouts. A1/A2 show the loudest channel on that output bus; Mic shows the active input after mute. Green/amber/red runs from -60 to 0 dBFS. LEVEL N/A means readings are unavailable or stale. These sampled meters are not calibrated analog VU or guaranteed true-peak meters.
