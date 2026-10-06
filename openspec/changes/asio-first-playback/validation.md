# Validation

- `scripts/check.ps1 -GUI` passed: Go tests, vet, browser model and interaction checks, all 40 strict OpenSpec checks, native companion checks, canonical build, WebView2 accessibility/actions/close/reopen/host-EOF checks.
- Race detector skipped because the configured CGO toolchain is disabled; no toolchain changes made.
- New routing checks cover two connected interfaces with conflicting downstream preferences, fallback after the winning interface disconnects, unavailable microphone channels and ambiguous presence. Existing Normal/VR, ASIO playback migration and stack-off checks pass.
- Mixer checks cover authoritative mute/unmute, external drift, journal reload, old-bus release, unchanged gain and rejection of stale gain actions. Meter observation now includes every physical output bus.
- Rendered browser fixtures reviewed at wide and narrow sizes; updated Audio/deck baselines preserve branding, semantic colors and mic dial position.
- Detached redirected CLI regression passed. Installed executable's configuration check and read-only plan/devices commands passed without a console or dialog.

## Personal installation
Backups and before/after native snapshots are in ignored `.local/rollback/asio-first-playback`. Converted the audio configuration to ordered ASIO entries, appended Volt at the existing playback fallback priority, replaced Home dial 0 with Playback and cleared dial 1. Other bindings and preferences remain intact.

At deployment, Volt was physically absent and Arena occupied A1. Preserved the old requested mute for that active destination (unmuted), rather than transferring the unrelated saved mute from vacant A2. After host restart, readback confirmed A1 mute=0 and obsolete A2 mute=0. All observed Strip/Bus gains were identical before and after restart (playback -21 dB; former second bus -8 dB). The new runtime owns playback mute from Snoofer's persisted preference.

The single canonical `bin/snoofer.exe` was installed and restarted with the existing live configuration. These readbacks do not establish audible correctness, physical dial behavior or multi-interface/VR hardware transitions; those acceptance items remain unchecked.
