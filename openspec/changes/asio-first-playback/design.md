# Design

Resolve ASIO globally before Normal/VR source selection. Each interface has driver and physical-presence regexes and explicit desk/lav input channel mappings (zero means unavailable). These are the two physical ASIO microphone strips supported today; no arbitrary patch editor is added. Empty ASIO list disables ASIO. Skip absent candidates; ambiguous matches fail closed with diagnostics. Installed drivers alone are not presence.

Playback priorities accept WDM and ASIO candidates. An ASIO candidate is eligible only when it matches the selected interface. Manual selections have the same eligibility rule. Missing explicit microphone selections fall through the profile priority list. Selecting playback never changes the interface winner. Retain unmanaged-slot protection and transition gating. Mic stack off clears patches but retains the interface clock and playback.

Playback gain resolves the planned bus and binds actions to its device identity. No gain writes occur except explicit knob actions. Replace A1/A2 controls with gain-playback, preserving mic and soundboard controls. Personal layout uses the first output dial for Playback and clears the redundant second dial without shifting other controls.

Playback mute is persisted desired state, imposed as either 0 or 1 on the current output and read back before routing. Acquire ownership with a durable baseline, then restore obsolete outputs after routes settle. Keep microphone baseline behavior unchanged. UI requested state never derives from native mute; readback only marks pending/verified.

This supersedes Volt-only fallback and fixed A1/A2 surface requirements in volt-playback-fallback, mixer-surface-controls and streamdeck-knob-meters. Recovery remains limited to the already validated Volt callback setup.

Follow-up: move the Mic dial from index 2 to index 1 beside Playback. Clear index 2; preserve all other personal bindings.
