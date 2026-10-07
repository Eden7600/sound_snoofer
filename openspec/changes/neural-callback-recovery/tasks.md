# Tasks
- [x] Trace callback dispatch, native failure checks and current live process/configuration.
- [x] Specify transient recovery, sustained-failure limits and test cases.
- [x] Add a regression that fails with the current permanent single-gap latch.
- [x] Implement generation-based bounded recovery in the shared neural bridge.
- [x] Pass both real-model probes, native Go ABI tests, focused/full Go tests, vet and OpenSpec validation; record existing limitations.
- [ ] Build, commit and restore the selected LocalVQE host.
- [ ] Long-session hardware listening confirms no further permanent stop from isolated gaps.

Validation: the new regression failed at the first injected midstream omission with the old DLL. The rebuilt bridge passed both real models across all nine rate/block combinations each, isolated omission recovery, extra-output invalidation, repeated-gap timeout, absent-reference timeout, persistent failure latching, and compute deadline checks. Fresh and recovered fixture parity both measured max error 0.00000000 for each model. Native Go ABI and focused audio tests passed; vet and all 58 OpenSpec items passed. Full Go tests retain only pre-existing TestVoiceExample failure for the deleted config.voice.json; race skipped because CGO_ENABLED=0. No model/latency/routing preferences changed. The precise upstream cause of the live callback gap is still unobserved.
