## 1. Design
- [x] Specify engine selection, native worker ownership and compatibility limitations.
## 2. Dependency and native engine
- [ ] Build pinned LocalVQE and deploy verified models/licenses.
- [ ] Implement bounded asynchronous native insert with resampling, generations and failure pass-through.
- [ ] Verify real inference, framing, reset, signal preservation and deadlines.
## 3. Plugin and GUI
- [ ] Persist and validate engine selection; safely detach/switch engines.
- [ ] Add GUI selector and model limitation/status, disable inapplicable Strength.
- [ ] Test switching including failure and save rejection; inspect rendered GUI.
## 4. Release checks
- [ ] Run focused/full Go tests, vet, spec validation and production build; record preexisting failures.
- [ ] Commit validated implementation and restore normal host launch.
- [ ] Real-room speaker echo/double-talk listening comparison with both neural models (user hardware acceptance).
