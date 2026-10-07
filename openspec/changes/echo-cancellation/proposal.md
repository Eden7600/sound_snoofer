`# Echo cancellation
`## Why
`With speakers instead of headphones, the microphone picks up what the speakers play, so callers and recordings hear themselves and the game or music. [VoiceMeeter-AEC](https://github.com/BeeeFX/VoiceMeeter-AEC) solves this as a separate app with WebRTC AEC3 through Voicemeeter's Insert virtual ASIO driver and PATCH INSERT. Snoofer already owns Voicemeeter's audio callback for its health monitor, so it can cancel echo itself, with no extra app, driver or patching.
`
`## What Changes
`- **`aec` plugin:** WebRTC AEC3 cancels speaker echo from the managed microphone. The reference is the bus that plays to the speakers, and the result is written back on the mic strip's input insert.
`- **Engine:** the freedesktop \`webrtc-audio-processing\` v2.1 sources (AEC3, BSD-3) and an abseil subset (Apache-2.0) are vendored under \`third_party/\`. \`scripts/build-aec.ps1\` builds them with an explicit \`cl.exe\` source list into \`bin/snoofer-aec.dll\`, which exposes a small C ABI.
`- **Shared callback:**
`  - **Why:** Voicemeeter allows one audio-callback registration per process.
`  - **Registrant:** the monitor DLL stays the only registrant. It gains an optional insert hook: function pointers and a context supplied by the AEC DLL.
`  - **Input insert:** registered only while the hook is set.
`  - **Owner:** the Voicemeeter adapter (\`internal/voicemeeter\`) owns registration, as today.
`- **Controls:** Echo cancellation (Auto, On, Off) and Strength (Strong, Balanced, Gentle).
`  - **Status:** shows how much echo is removed (ERLE), the estimated delay and whether it is active.
`  - **Placement:** a deck key and an Audio screen card.
`
`## Impact
`- **Code:**
`  - new \`third_party/webrtc-audio-processing\` and \`third_party/abseil-cpp\` (vendored, with licences);
`  - \`internal/aec\` (Go wrapper and C++ shim);
`  - the monitor C source and \`internal/voicemeeter\` callback management;
`  - \`plugins/audio\`, which supplies channel targets and installs the hook;
`  - the new \`plugins/aec\`;
`  - the build script and the GUI.
`- **Audio path:** while active, the mic strip passes through AEC3, which adds about 10 ms of framing latency. Routing, gains, mutes, the recorder and the invariants in \`CLAUDE.md\` are unchanged.
`- **Licensing:** BSD-3 and Apache-2.0 notices ship with the vendored sources. No Voicemeeter or ASIO SDK code is redistributed.
