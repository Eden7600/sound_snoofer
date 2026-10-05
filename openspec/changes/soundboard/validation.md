# Validation — 2026-10-05

- Canonical scripts/check.ps1: all Go tests and vet passed; all 35 OpenSpec changes passed strict validation; callback companion self-test and soundboard native build passed.
- Race detector unavailable because CGO is disabled; toolchain unchanged.
- Opt-in TestNativeClipPlayback: real MP3 decoded on the exact DirectSound VAIO3 renderer at -100 dB. Replacement, completion, missing-file rejection, missing-renderer rejection and cleanup passed. No recorder transport used.
- Opt-in TestLiveSoundboardRouting: configured audio/VR/soundboard host reached available clip controls after observing VAIO3 sends. No clip or recording started by this check.
- Catalogue tests cover MP3 filtering, stable identity and file changes. Artwork tests cover case-insensitive JPEG/PNG matching, precedence, corruption/nonsquare fallback and removal refresh. Native-size rendered examples inspected in docs/design/soundboard.png.
- Automatic page tests cover overflow, manual/shared positions, unchanged Home and stale key events during catalogue updates. Artwork cache refresh and unchanged-image reuse tested.
- Personal configuration validated; Home compared exactly with pre-soundboard backup. Soundboard enabled for C:\Users\Eden7600\Music\Soundboard; Stop at key 35 on its page. Existing page dial provides navigation.
- bin/snoofer.exe rebuilt and restarted with its original no-argument launch. SHA256: 6EA132FD362A0EDD1004845AD0E6AC7C7780DC3AA4D19AA1935AE9A0CD13547A.
- Audible microphone/local output and physical Deck acceptance remain unperformed. Automated routing/native checks do not prove audible output.

## Artwork compatibility fix
fah.png is 85x82; sadge.png contains WebP data. Both now pass actual-file catalogue/thumbnail checks and their decoded previews were inspected. Rectangular images fit proportionally with transparent padding. Added golang.org/x/image for WebP decoding and proportional resampling; original image files remain unchanged. GIFs use the first frame on the logical canvas. Regression tests cover misleading extensions, proportional fitting and GIF positioning/transparency. Full canonical checks passed (tests, vet, 35 OpenSpec changes, native companions and build); restarted bin/snoofer.exe. Race detector remains unavailable with CGO disabled. Physical display confirmation remains user acceptance.

## Soundboard volume dial
Bound soundboard.volume to the first dial on the personal Soundboard page. Gain commands use the existing serialized audio worker and readback verification. Turning adjusts Strip[7].Gain within -60..+12 dB; pressing sends an absolute-zero reset, independent of previously displayed gain. Regression tests verify clamping, delayed readback, mute/other-strip preservation, and rejection of stale, preview or disabled requests. Canonical tests, vet, 35 strict OpenSpec checks, native builds and configuration validation passed. Restarted bin/snoofer.exe. Physical dial acceptance remains unperformed.
