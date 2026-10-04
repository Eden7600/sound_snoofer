# Proposal

## Why

B1 is now the dedicated recording mix. Sound Snoofer must let the user choose microphone pre/post Element and computer audio independently, then start and stop Voicemeeter's built-in recorder from the TUI.

## What Changes

- Add an opt-in Potato recording profile that owns all strip-to-B1 sends and excludes unselected sources.
- Add Record microphone On/Off, Mic tap Pre/Post, Record computer audio On/Off, and explicit Start/Stop controls and observed recorder status.
- Post requires enabled Element voice mode; it becomes inactive in Direct mode instead of recording dry audio. Recording does not create a separate processing chain.
- Record the selected B1 stereo mix through Voicemeeter's bus-recording mode, preserving its recording directory and file-format choices.
- Persist source choices, never transport commands or a desire to auto-resume recording. Support existing saved voice choices without reset.
- Verify routing and recorder readiness before Start. Never retry an uncertain Record command automatically.

## Capabilities

### New Capabilities

- `recording-control`: B1 source selection, built-in recorder preparation, explicit TUI transport and recording status.

### Modified Capabilities

None in the canonical inventory, which is still empty. This builds on the unarchived selectable-voice-routing change. Its statement that B1 is untouched remains true without the recording profile; this new opt-in capability explicitly overrides that ownership boundary. Existing B2/B3 voice routing and monitoring behavior remain the baseline.

## Impact

Extend internal/config intent and state compatibility, internal/routing's desired matrix and phased transitions, Remote API recorder observation/allowlist, the controller's explicit command handling, and TUI rule rows. Update configuration examples, setup/rollback documentation and tests. No audio encoding, file browser, background capture, plugin hosting or external recording application is introduced.

Read-only inspection on 2026-10-03 found the installed recorder stopped, bus mode on, B1 armed, stereo channels and multitrack off. Recorder.B1 was on, which is a tape playback send rather than the capture-source selector. These observations are not evidence of a successful recorded file.
