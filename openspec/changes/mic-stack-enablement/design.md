## Decisions
Reuse Intent.Enabled as the shared master switch; no new duplicate field. A false value never overwrites Source, and source edits never modify Enabled. The profile resolver may resolve unavailable targets but must preserve a disabled master across all activation/fallback paths.

Legacy Normal source Off normalizes to Enabled=false, Source=auto. Saved VR source Off also normalizes to auto and disables the shared stack, so an old intentional stop cannot silently reconnect. Internal unresolved source sentinels remain implementation details; public target controls do not offer Off. Existing legacy config priority sentinels are accepted for safety; they are not target selections.

Add audio.mic-stack enablement beside existing audio.mic-mute in a Mic stack group. Preserve audio.source and fixed-profile IDs so existing deck bindings keep working; rename their presentation to Mic stack target and publish the saved target rather than an effective Off/unavailable sentinel. Normal and VR targets remain in separate editable profile sections. The master is never subdued by VR. Do not change user-owned deck layouts.

Disabling uses existing MicActive teardown: clear managed hardware input assignments and ASIO input patches, microphone sends, monitoring and processing return. Keep Volt reserved at A1 for playback; do not terminate Element, alter Windows endpoints or issue recorder transport. Mute remains a native mute operation, not a disconnect. Save-before-apply, revisions and confirmation safeguards remain unchanged.

Validate persistence/legacy handling, target edits while disabled, Normal-to-VR transitions, full teardown and re-enable with shared mute preserved, and semantic controls without Off. Build separately if the current executable is running; do not stop it silently.


## Validation and deployment
Full Go tests and vet passed. The default GUI replacement build and personal-envelope offline check passed; all 32 OpenSpec changes passed strict validation. Regression coverage includes persistent disabled targets/mute, safe legacy Off handling, source edits that cannot enable, shared master across Normal/VR, full mic teardown with recording/playback retained, and semantic controls/error visibility. No physical switching test was performed. The existing Snoofer process remained running, so bin/snoofer-next.exe was prepared beside the existing companion/config rather than replacing the in-use bin/snoofer.exe. Personal state files were not changed.

## Requested personal Stream Deck layout
At the user's request, place audio.mic-stack immediately before audio.mic-mute on Home. Shift the existing first four bindings right by one into the empty fifth key, preserving all other positions and shared bindings. This is a personal configuration edit; default layouts remain unchanged.

Installed bin/snoofer.exe after shutdown; SHA-256 matched the validated candidate and the installed executable passed the offline personal-config check. Backed up the previous executable, companion and configuration under .local/rollback/pre-mic-stack. Verified the saved Home layout starts with Mic stack then Mic mute and retains 36 keys. No live hardware test or launch was performed.
