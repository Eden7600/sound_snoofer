## Decisions
Users know the application. Use one or two words, rely on icons, and show state without explanatory sentences. Core Control gains optional ShortLabel for compact surfaces; the normal Label stays explicit where the TUI has no icon. Audio supplies compact names for every built-in audio binding, media supplies transport names. Keep profile qualifiers only for fixed-profile controls where they distinguish actions. Runtime diagnostics remain detailed in TUI; deck statuses use Wait, Error, N/A or VR. Remove the disabled-stack explanatory status and shorten VR override wording. Do not rewrite personal layouts.

Remove only the 1px perimeter from every key. Retain the state badge, icon, antialiasing and hardware rotation. Direct icon is a straight input-to-output arrow, Element passes through an FX block. Pre/Post tap icons share a processing block and branch to a capture dot before/after it; selected state chooses the icon. Mic stack uses a power symbol, distinct from mute. Existing code-drawn primitives are the asset system; no raster asset dependency.

Use one build entry point: scripts/build.ps1 [-Tags ...]. Fixed application output bin/snoofer.exe and existing native companion in bin. Refuse locked app/companion before any build writes, with an instruction to exit Snoofer. Never create a renamed app executable, relocate output, or kill the running app. Reuse native monitor build/self-test. check.ps1 owns test/vet/OpenSpec/race checks and invokes build.ps1 rather than duplicating compilation. Preserve optional plugin build tags. Preserve user's config/journals.

## Validation
Check compact labels/statuses, command identity, distinguishable state icons, no key perimeter, and preview real renderer output. Test existing running-file guard without modifying outputs; run full Go tests/vet/OpenSpec and the build script once app has exited. Offline config check only, no automatic live launch.

## Consistency requirement
The user's follow-up expands this change to a stable cross-surface UI contract in docs/ui-contract.md: goals/tone/content, canonical wording, status precedence, color semantics and invariant interactions. Centralize deck palette/precedence, retain Studio shapes/placement, make ordinary Off neutral (power icon, not mute slash), and prevent normal TUI detail text from being warning-colored. Future incidental redesigns are forbidden by root/scoped AGENTS instructions. Tests cover competing state priorities.

## Verified result
Full tests, vet, all 34 strict OpenSpec validations, native callback self-test, canonical scripts/check.ps1 to scripts/build.ps1 compilation and installed offline config validation passed. Running-file guard was exercised and the prior executable hash remained unchanged. Final app output is bin/snoofer.exe; no alternate app output was created. Visual baseline inspected at native size and saved in docs/design/streamdeck-controls.png. Race checks were skipped because CGO is disabled; physical appearance acceptance remains open.

The user subsequently authorized stopping/restarting this repository's Snoofer for future builds without repeated confirmation. Root AGENTS records this permission. The build script retains its locked-file guard; the agent may handle lifecycle separately, preserving launch configuration and leaving unrelated audio apps alone.
