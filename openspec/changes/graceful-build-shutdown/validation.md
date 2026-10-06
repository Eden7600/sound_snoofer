# Validation

- `scripts/check.ps1 -GUI` passed: Go tests, vet, all 41 strict OpenSpec checks, native companion checks, browser UI checks and native desktop checks. Race checks remain skipped because CGO is disabled.
- Detached GUI fixtures close through their input pipe on cleanup. Exit deadlines clear their timers; any necessary forced GUI fallback is awaited and fails validation. These fixtures never load audio plugins.
- A deliberately unclosed GUI fixture made stop.ps1 time out; its process remained alive and subsequently exited normally after EOF.
- An unrelated parent PID did not authorize closing its Snoofer child. The canonical tray fixture then exited with code 0 on the scoped close request; repeated stop succeeded harmlessly.
- Adapter test confirms monitoring stops before Logout and DLL release, exactly once. Existing audio-worker ownership/cancellation tests passed.
- Live personal host PID 30612 exited normally through stop.ps1 in 687 ms including helper setup. The app was then restarted with its original live configuration. No Voicemeeter/Element process was killed or restarted, and no forced host stop was used.

Normal exit verifies the host's cleanup path returned without error. It does not establish the state of registrations left by earlier hard terminations; no API connection count or recovery of historical stale registrations was claimed.
