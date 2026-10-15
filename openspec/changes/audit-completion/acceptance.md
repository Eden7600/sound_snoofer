# Bounded native media acceptance preparation

Use uniquely owned scratch native MediaPlayer processes with silent WAV input,
Volume 0, looping playback and automatic System Media Transport Controls. Give
each a unique title/AppUserModelID and retain its exact PID/process handle. The
healthy player publishes an increasing title/clock heartbeat. Do not reuse an
installed player's existing process or issue commands to a user's media session.

Build only the scratch helper with the existing MSVC setup; no production source,
canonical DLL or test hooks are changed. Use one fresh run directory for its WAV,
logs and ready/release/stop files. An independent watchdog requests normal stop
and deferral release at 85 seconds and reports any cleanup overrun at 90 seconds.
A hung native call can exceed this cooperative bound; do not force exit or claim
a guaranteed process lifetime. Reserve time for cleanup and report failed normal
cleanup before stronger action. Compilation and pure checks start no media.

First establish real attributable SMTC publication in an approved session, then
perform a bounded cooperative command stall using MediaPlayer.CommandManager's
NextReceived/PauseReceived deferral. Capture/log each exact command once, delay
execution at most five seconds or until release/stop, apply it once and complete
the deferral. Do not rely on blocking the UI pump: manual SMTC ButtonPressed has
no deferral and Windows dispatches it off the UI thread.

The OS is not documented to keep GSMTC TryPauseAsync pending for that deferral.
Use the canonical native media DLL against only the verified owned session to
measure the real HRESULT and elapsed time. A near-two-second 0x800705B4 timeout
establishes the precondition for an uncertain-outcome check; early success/false
does not. Report the actual result rather than changing the production timeout.
If this precondition is unavailable, inspect a bounded externally supervised
freeze of only the verified owned process as a separate refined step before use.

During a successful stall, inspect Snoofer's live media updates for the healthy
owned session. For uncertain action acceptance, send exactly one native action
from Snoofer, observe its correlated Unknown/uncertainty feedback and keep waiting
through release/recovery. Require the player log to show one command and no replay.
An isolated GUI fixture or native probe alone cannot establish that whole journey.

Cleanup: release all deferrals, signal each owned helper to stop normally, confirm
each exact process exits and its SMTC session disappears, then remove only its own
silent WAV. Keep logs. Do not terminate unrelated players or force-stop Snoofer.
If normal cleanup fails, report the failure before stronger cleanup. Any GUI input
or visible launch requires the user's freshly approved timed control session.

The scratch native probe uses a Go test overlay only, leaving the repository and
canonical DLL untouched. Its default is disabled. Match exactly one owned nonce/role
title and explicit AppUserModelID after root verifies the live helper PID, creation
time and executable against its readiness record. Reject absent/ambiguous matches.
Read-only snapshots show only the owned sessions. An explicit Next/Pause option
sends exactly one command to the captured instance and records elapsed time plus
ErrTimeout (native 0x800705B4), refusal or success without retry. Keep OS-thread
ownership and close the client; bound the scratch test process. Pure selector tests
cover wrong identity, missing/duplicate sessions and unsupported command choices.
