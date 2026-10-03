# Tasks

## 1. Go project and development tooling

- [x] 1.1 Create the Go module, CLI/core/Windows-adapter structure, build tags, and ignore rules; verify a windows/amd64 build succeeds.
- [x] 1.2 Pin OpenSpec development tooling and document Go/OpenSpec commands; verify the documented commands work from a fresh terminal and strict OpenSpec validation passes.

## 2. Remote API discovery

- [x] 2.1 Retrieve the official SDK header and implement absolute-path DLL discovery, checked exports, fixed-width ABI types, Unicode strings, login/refresh/logout, and edition detection; verify adapter tests cover missing DLL, signed errors, and disconnected state.
- [x] 2.2 Implement full input/output inventory and current managed-assignment reads; verify fake-adapter tests reject partial snapshots and preserve driver/direction distinctions.
- [x] 2.3 Add devices text/JSON output and setup documentation; verify CLI tests produce useful connection errors and inventory commands issue no setting writes.
- [ ] 2.4 Run a real-device discovery spike with Voicemeeter running and Volt 2/AirPods unplug/replug; record actual inventory freshness. If WDM enumeration is stale, add and test a Core Audio availability provider before live routing is considered complete.

## 3. Configuration and routing planner

- [x] 3.1 Implement strict versioned JSON config, Go regex compilation, direction/driver scoping, and edition-aware target validation; verify tests for malformed patterns, unknown fields, duplicate targets, timing bounds, and Potato-only slots on Banana.
- [x] 3.2 Implement ordered independent input/output selection with unique regex matches and explicit unresolved reasons; verify tests for Volt/webcam and AirPods/speaker precedence, fallback, reconnection, reordered inventory, missing candidates, broad-pattern ambiguity, and case-insensitive matching.
- [x] 3.3 Add plan output and an illustrative placeholder config; verify no-write CLI tests and document Go regex syntax, candidate ordering, and unresolved-selection behavior.

## 4. Apply and watch

- [x] 4.1 Implement serialized minimal writes and fresh readback with bounded timeout; verify fake-adapter tests for unchanged assignments, pending success, timeout, partial failure, and unmanaged-setting preservation.
- [x] 4.2 Implement watch with injectable time, polling, debounce, reconnection, and bounded backoff; verify deterministic tests for device flapping, invalid snapshots, disconnected engine, edition changes, and retry recovery.
- [x] 4.3 Implement apply-once and watch CLI modes, explicit live opt-in, cancellation, writer ownership, and state-change logs; verify dry-run makes zero writes, a second writer is rejected, cancellation releases resources, and failures return appropriate exit codes.
- [x] 4.4 Document live-watch ownership of configured slots, A1 interruption, readback limitations, and manual recovery; verify documented commands against a test configuration.

## 5. Hardware acceptance and integration

- [x] 5.1 Confirm the user's managed input strip/output bus and actual device patterns, record current assignments, and review a dry-run; verify the personal config selects intended endpoints without writes.
- [ ] 5.2 On Banana, test Volt 2/webcam and AirPods/SteelSeries connect/disconnect cycles plus all-candidates-absent behavior; verify actual microphone/playback audio and record any engine restart requirement without hiding it behind automatic restarts.
- [ ] 5.3 Verify Voicemeeter restart recovery and that Element-related patch, insert, gain, mute, and strip-to-bus settings remain unchanged; record observed results separately from automated tests.
- [ ] 5.4 Verify Potato behavior on an available installation; if unavailable, record the hardware test as pending rather than claiming it passed. Verify automated capability tests in either case.
- [x] 5.5 Run go test ./..., go vet ./..., the Windows build, and strict OpenSpec validation; archive the change only after required implementation and acceptance tasks are complete.

## 6. User-confirmed ASIO and logical routing rules

- [x] 6.1 Implement studio configuration, companion-presence gating, paired channel patches, and lowest-free-output allocation; verify studio planner tests.
- [x] 6.2 Implement allowlisted numeric API writes and ordered verified topology application; verify failure handling, bidirectional transitions, and idempotence tests.
- [x] 6.3 Migrate playback sends and enforce semantic virtual-input rules on every reconciliation; verify manual drift repair and Banana/Potato source-index tests.
- [x] 6.4 Replace the personal WDM-only Volt configuration and update usage/specification documents; verify strict OpenSpec validation and a read-only real-device preview.
- [ ] 6.5 Perform hardware acceptance of separate Volt channels, ASIO departure/reconnect, playback migration, and primary VAIO audibility; record observed results without equating readback with listening.
