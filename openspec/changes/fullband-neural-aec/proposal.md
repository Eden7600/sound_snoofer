# Preserve upper vocal frequencies with hybrid AEC

## Why
The user prefers LocalVQE voice cleanup to WebRTC but hears reduced vocal quality from its 16 kHz bandwidth. Try the proposed hybrid: LocalVQE for lower frequencies and echo-controlled upper frequencies.

## What changes
Add an opt-in LocalVQE full-band engine using the existing v1.3 voice model and bundled WebRTC AEC3. Require a 48 kHz host stream. Preserve existing choices and saved selection until explicitly switched. No new downloads or dependencies.

## Impact
Native neural worker, native constructor wrapper, engine menu, UI contract and focused DSP/integration tests. Routing, gains, recorder and callback recovery remain unchanged.
