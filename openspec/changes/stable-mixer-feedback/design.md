## Design
Gain writes remain serialized and identity-checked. After one setter call, poll parameter readback every 10 ms for up to 100 ms using the controller clock and context; accept a 0.01 dB tolerance. Do not repeat the relative write. Genuine setter/read failures and verification timeout remain errors. Queued turns execute only after verification, preserving relative increments.

Represent delayed mute readback with a typed pending sentinel, preserving mute ownership and the existing block on dependent route writes. Controller Step schedules another observation at 20 ms without emitting an error for this sentinel.

Track observation errors in the observed backend wrapper independently of controller events. Shared Connected depends on successful native observation, not route conflicts, mute pending or setter errors. Preserve diagnostics in Error and withhold a plan only when it cannot be built; successful observation still means connected. True read failures continue to mark audio unavailable. Recovery health uses observation validity rather than unrelated controller errors.

Keep observed gain text on the touch strip during scoped feedback; show ERR or WAIT on a separate status line so valid dB is not replaced. Cache includes that status. Unknown native readbacks remain unknown.
## Verification
Use delayed fake native readbacks and controlled clocks to cover eventual gain success, timeout, setter failure, cancellation, relative increments, pending mute safety, and observation failure/recovery. Compare deck presentation to ensure diagnostics retain dB. Full suite, vet, strict spec validation and replacement Windows build. Physical confirmation remains separate.
