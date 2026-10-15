# Design

Keep output-state words in Control.Value (In use, No device, Disconnected, No output). Remove the descriptive device name from Control.Status and include it in the already-existing output header ViewData object as Device, alongside ID and Bus. The device name is information, not an error. Hidden slots remain hidden, available status keys stay display-only, and source switches keep their current IDs and actions.

Routing uses ViewData.Device for its device caption, current option, duplicate-choice filtering and initial selection. Its structural signature already includes slot ViewData, so device changes still rebuild the applicable controls. Keep the stable slot ID and configuration-hash stale guard. Update the transitional manual fixture and regenerate real audio fixtures; the deck fixture drift test verifies any effect on provider-derived choices.

Do not weaken bindingTile/controlTile's generic status precedence or add an audio-ID special case to the renderer. Healthy destination state then reaches the native renderer through its existing path. Test real generated audio output through bindingTile, including the known healthy device name, and explicit No device/Disconnected/No output/hidden variants. Keep tests for genuine Pending, Unavailable and error statuses. Use the existing offline 112px renderer preview mechanism to inspect state colors and layout without connecting to hardware.

Tests must fail with the old Status publication and old Routing consumers. Verify names and device selection through provider-backed browser fixtures at both supported sizes. The UI contract will state that the device caption is descriptive and destination state stays a state word on deck keys.

No canonical build, deployed files, running-host interactions or native app launches while the user's restriction remains active. Offline renderer tests are permitted; leave packaged/hardware acceptance unchecked.
