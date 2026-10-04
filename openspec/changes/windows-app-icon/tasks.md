## Implementation
- [x] Embed mascot resources and make regeneration reproducible.
- [x] Set attached controls window icons using bounded native calls.
## Verification
- [x] Run tests, vet, strict spec validation and Windows build.
- [x] Verify native executable icon extraction and source-resource agreement.
- [ ] Visually confirm the taskbar mascot in the user's window host.

Evidence: full Go suite, vet and strict OpenSpec validation passed. The resource test compares all seven embedded PNG frames to the tray ICO and successfully loads native 16/32/256px icons. Windows ExtractAssociatedIcon succeeded against bin/sound-snoofer-next.exe. The running application was preserved; the new controls window icon needs confirmation after restarting into the replacement build.
