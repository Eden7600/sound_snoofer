# Snoofer

Snoofer runs in the Windows tray and hosts optional compiled plugins. Open controls from the tray. Closing controls leaves Snoofer running; Quit releases its resources without stopping the recorder or resetting mixer routes.

The default build includes Audio, VR, Stream Deck and Windows media. Enable or disable them in the Plugins tab. Changes show dependent plugins together and restart the application after saving. Ordinary control and layout edits apply live. Missing hardware is shown as disconnected; failed plugin branches can be retried with r.

## Controls

Use Tab to switch Controls/Plugins, arrows to select, Enter to edit or press, +/- for gains, and q to close the controls window. Selections support arrows and Enter; Esc cancels. Page Up/Down, Home and End navigate long views.

Normal microphone/playback and VR microphone/playback are separate sections. SteamVR running makes VR effective. Overridden Normal sections remain editable and explain that their edits apply outside VR. Mute, gain and recording preferences are shared. Unknown engine or device state is not reported as verified success.

Microphone Off disconnects managed mic routing without stopping playback or recorder transport. Muting does not change the selected microphone. Recorder transport remains available as Stream Deck semantic bindings; the removed Actions section stays removed. An external engine-restart request that may interrupt recording produces an Enter-only confirmation under System.

## Stream Deck

The integrated configurator edits pages, Home, shared positions, keys and dials. Save commits a validated draft live; Cancel discards it. The effective-position preview shows the merged binding. Device layouts can be edited while hardware is disconnected.

The sixth dial rotates through pages with wraparound; press it for Home. The first five dials are assignable. Shared positions are reserved on every page. Missing-plugin bindings remain visible as Unavailable and become usable again when their original provider returns.

See [plugin and configuration guide](plugins.md) for build composition, settings ownership and external plugin authoring.


The Bubble Tea interface uses blue section headers, a bordered viewport, aligned values and a highlighted selection. The header identifies the active tab; status and keyboard hints remain at the bottom. NO_COLOR preserves text selection markers without color. Terminals smaller than 42×10 show a resize notice and block hidden edits or confirmations.
