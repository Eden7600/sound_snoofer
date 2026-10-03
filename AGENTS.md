# Voice Snooter development standards

## Project and scope

Voice Snooter is a Windows Go application that manages Voicemeeter devices,
routing and recording through the installed Remote API, with a persistent TUI.
Use the Go version declared in `go.mod`. Do not redistribute Voicemeeter's DLL.
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
  The controller applies and verifies plans; the TUI presents state and submits
  typed actions. Rendering and browsing selections must never write audio settings.
- Preserve save-before-apply, stale-action rejection, atomic saved-choice updates,
  drift checks, bounded verification and default dry-run behavior.
- Mic source Off disconnects all managed mic and Element-return sends. It must
  not disable computer playback/capture or operate recorder transport.
- Off also clears managed microphone input assignments and ASIO input patches;
  Volt remains assigned to A1 and playback retains its normal output.
- Volt ASIO owns A1 when active. Physical channel 1 feeds stereo input 1 (L/R),
  channel 2 feeds stereo input 2 (L/R). Playback takes the lowest free output and
  its routing follows that output. Device matching uses Go regular expressions.
- Treat presence, absence and ambiguous matches distinctly; installed ASIO
  drivers alone do not prove hardware presence. Preserve fallback priorities.
- Potato voice routing uses B2 to Element through AUX Virtual ASIO, AUX as the
  processing return, and B3 for the application microphone. Prevent AUX-to-B2 loops.
- Under the recording profile, B1 is the recording mix. Post mic capture requires
  enabled Element voice mode; never silently substitute dry mic audio.
- Recorder capture arming and tape playback sends are different controls.
  Preserve native file format, output directory and other unowned settings.
- Start/Stop are explicit one-shot commands, not persisted desired state. Never
  automatically retry an uncertain Start or start recording on launch/reconnect.
- Changes to these invariants require an explicit OpenSpec design update.

## Tests and verification

- Test observable behavior, transitions and meaningful failure paths. Avoid tests
  that merely repeat implementation details or add coverage without useful checks.
- Use table-driven cases when they clarify a behavior matrix. Give failures enough
  context to identify the scenario and expected versus observed result.
- Use controlled backends, temporary directories and deterministic clocks for
  device, persistence and timing tests. Avoid sleep-based synchronization.
- Cover cancellation, stale commands, disconnect/reconnect, ambiguity, partial
  application and failed readback when affected by a change.
- TUI changes need keyboard, selection, resize, error-state and text-sanitization
  checks, plus an interactive smoke test when interaction or layout changes.
- Automated tests must not depend on live audio hardware or change personal mixer
  settings. Use isolated profiles for UI smoke tests.
- Run focused tests during development, then the relevant regression suite, vet
  and Windows build before delivery. Do not repeatedly rerun unchanged checks.
- For concurrency changes, run the race detector where supported. Report missing
  toolchain prerequisites or unperformed checks explicitly; never imply a pass.
- Native API readback, successful disk output and audible correctness are separate
  acceptance claims. Do not mark listening/file/hotplug tests complete from mocks.

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
go build -o bin/voice-snooter.exe ./cmd/voice-snooter
go test -race ./...
node node_modules/@fission-ai/openspec/bin/openspec.js validate <change-name> --strict
```

The race detector requires a supported platform/toolchain; do not install or
change the toolchain silently to make it run. If the executable is in use, build
a clearly named replacement and report its path; do not silently stop live audio.
Report what changed, what was verified and any remaining limitations concisely.

## References

- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- [Google Go Style Guide](https://google.github.io/styleguide/go/guide)
- [Organizing a Go module](https://go.dev/doc/modules/layout)
- [Go race detector](https://go.dev/doc/articles/race_detector)

Use these as supporting guidance, with the concrete project rules above taking
precedence. Do not impose arbitrary coverage quotas, function-length limits or
mandatory architecture patterns without an agreed project need.
