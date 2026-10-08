# Snoofer development standards

## Project and scope

Snoofer is a Windows Go plugin host with a core tray, GUI, lifecycle and semantic controls. Optional audio, VR, Stream Deck and media plugins own their integrations. Audio preserves the Voicemeeter routing and recording invariants below. Use the Go version declared in `go.mod`. Do not redistribute Voicemeeter's DLL.
These instructions apply throughout the repository unless a deeper AGENTS.md
provides more specific guidance. Explicit user instructions take precedence.

## OpenSpec workflow

- Design every code change in OpenSpec before implementation, including fixes,
  refactors and tests. No ad hoc code edits outside the relevant change's scope.
- Read the applicable proposal, requirements, design and tasks before editing.
  Check current implementation too; a planned feature is not proof it exists.
- Update planning artifacts before changing behavior or expanding implementation
  scope. Surface material conflicts rather than silently choosing between specs.
- A request to prepare a spec produces planning artifacts only. A request to
  implement authorizes work within that design; avoid redundant approval gates.
- Mark a task complete only when its stated implementation and verification are
  complete. Leave unperformed hardware acceptance unchecked.
- Keep existing user changes, personal configuration and unrelated work intact.
- This file and documentation-only corrections do not require invented behavioral
  requirements; use the appropriate documentation/tooling workflow when needed.

## Implementation Planning
- IMPORTANT: Before implementing any feature, thoroughly search the codebase to check if it's already implemented or partially implemented
- Always prefer reading existing similar implementations as reference before creating new patterns
- When implementing cross-system features (frontend -> API -> backend), trace the complete data flow to ensure fields are properly passed through all layers


## Go style and maintainability

- Run `gofmt` on changed Go files. Group standard-library imports separately from
  external and project imports; avoid aliases unless they improve clarity.
- Write readable statements and blocks. Do not compress multiple operations onto
  one line or treat formatting as a substitute for readable structure.
- Prefer clear control flow and early error returns over deeply nested branches.
- Use descriptive names for domain concepts and long-lived state. Short names are
  appropriate for small, obvious scopes and conventional receivers or indices.
- Keep packages cohesive and dependency direction clear. Avoid generic `utils`
  packages, speculative frameworks and abstractions without a concrete need.
- Define small interfaces at consumer boundaries when they express real behavior.
  Do not create an interface for every concrete type or solely to mirror a mock.
- Keep the exported API small. Document exported declarations and non-obvious
  invariants; comments should explain intent, constraints or trade-offs.
- Prefer the standard library where suitable. Explain new dependencies and keep
  `go.mod`/`go.sum` changes intentional; avoid unrelated dependency upgrades.
- Use build constraints for platform-specific implementations. Keep unsafe/native
  code isolated in the Voicemeeter adapter and document ABI assumptions.

## Errors and state

- Check returned errors, including persistence and cleanup errors when meaningful.
  If an error is intentionally ignored, make the reason clear.
- Return contextual errors and use `%w` when callers should inspect the cause.
  Use `errors.Is`/`errors.As` instead of comparing error-message strings.
- Do not panic for expected device absence, invalid configuration or API failure.
- Separate concise user-facing status from full diagnostic details. Do not encode
  control flow or state transitions by parsing presentation strings.
- Distinguish requested, queued, pending, observed, failed and unknown state.
  A successful setter call alone does not prove the mixer applied the change.
- Preserve explicit distinctions between missing values and valid zero values,
  especially in native readbacks and optional configuration fields.

## Concurrency and resource ownership

- Every goroutine needs an identifiable owner, cancellation path and shutdown
  behavior. Avoid unbounded queues, leaked goroutines and blocking UI operations.
- Pass `context.Context` explicitly, normally as the first argument. Retain it in
  framework state only when required by the framework's method signatures.
- Serialize native audio operations through the owning worker. Preserve OS-thread
  affinity where required by the adapter and hold live writer ownership for writes.
- Do not pass mutable maps or slices across worker/UI boundaries without a clear
  ownership transfer, copy or synchronization policy.
- Identify who closes channels and releases files, locks, timers and DLL handles.
  Keep shutdown bounded and release resources on failure as well as success.

## Architecture and audio invariants

- Keep routing calculations deterministic and separate from native reads/writes.
  The controller applies and verifies plans; the GUI presents state and submits
  typed actions. Rendering and browsing selections must never write audio settings.
- Preserve save-before-apply, stale-action rejection, atomic saved-choice updates,
  drift checks and bounded verification. GUI/watch default to live; --dry-run
  explicitly selects preview. Development smoke tests must opt into preview.
- Mic stack disabled disconnects all managed mic and Element-return sends while retaining Normal/VR targets. It must
  not disable computer playback/capture or operate recorder transport.
