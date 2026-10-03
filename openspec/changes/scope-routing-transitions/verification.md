# Verification

- Passed `go test ./... -timeout 30s`, `go vet ./...`, strict OpenSpec validation and Windows build to `bin/voice-snooter.exe`.
- Controller traces assert exact numeric writes for monitor Off/Post, recording mic/computer toggles, recording tap, Direct mode, Lav source, playback toggle and mic Off. Each intermediate state is checked for duplicate voice/feedback; converged replanning issues no writes.
- Planner traces cover active/inactive ASIO input patch changes, active/inactive webcam replacement, playback replacement and A1 restoration. Only dependent sends are gated before hardware writes; computer recording remains stable.
- Existing failure, readback, inventory drift, cancellation and recovery tests pass.
- Live listening acceptance remains unchecked; no personal mixer settings were changed during tests.
