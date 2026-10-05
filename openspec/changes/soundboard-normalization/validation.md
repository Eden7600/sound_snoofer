# Validation

- User explicitly chose peak matching to preserve intentional perceived-loudness differences. No loudness leveling or compression is implemented.
- Unit checks pass for cache reuse, source invalidation, cancelled/failed preparation cleanup, pruning, single-job replacement and Stop preventing late playback.
- Native Media Foundation decoding passed synthetic PCM signal and silence checks. Real fah.mp3 measured -0.9999 dBFS after normalization; original-file SHA256 stayed unchanged. Normalized WAV opened and ran through the exact VAIO3 renderer with native output muted, then closed successfully.
- Canonical scripts/check.ps1 passed all Go tests, vet, 37 strict OpenSpec checks, callback self-test, native companions and the single bin/snoofer.exe build.
- Snoofer restarted with its prior no-argument configuration. Physical/audible acceptance remains unchecked; race detection remains unavailable with CGO disabled.

Native decoder reference: https://learn.microsoft.com/en-us/windows/win32/medfound/tutorial--decoding-audio
