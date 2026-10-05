# Snoofer troubleshooting

- **Plugin unavailable:** Open Plugins for the exact startup/dependency error. Fix settings or restore the required integration, then use r to retry. Unrelated plugins continue running.
- **Audio writer already active:** Quit the other Snoofer instance before enabling live audio. Do not run two live writers.
- **Controls cannot open:** Use the repository-built bin/snoofer.exe. The controls process uses a detached process and allocates its own console; private pipes remain separate.
- **Saved choices/config changed:** Reopen the editor or reload after external edits. Stale revisions are rejected instead of overwriting newer choices.
- **VR overrides Normal:** Normal remains editable; those changes become effective when SteamVR stops. Shared mute/gain/recording controls remain active.
- **VR detection unknown:** The last known profile remains effective. Inspect process-access failures; Snoofer does not infer that a headset is worn.
- **Unavailable deck binding:** Its provider is disabled, absent, failed or has removed that control. The saved binding is retained. Restore that provider or assign a different control.
- **Shared-position conflict:** Clear the page-specific binding at that position before making it shared.
- **Shutdown timeout or native cleanup uncertainty:** Automatic relaunch is blocked. Resolve the old process before restarting so a second native owner is not created.
- **VR input number changed:** Clear the former owned input/monitor sends before deliberately replacing its audio ownership journal. The error identifies the journal path.
- **Audio engine stall:** API responsiveness/sample-rate readback alone is insufficient. The callback monitor reports processing progress, not audible output. Existing automatic-recovery eligibility, recorder guard, cooldown and uncertainty limits still apply.

Keep personal configuration and operational .state.json, .mutes.json, .recovery.json and .vr-ownership.json files with their configured state_path. Do not remove recovery journals to force a retry.

Build and rollback details for this refactor are recorded under openspec/changes/modular-snoofer/implementation.md. Physical-device acceptance is separate from automated tests and read-only preview checks.
