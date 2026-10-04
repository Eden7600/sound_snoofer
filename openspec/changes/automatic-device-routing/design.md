# Design

## Context

Sound Snoofer now has a working Go CLI and initial fixed-slot WDM routing. The user clarified that its primary purpose is automatic rule enforcement: Volt uses ASIO A1, channels 1 and 2 feed separate stereo strips, playback uses the lowest free hardware output, and primary VAIO always feeds playback. This supersedes the initial WDM-only Volt assumption. The lav currently occupies Volt input 2.

## Goals / Non-Goals

Goals: separate logical source and device roles from physical strip/bus indexes; enforce source rules even without device changes; retain deterministic preview and verified writes.

Non-goals: Element processing profiles, automatic application/engine launch, gains/mutes, arbitrary parameter scripts, ASIO insert processing, and a GUI.

## Decisions

- Retain version-1 fixed routes for compatibility. Add mutually exclusive studio configuration with regex driver/presence selection, ordered playback/fallback candidates, playback-send migration, and semantic playback_sources.
- ASIO driver enumeration is insufficient presence evidence. Require a unique WDM input companion; do not open it for capture. Ambiguity or unmanaged A1 ownership blocks a transition.
- Map Patch.asio[0..3] to [1,1,2,2]. These are two stereo strips receiving separate centered mono signals. When Volt disappears, reset those cells to zero and use the webcam on input 1; input 2 remains available without stale ASIO capture.
- Allocate playback to the lowest empty or owned output, excluding A1 while ASIO is active. Names matching configured playback regexes define output ownership across restarts. Preserve unrelated devices. No destination means no topology writes.
- Build explicit ordered operations: disable obsolete patches when leaving ASIO; prepare input selection; assign interface/playback outputs; install active input patches; migrate sends; enforce semantic source rules; clear former playback slots last.
- Move the union of enabled sends from owned playback outputs, then apply explicit source rules. virtual:1 maps to strip 3 on Banana and strip 5 on Potato. A manually disabled primary playback send is restored even if the device remains unchanged. Other virtual sources can be named; virtual:3 requires Potato.
- Revalidate the full plan before a transaction, compare order-independent inventory before every operation, detect unexpected target mutation, and verify each readback. Topology transactions are non-atomic; report partial progress and replan after failure. Fixed-route reconciliation remains unchanged.
- Numeric writes are limited to the four input patch cells and valid hardware-bus strip buttons. Windows uses GetParameterFloat for readback and a generated single integer assignment through SetParameters for writes; allowlisted names and bounded integer values prevent arbitrary script execution. This avoids floating-point syscall argument ambiguity.
- Keep serialized API access, one login/logout, an OS-thread-pinned CLI, one active writer per session, dry-run defaults, debounce, and bounded retry delays.

## Risks / Trade-offs

- Physical WDM presence freshness still requires unplug/replug acceptance; no engine restart workaround is added.
- ASIO/output changes can interrupt audio; each step is verified but there is no atomic rollback.
- Playback regexes identify outputs the application owns. Broad patterns can unintentionally claim a manual assignment; ambiguous candidate selection is rejected.
- On interrupted migration with multiple playback outputs, unioning enabled sends favors retaining audio. Exact state across a process crash has no persisted journal.
- Device-name readback does not prove audible audio or fully identify the driver. Live listening and channel isolation remain pending.

## Validation

Tests cover ASIO-presence gating, paired channel patching, output reservation/exhaustion, Banana/Potato semantic indexes, primary-send drift repair, migration in both directions, idempotence, numeric write allowlisting, and partial failure. Read-only discovery and dry-run execute against the installed Banana. Physical listening, unplug/replug, and Potato acceptance remain manual.

## References

- https://download.vb-audio.com/Download_CABLE/VoicemeeterRemoteAPI.pdf
- https://github.com/vburel2018/Voicemeeter-SDK/blob/main/VoicemeeterRemote.h
