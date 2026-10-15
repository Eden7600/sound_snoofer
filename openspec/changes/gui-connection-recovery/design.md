# Design

Keep the existing polling loop and finally scheduling. Track consecutive transport failures locally in that loop, capped at six. After a failure, delay by min(5000, 100 * 2^failures) milliseconds (200, 400, 800, 1600, 3200, then 5000). A successful Desktop.State response resets the counter and reconnecting flag before rendering, even when it returns null. Normal Audio, Soundboard, App Audio and Media polling remains 100 ms; other screens remain 200 ms.

Separate the State call's error handling from receive/render errors: only a missing Desktop binding or rejected State request enters recovery. A rendering error still uses existing error feedback, and finally keeps polling alive. Do not convert failed polls into showError calls or clear existing action feedback on recovery.

Store only reconnecting: boolean in shared state. A persistent, initially hidden #connection element immediately below the header uses role=status and the existing attention palette. Its concise text is Reconnecting to Snoofer… Controls are unavailable. The normal update path controls visibility, including when navigation changes screens. Keep the last received screen available for reading and navigation; do not rebuild or clear it solely because connection status changes.

At the shared send boundary, refuse input while reconnecting with Host disconnected; action not sent. Preserve controls' existing focus and input behavior; do not make the whole screen inert. Do not save refused actions, retry uncertain sends, or infer success from a successful reconnect. Existing global pending behavior is unchanged here and will be replaced in gui-action-outcomes.

Use isolated Playwright tests with the existing fake host and controlled browser time. Fail/recover State without changing production providers. Verify growing bounded retry intervals, global banner placement/visibility, retained last state and unresolved errors, explicit refusal without Send, recovery to normal polling, and no replay of a failed one-shot. A successful State followed by a throwing render must not mark the transport disconnected or stop scheduling. Update the UI contract's former host-loss error-notice rule intentionally.

The user has lifted the temporary running-app restriction. Root coordinates canonical packaging and native acceptance after all action-result integration is stable, then restores the prior launch configuration. Source-served fault tests establish reconnect banner/backoff behavior; native EOF checks establish shutdown, not a rejected State binding.