- Disabling also clears managed microphone input assignments and ASIO input patches;
  the selected ASIO interface remains assigned to A1 and playback retains its normal output.
- ASIO priority selects one interface for A1 before microphone/playback selection. Only that interface supplies eligible ASIO inputs/outputs. Per-interface desk/lav channel mappings feed stereo inputs 1/2 (L/R); zero means unavailable. Playback takes the lowest free output and
  its routing follows that output. Selecting the loaded ASIO interface for playback reuses reserved
  ASIO A1; other playback devices still use the lowest free output. Device matching
  uses Go regular expressions.
- Output slots (up to three) name one static device each. A bus holding a slot device
  is not free for Playback (Playback takes the last slot's bus only when nothing else
  is left) and stays reserved while the device is disconnected. Slot devices are never
  playback candidates. Snoofer owns only the slot's source sends (playback sources,
  monitor tap, soundboard, tape) on its bus; removed slots become unmanaged.
- Treat presence, absence and ambiguous matches distinctly; installed ASIO
  drivers alone do not prove hardware presence. Preserve fallback priorities.
- Potato voice routing uses B2 to Element through AUX Virtual ASIO, AUX as the
  processing return, and B3 for the application microphone. Prevent AUX-to-B2 loops.
- Under the recording profile, B1 is the recording mix. Post mic capture is a preference: effective Direct uses Pre, visibly marked
  as a fallback, and restores Post when Element processing is active.
- Recorder capture arming and tape playback sends are different controls.
  Preserve native file format, output directory and other unowned settings.
- Start/Stop are explicit one-shot commands, not persisted desired state. Never
  automatically retry an uncertain Start or start recording on launch/reconnect.
- Changes to these invariants require an explicit OpenSpec design update.

## Git commits

Stop and commit frequently throughout every implementation session. The user reviews each commit as it lands and may return with suggestions. Commit completed, validated logical blocks that make sense independently for review; do not accumulate unrelated work until the end. Structure proposals, specifications and task lists around these commit-sized checkpoints. Report each completed commit and its validation. Never include unrelated user changes or generated runtime files.

## Commands

All commits must use Conventional Commits: `type(scope): description`, with an
optional scope (for example, `feat(tui): add source selection menu` or
`chore: initialize repository`). Use an imperative, concise description. Mark
breaking changes with `!` and explain them in a `BREAKING CHANGE:` footer.

Run from the repository root; use an explicit tool path if Go is not on PATH.

```powershell
gofmt -w <changed-go-files>
go test ./... -timeout 30s
go vet ./...
./scripts/build.ps1
go test -race ./...
node node_modules/@fission-ai/openspec/bin/openspec.js validate <change-name> --strict
```

The race detector requires a supported platform/toolchain; do not install or
change the toolchain silently to make it run. Use scripts/build.ps1 for every app
build; bin/snoofer.exe is the only application build output. Never create alternate
names or output directories. The user authorizes stopping and restarting this
repository's Snoofer for builds without asking again. Use scripts/stop.ps1 (also invoked by build.ps1) for graceful shutdown and wait for exit. Never use Stop-Process/taskkill/TerminateProcess on the audio host as routine build or test cleanup. A timeout aborts the build; do not silently force-kill. Restore the prior launch configuration after validation. Do not stop Voicemeeter, Element or unrelated apps. The build
script closes the repository host normally and refuses any remaining locked outputs; scripts/check.ps1 delegates builds to it.
Report what changed, what was verified and any remaining limitations concisely.

# Behavioural guidelines

**Tradeoff:** These guidelines bias toward caution over speed. For trivial tasks, use judgment.

## 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:
- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

## 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.

When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

## 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:
- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:
```
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.

---

**These guidelines are working if:** fewer unnecessary changes in diffs, fewer rewrites due to overcomplication, and clarifying questions come before implementation rather than after mistakes.

## UI language
Users know this application. Prefer short nouns and state words, omit redundant
Active/Enablement prefixes and tutorial prose, and let icons/group headings supply
context. Keep distinctions needed for correctness and concise consequential-action
warnings; detailed diagnostics remain available. Compact surfaces use ShortLabel
when an icon makes the full label redundant. See internal/streamdeck/AGENTS.md for
the Stream Deck graphics and copy rules.

Before changing any UI, read docs/ui-contract.md. It is the shared baseline for goals, tone, content, vocabulary, color semantics and stable interaction/layout rules. Routine builds must not alter that baseline. Update the contract and focused presentation tests for intentional UX changes; retain unrelated behavior and verify actual renderer output.

Playback gain/meter follows the resolved listening destination. Never copy, persist or reset gains during routing. Playback mute is persisted application intent, imposed and verified in both directions; native mute must not overwrite that preference. Fixed A1/A2 controls are not user-facing controls.
