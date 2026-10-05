## Implementation
- [x] Add bounded per-binding action feedback without cross-client/global-notice contamination.
- [x] Replace global Stream Deck overlays with independent binding status and complete presentation caching.
## Verification
- [x] Test error isolation, expiry/retry, unrelated pending operations, audio disconnect and unchanged JPEG tiles.
- [x] Run tests, vet, strict OpenSpec validation and Windows build.
- [ ] Confirm the corrected indicators on the physical Stream Deck after replacing the running build.

Verification: full Go tests and vet passed; strict OpenSpec validation passed; Windows replacement built at bin/sound-snoofer-deck-status.exe. Physical deck confirmation remains pending. Race detector unavailable with the current CGO-disabled toolchain.

