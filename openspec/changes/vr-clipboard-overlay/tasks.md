# Tasks

- [x] 1. Commit the design and requirements; validate OpenSpec strictly.
- [x] 2. Implement and test guarded Windows target capture and keyboard shortcuts with no clipboard reads.
- [x] 3. Implement and test the installed-runtime OpenVR adapter and explicit ABI/lifecycle handling.
- [x] 4. Implement the pure panel renderer/gesture model and isolated worker/plugin integration, disabled defaults and diagnostics. Inspect a synthetic rendered preview.
- [x] 5. Run reviewed isolated Go/vet/race checks and strict spec validation, document results and commit logical blocks. Preserve all unrelated changes and live state.
- [x] 6. Following the user’s explicit “Deploy it”: canonical build/deployment, exact host launch restoration, VR Clipboard enablement and SteamVR connection check. Preserve other plugin settings and restore the prior controls window.
- [ ] 7. Physical headset acceptance: input target, all seven keys, held/multiple controllers, blocked/elevated focus, runtime shutdown/reconnect and native cleanup. Accept user confirmations without redundant verification.

- [ ] 8. Correct button-position handling to use MouseMove state; add realistic fake regressions and run scoped tests/vet/race. Keep Windows-focus diagnosis separate until live evidence arrives.
- [ ] 9. Deploy the validated compatibility correction and obtain real button/target feedback without inferring success from process presence.
