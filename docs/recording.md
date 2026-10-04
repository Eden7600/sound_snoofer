# B1 recording

Enable `studio.recording` alongside `studio.voice` on Potato. The voice example
includes `"recording":{"computer_sources":["virtual:1"]}`. Both capture toggles
default Off; the mic tap defaults Pre. Existing voice-only saved choices remain
valid. The next edit saves recording choices in the same version-1 sidecar.

This profile owns every strip's B1 button. B2 remains the Element send and B3 the
application microphone. Primary VAIO is computer capture by default; optionally
include `virtual:3`. AUX (`virtual:2`) is reserved for the Element return.
Computer capture is independent of playback rules and speaker availability;
programs on other Windows endpoints are not captured automatically.

In Controls, recording settings are Record Computer Audio, Record Microphone,
then Recording Mic Stage (Pre/Post). Enter or Space changes a setting. Start
Recording and Stop Recording are under Actions and require Enter.
Mic capture requires a source other than Off. Enter or Space on Source opens
Desk/Lav/Webcam/Off; arrows select, Enter confirms, Escape cancels. Browsing does
not change routing. There is no separate Off hotkey or Microphone On/Off row.
Pre uses the effective mic including fallback. Post requires Element mode and
uses AUX. Switching to Direct automatically selects Pre; Post cannot be selected
in Direct. Returning to Element keeps Pre until you choose Post again. Older
saved Direct/Post choices are normalized to Pre on load.

Select Start recording or Stop recording and press Enter in live mode. Start
requires settled routes and at least one eligible source. It prepares stereo
bus capture with only B1 armed, disables tape playback into B1/B2/B3, verifies
the setup, and issues one REC command. An already compatible Recording state
makes Start a no-op. Playing/Paused/Unknown must be resolved before Start. Stop
is available even when routing conflicts. No transport choice is saved, and
launch, reload, reconnect, apply and watch never automatically start recording.

Set the destination folder and format in Voicemeeter's recorder before use.
Sound Snoofer preserves those settings, A tape playback sends, B1 gain/mute and
effects. This is a post-fader bus mix: Pre means before Element, not raw input.
A muted B1 is shown as a warning. Readback confirms parameters, not audio or a
successfully saved file. Verify a short recording in the chosen folder.

Source edits affect an ongoing recording, with a short disable-before-enable
gap when switching taps or devices. Recording-only edits leave Discord routes
alone. Both inclusions Off makes silence; it does not stop transport. Quit or
dry-run also leaves transport running. Use Stop explicitly.

Transport errors/timeouts are never automatically retried. Inspect fresh status
before trying again. External transport is observed. Incompatible active recorder
settings freeze B1 reconciliation until stopped; voice rules continue. After
preparation, live operation restores tape B sends to Off while stopped; active
conflicts are reported instead of rewriting the active recorder setup.

The API uses `Recorder.record/stop/pause/play`, `Recorder.mode.recbus`,
`Recorder.ArmBus[0..7]` (index 5 is Potato B1), `Recorder.Channel`,
`Recorder.mode.MultiTrack`, and tape playback `Recorder.B1/B2/B3`.
`Recorder.B1` is not a capture selector. Native REC can toggle pause, so Start
is guarded and issued once. API writes and external applications are not atomic;
a competing application can still change state between a read and write.

Before adopting the profile, back up config, sidecar, strip B1 sends and recorder
settings. Removing the profile relinquishes ownership but does not restore old
sends. Restore the backup explicitly with enforcement stopped for rollback.
