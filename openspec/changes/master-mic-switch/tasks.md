# Tasks

## 1. Master switch

- [x] 1.1 Implement all managed mic send disconnection and zero-only recorder-conflict exception; verify Off/On, absent profile, unknown/conflicting recorder, hotplug/fallback and unchanged computer sends in tests.
- [x] 1.2 Label and document Microphone On/Off; verify Space sends the existing persisted enabled action and source preferences survive.

## 2. Integration

- [x] 2.1 Run Go tests, vet, Windows build and strict OpenSpec validation; record automated results separately from live audio acceptance.
- [ ] 2.2 Verify live mic disconnection and restoration by listening and inspecting mixer sends while computer recording continues; leave pending until performed.
