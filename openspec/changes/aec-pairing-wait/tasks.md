# Tasks
- [x] Capture input/output formats and identify the repeated resynchronization cause.
- [x] Specify and implement the callback correction with a reproduction check.
- [x] Run native and Go checks, canonical build, and live host verification.

Validation: reproduced deployed-monitor batching (448 gaps in 10 seconds, no format mismatch) and showed that timeBeginPeriod alone was insufficient. Honoring the process timer request eliminated gaps. Rebuilt production monitor passed two 10-second neural captures with Active, 625 frames, zero gaps and underruns; final probe also verifies duplicate start refusal, repeated stop, and exact prior process policy restoration. Callback ABI/pass-through checks, full Go suite, vet, strict change validation and canonical build passed. Normal host restored with LocalVQE voice cleanup; GUI timing reports zero gaps/underruns, worker 4.2 ms and queue 4.0 ms peaks. Long-duration listening acceptance remains unverified. Race detector unavailable in the existing CGO-disabled toolchain.

