# Graceful build shutdown

## Why
Forced termination of the live host bypasses callback deregistration and Voicemeeter logout. GUI test failure cleanup also kills children without awaiting exit.

## What Changes
- Reuse the tray's existing WM_CLOSE lifecycle from a repository-scoped stop script.
- The canonical build invokes graceful stop, waits for exit, and aborts on failure/timeout without forced host termination.
- GUI smoke cleanup closes the private input pipe first and waits; forced fallback is allowed only for that test's audio-free GUI child and fails validation.
- Verify shutdown and native callback-stop/logout/release ordering. No new IPC protocol or audio routing behavior.
