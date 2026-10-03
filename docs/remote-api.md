# Remote API bindings

Bindings were checked against the [official SDK header](https://github.com/vburel2018/Voicemeeter-SDK/blob/main/VoicemeeterRemote.h) on 2026-10-03. The [Remote API manual](https://download.vb-audio.com/Download_CABLE/VoicemeeterRemoteAPI.pdf) documents the connection lifecycle, device enumeration, assignments, and patch parameters. Only the installation's DLL is loaded. Numeric reads use GetParameterFloat; allowlisted integer patch/send assignments use SetParameters.

- Windows LONG is int32 even in a 64-bit process. Status conversion preserves negative codes.
- The W variants use UTF-16 buffers; parameter names still use NUL-terminated ASCII. Device description buffers hold 256 UTF-16 code units; parameter readback holds 512.
- Login accepts 0 (connected) and 1 (engine not launched). Only one login and logout occur per process lifetime. Engine restart recovery uses refreshed state rather than repeated logins.
- The CLI pins its OS thread for the lifetime of the session, and the adapter serializes its calls. This honors the refresh API's single-thread requirement.
- Return codes, not GetLastError, determine API results. A negative refresh result or failed inventory/read discards the whole snapshot.
- Device setters use individual string parameters. Public device names are literal values, never interpolated into a parameter script. Numeric script targets are allowlisted and values range-checked; arbitrary scripts are not exposed.
- Fixed slots support WDM; studio mode supports ASIO on A1 only, gated by companion WDM presence. ASIO enumeration alone is not hardware availability evidence.

Regression tests exercise signed status handling, missing DLL errors, disconnected sessions, partial snapshots, fixed target mapping, and lifecycle cleanup. Live discovery works on the installed Banana instance. Real WDM assignment/readback, device-churn freshness, Potato, and listening tests remain pending.

## Opt-in voice profile

Potato voice routing reads strip A1..A5 and B1..B3 values and writes only boolean sends within edition limits. The profile owns Strip[0..7].B2/B3 and Strip[0,1,2,6].A1..A5. B1 remains read-only to this profile. Existing Patch.asio[0..3] is limited to channel values 0..2. No gain, mute, insert, or arbitrary script setters are added.

Semantic mappings in Potato: desk=Strip[0], lav=Strip[1], webcam=Strip[2], primary VAIO=Strip[5], AUX return=Strip[6]. B2 feeds AUX Virtual ASIO input channels 1/2; its output returns to AUX. B3 is the capture bus for Discord. Driver capture names vary by installation.

The final operation matrix is separate from execution transitions. A transition first gates owned voice sends, then configures devices/patches, then enables the desired sends. Each phase updates expected readback state; any other owned-cell mutation aborts the pass. No audio stream health is implied by a successful readback.
