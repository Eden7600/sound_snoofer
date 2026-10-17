# VR Clipboard

VR Clipboard adds a compact controller-operated panel to the SteamVR dashboard.
It is separate from Snoofer's VR audio profile and starts disabled.

After a build is deployed, enable **VR Clipboard** on Plugins using the usual
restart confirmation. With SteamVR running, select **Snoofer Clipboard** in the
dashboard. Select the text or field in the desktop application first. The panel
shows its executable name; **Use Focused App** captures the latest eligible
desktop window again.

- **Copy**, **Cut**, **Paste**, **Select All** send Ctrl+C, Ctrl+X, Ctrl+V, Ctrl+A.
- **Tab**, **Backspace**, **Enter** send one key press each.
- Press and release inside the same button. Holding does not repeat.

The panel sends shortcuts and never reads or stores clipboard contents. It
restores the explicitly displayed target with ordinary Windows focus rules and
refuses changed, blocked, elevated or locked targets and conflicting held keys.
**Shortcut sent** means Windows accepted the input; an application may handle a
shortcut differently. Select the target on the desktop and try again if Windows
refuses focus. No shortcut is automatically retried. Windows does not offer an
atomic HWND-bound SendInput operation; simultaneous external focus changes and
reused window handles within the same process remain platform limits.

The plugin uses the installed SteamVR OpenVR loader and currently supports
Windows amd64 with IVROverlay_028. Its separate hidden worker cannot stop the
audio host. It connects only after SteamVR is observed running; OpenVR overlay
initialization can still start the runtime if SteamVR exits during that check.
After worker/runtime exit, it waits for SteamVR to disappear and return before
starting another worker. Missing/incompatible runtimes appear in Third-party
apps. Disable the plugin through the usual Plugins lifecycle to stop it.

## Current development status
Source implementation, isolated tests, scoped vet and race checks, strict
OpenSpec validation and synthetic panel inspection are complete. Following the
user’s explicit deployment authorization, the canonical build was installed,
VR Clipboard was enabled, and Snoofer restarted with the prior configuration.
SteamVR accepted the overlay connection. Other plugin settings were preserved
and the controls window was restored. Physical controller/shortcut acceptance
and runtime-exit testing remain pending.

A corrective build addresses target-capture rejection and coordinate-free
controller button reports. Real headset verification of these fixes remains
pending.
