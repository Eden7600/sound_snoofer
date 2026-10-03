# Design

Use a shared pure filter returning a fresh device slice in original order. Detect VB-Audio virtual hardware IDs (including the observed VBAudioVACWDM) and known virtual endpoint name families: Voicemeeter, VB-Audio Virtual, VB-Cable, Virtual Audio Cable and SteelSeries Sonar. Match case-insensitively. Hardware ID detection handles truncated/renamed cable endpoints; name detection covers virtual ASIO entries whose IDs are GUIDs. Do not treat ASIO itself as virtual or hide unknown hardware. The API provides no universal virtual-device flag; unfamiliar virtual drivers without recognized identifiers remain visible until classified.

Apply only to the TUI inventory list and a copy of the CLI devices snapshot before text/JSON serialization. Preserve assignments, full internal snapshots and routing inventory keys. Keep error/disconnected handling and text sanitization. If the filtered list is empty, show 'No physical devices found.' in the Devices list. Document that filtering is based on known virtual driver identities.

Verify actual observed VB-CABLE and Voicemeeter ASIO entries, case folding, truncated cable names, preserved physical ASIO/WDM devices, nonmutation and consistent CLI/TUI behavior. Run regression tests, vet, build and a preview-only terminal smoke.
