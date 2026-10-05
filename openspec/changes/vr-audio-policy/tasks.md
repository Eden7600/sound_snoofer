## 1. Feasibility and design checks
- [ ] Validate Windows default-setting adapter and role readback on supported Windows; document unsupported behavior.
- [ ] Collect actual Index and Beyond 2E endpoint identities and validate regex examples without changing live routes.
## 2. Implementation
- [x] Add versioned profiles, preferences, validation and migration.
- [x] Add session-scoped runtime observation and classification before generic matching.
- [x] Integrate independent effective selection, stable identity checks and reserved-strip validation.
- [x] Implement bounded default-protection worker, polling, verification and contention status.
- [x] Add Controls/Graph presentation through existing state flow.
## 3. Automated verification
- [ ] Cover running/stopped/unknown, reconnect, ambiguity, equal priorities, Source Off, partial headset profiles and saved-choice restoration.
- [ ] Cover all six role/direction pairs, invalid targets, contention, cancellation, unsupported setter and dry-run.
- [ ] Run Go tests, vet, Windows build, strict OpenSpec validation and isolated TUI smoke.
## 4. Hardware acceptance
- [ ] Exercise Index and Beyond 2E start/exit/hotplug with independent preferences and Volt A1 retained.
- [ ] Verify Windows defaults inside/outside VR, protection disable, competing SteamVR changes and actual app audio.
