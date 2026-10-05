## Current behavior and supersession

The element-process-fallback and sound-snoofer-default-launch changes supersede historical Direct/Post normalization, no-dry-bypass, and preview-default statements below. Current behavior preserves Post preference, uses effective Pre in Direct, automatically falls back to Direct when Element is unavailable, and defaults to live. Historical checkboxes are evidence only, not instructions to restore superseded behavior. Physical acceptance remains pending. The harden-application-consistency change adds reset/discard confirmation without replaying transport.

# Design

Add a shared intent method normalizing only the recognized Direct/Post combination to Pre. Invoke from recording normalization, editing and SaveIntent on a cloned value before validation/serialization. LoadEffective normalizes decoded legacy intent before validation; effective intent normalization also protects direct planner callers. Invalid spelling still produces validation errors. Save-before-apply, revision checks and clone ownership remain intact. Do not rewrite sidecars merely by loading; subsequent explicit saves persist the normalized value.

In Direct mode, activating Recording Mic Stage does not enqueue a Post choice and shows a concise 'Pre only in Direct' explanation. Element retains Pre/Post cycling. Mode edits save mode and resulting stage in one intent transaction. Do not auto-restore Post on return to Element. Monitor behavior is outside this change.

This intentionally replaces the prior silent inactive recording behavior: when mic recording is enabled, switching to Direct now records the dry mic through Pre. Recorder Start/Stop and computer capture are unchanged. Verify legacy load, save and failed-save behavior, invalid values, direct planner callers, editing and UI constraints.

For pending live plans, choose the configured debounce if any changed operation assigns/clears a device (including legacy routing decisions), otherwise 100 ms for sends and patches. Return the smaller of normal poll and remaining debounce time while waiting. Preserve normal idle polling, verification, inventory stability and error backoff. A new desired plan resets its deadline. Mixed device/routing plans retain the device delay. The 100 ms value is a scheduling delay, not a guarantee of completed native readback within 100 ms. Test with fake clocks.
