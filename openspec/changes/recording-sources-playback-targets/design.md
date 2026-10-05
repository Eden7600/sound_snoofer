## Design
### User model
Recording Sources contains Computer audio, Microphone and Mic stage (Pre-VST/Post-VST preference with the existing effective fallback). These exclusively determine the B1 capture mix.

Playback Targets contains Speakers/headphones (follows the selected physical playback output) and Discord/apps (B3). These are independent send toggles and may be selected together. The user clarified that clip playback must NOT pass through Element. Element exclusively processes live microphone audio; its inclusion in the earlier target question was a misunderstanding and is superseded by this clarification.

Capture transport is Start/Stop Recording, represented by one observed-state toggle on the deck. Soundboard consists of configured named clips, each with one Play action: resolve file, load it, then start immediately after preparation is verified. No separate Load button or implicit recording-source edits. A shared Stop Playback action is available in the TUI and may be explicitly mapped to a deck key. The removed tape/loop/To VST deck controls do not return as defaults.

### Layout and state
TUI groups: Recording Sources, Playback Targets, Recording actions, Soundboard. Clip rows display a sanitized name and actual playback/pending/error state. The deck reserves row 3 for the first nine configured clips in configuration order; custom profiles use clip:<stable-id> bindings. Media remains zero-based keys 27/28/29. Blank clip slots stay blank; no placeholder files or invented clips.

Use a distinct persisted PlaybackChoices object rather than continuing to add playback flags to RecordingChoices. Configured clips have a unique stable ID, a display name, and a file path resolved relative to the config directory. Validate IDs, duplicates, labels and paths; a missing clip file disables only that clip with an actionable reason, not the entire app. A path is passed directly to the native API, never interpolated into a command script. Do not serialize file contents or replay clip actions after restart.

### Transport contract
The native recorder is a single capture/playback transport. A clip press during active or paused recording is rejected without stopping recording. Start Recording during clip playback is similarly rejected; the user stops playback first. A clip press during existing soundboard playback replaces it: verify Stop, verify the newly loaded file, then Play exactly once. Pressing the same clip restarts it from the beginning. Failed validation leaves the current clip alone; uncertain stop/load/play outcomes abort and are never automatically retried. No enabled/available playback destination means no Play command.

File loading capability, supported formats, loaded-file identity readback and short-clip completion semantics must be verified against the installed Remote API before implementation. Success of a setter alone does not prove the new file was loaded. Test that load failure never plays a previously loaded file. If exact identity cannot be observed, document the limitation and establish a safe supported verification method before shipping.

### Routing and temporary ownership
B1 remains capture-only; tape never feeds B1 or B2. Playback sends are exactly Recorder.A<n> for the active physical output when Speakers is enabled, and Recorder.B3 when Discord/apps is enabled. Clear other owned tape sends to prevent stale destinations. Multiple selected targets hear the same original clip. Element/AUX remains exclusively in the live microphone path and receives no clip playback.

Playing, stopping or switching a clip does not suppress live mic sends, disconnect devices, select source Off, alter native mic mute, or change Element routing. A recorded processed microphone snippet already contains its processing; playback does not process it again. Live microphone and clip may naturally mix on the app bus if both are enabled. Mic mute silences the live mic path while clip sends remain independently controlled; bus mute still applies normally to all audio on that bus.

Recording preferences do not change when targets or clips change. Playback target preferences do not block recording merely because a target is selected; only actual active playback/unknown transport blocks capture. Launch/reconnect may reconcile safe routing preferences but never start capture or playback. Loop, if retained in the TUI, is a playback option, not a recording source, and is off for one-shot soundboard playback unless explicitly opted into per clip in a later design.

### Migration
Read legacy ToVST/Loop/TapeRoutingManaged without breaking existing config files. Preserve capture preferences. Legacy ToVST does not become an Element playback target: retire that routing behavior and explain the migration in release notes. Clear legacy Recorder.B2 safely only when the transport is observed stopped; if an existing rehearsal is playing, surface a stop-required migration state instead of altering active playback. Do not change live mic routes during this migration, submit playback, or unmute anything. Maintain backward-compatible storage reads until a later explicit format cleanup.

### Verification
Deterministic tests cover both target combinations and neither target, device changes, mic Off/muted, unchanged live Element routes, capture/playback conflicts, path validation, missing files, failed/uncertain load and play, rapid repeated clip presses, stale queued actions, bounded cancellation and preference migration. TUI keyboard/resize checks and a dry-run interactive smoke test are required. Physical acceptance separately verifies actual capture files, speaker/app destination combinations and absence of feedback or double processing.
