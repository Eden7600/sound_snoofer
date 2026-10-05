## 1. Feasibility
- [ ] Reproduce the confirmed whole-engine audio stall, for which the user reports engine restart fixes audio; determine whether Remote API calls remain responsive and capture clock-device evidence.
- [ ] Compare connected, silent, muted, failed and recovering readbacks for WDM and ASIO; document which signatures are reliable and which failures remain undetectable.
- [ ] Validate restart effects on Element, recorder transport and owned mute; keep unproven signatures in observation-only mode.
## 2. Implementation
- [ ] Add health evidence/state and transition grace without slowing numeric setting edits.
- [ ] Add narrow standalone engine restart action, policy toggle, recorder guards and persistent attempt budget.
- [ ] Add verification/reconciliation with latest intent and ambiguous-result handling.
- [x] Add Controls/tray actions and optional Stream Deck binding with pending/error status.
## 3. Automated verification
- [ ] Cover silence/mute/gates, missing devices, SteamVR exit, ASIO-patched strips, observation failures and target changes.
- [ ] Cover cooldown, restart budgets, app restart, clock changes, duplicate actions, unknown outcomes and recording deferral.
- [ ] Cover concurrent edits, dry-run, preserved assignments/sends/mute, no transport replay and a simulated stuck actor without a second writer.
- [ ] Run tests, vet, Windows build, strict validation and isolated UI smoke.
## 4. Hardware acceptance
- [ ] Verify actual incident detection and engine recovery on the affected hardware, separately from mock success.
- [ ] Check audible output, Element return, mic mute, Volt A1 and recorder behavior after recovery.
- [ ] Check prolonged silence, ordinary hotplug, VR transitions and suspend/resume cause no spurious restarts.
