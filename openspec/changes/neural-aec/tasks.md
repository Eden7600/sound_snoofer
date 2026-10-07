## 1. Design
- [x] Specify engine selection, native worker ownership and compatibility limitations.
## 2. Dependency and native engine
- [x] Build pinned LocalVQE and deploy verified models/licenses.
- [x] Implement bounded asynchronous native insert with resampling, generations and failure pass-through.
- [x] Verify real inference, framing, reset, signal preservation and deadlines.
## 3. Plugin and GUI
- [x] Persist and validate engine selection; safely detach/switch engines.
- [x] Add GUI selector and model limitation/status, disable inapplicable Strength.
- [x] Test switching including failure and save rejection; inspect rendered GUI.
## 4. Release checks
- [x] Run focused/full Go tests, vet, spec validation and production build; record preexisting failures.
- [x] Commit validated implementation and restore normal host launch.
- [ ] Real-room speaker echo/double-talk listening comparison with both neural models (user hardware acceptance).

Validation: both models passed 18 total rate/block stream cases and exact direct-stream fixture parity. Native Go ABI model reload tests passed. GUI render/interaction and six GUI model tests passed; vet passed; all 57 OpenSpec items passed. Full Go tests fail only the pre-existing missing config.voice.json fixture (TestVoiceExample). Race detector skipped: CGO_ENABLED=0. Production build succeeded and the no-argument host launch was restored. Real-room listening remains unchecked.
