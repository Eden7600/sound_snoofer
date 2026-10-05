## Decisions
Use Voicemeeter GetLevel through the existing pinned audio worker, sampling every 100 ms when the audio plugin runs. Output A1/A2 show the maximum of their eight output channels (type 3); mic shows the maximum of the active hardware input pair after mute (type 2), following the same source as its gain knob. This is a sampled digital level meter, not calibrated analog VU ballistics or a guaranteed true-peak detector. The official SDK header defines offsets: output bus index times eight; physical input strip times two. No new callback or login.

Use immutable fresh level maps and timestamps in worker state; publish via the existing bounded state channel without accelerating routing or device inventory. No readings on disconnected/recovering state. Gain controls carry optional value-type telemetry; meter changes alone do not change command revisions. Consumer expires data after 500 ms, including a stalled producer. API failures and nonfinite/negative samples are unknown, distinct from measured silence; missing/disabled mic has no valid reading.

Render a 24-segment bar from -60 to 0 dBFS: green below -12, amber from -12, red from -3. Preserve label and numeric gain. Unavailable data shows LEVEL N/A, non-audio/navigation dials retain existing content. Retain current 150 ms surface cadence, reuse static key JPEGs, transmit only changed images. No smoothing, peak-hold, new settings or layout changes.

References: https://download.vb-audio.com/Download_CABLE/VoicemeeterRemoteAPI.pdf (Real Time Level Meter); https://github.com/vburel2018/Voicemeeter-SDK/blob/master/VoicemeeterRemote.h (GetLevel contract).

## Validation
Cover channel mappings, invalid/missing readings, disabled/rerouted mic, revision stability, stale expiry and meter rendering/cache. Run Go tests/vet, GUI build/config check, strict OpenSpec validation. Inspect a rendered strip locally. Hardware movement/readability remains a separate acceptance check.

Full Go tests and vet passed, plus the focused nonfinite-telemetry regression after final review. Strict OpenSpec validation, GUI build and offline personal configuration check passed. Visually inspected the rendered strip at native dimensions; removed the temporary preview and generator. Candidate: bin/snoofer-next.exe. Installation awaits Snoofer exit; physical hardware acceptance is still unperformed. Race checks remain unavailable with the installed CGO-disabled toolchain.

Snoofer exited during cleanup. Installed the candidate as bin/snoofer.exe, verified its SHA-256 and passed the installed offline config check. Previous executable/companion preserved in .local/rollback/pre-knob-meters. Removed the duplicate candidate and this task's build cache. No personal configuration or runtime journals changed.
