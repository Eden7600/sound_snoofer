# Design

## Decisions
Treat a voice intent with MicActive false as requiring no microphone input connections. Clear Patch.asio[0..3], clear managed input 1/2 assignments and the owned webcam on input 3. Keep existing ASIO hardware selection and A1 reservation independent of microphone intent: the user explicitly requires Volt to remain on A1. Playback topology therefore remains unchanged by Off. Preserve input 3 ownership rejection for unknown devices.

This supersedes the previous Off behavior that left microphone inputs assigned. Active microphone modes keep their current topology. Selecting a microphone again reconnects inputs through ordinary planning and verification. Off suppresses webcam-selection warnings; ASIO/output inventory errors remain subject to existing safety checks.

Keep transition gating before device/patch changes, readback verification, save-before-apply and preview semantics. Do not stop Element, disable Windows endpoints, or issue recorder transport commands. Volt remains open as the audio engine's output device; Off disconnects its input patches rather than releasing the interface. Device setter failure remains visible and reconciliation retries from fresh observations.

## Verification
Test Off/restore, absent and ambiguous webcam hardware, missing playback, unrelated assignments, repeat-plan convergence, stable A1/playback and failed device readback using synthetic backends. Run regression tests, vet and build. Real-device acceptance remains a separate unchecked task until performed.
