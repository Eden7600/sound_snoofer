# Design

## Scope and current callers

Recheck callers, tests, JSON fixtures and scripts immediately before each checkpoint. The
audit's old line numbers and estimates are not evidence that a symbol is still unused.

### 19. Worker state and obsolete helpers

`State.VRMic`, `VRPlayback`, `VRMicAvailable`, `VRPlaybackAvailable`, `PublishedAt` and
`ConfigPath` are populated in `worker.go` without production readers. Remove those fields
and only their computations, including the extra `routing.VRDevices` traversal used for
the old headset availability strings. Keep `VRConfigured`, profile selection/options,
`ObservedAt`, `Feedback` and all native observation/ownership logic. `routing.VRDevices`
itself has live planner, availability and profile callers and must remain.

`DependsOn` and its helper `MicStrips` have only a test caller. `NoticeAt` is test-only;
`NeedsAttention` has no callers. Remove these helpers and their dedicated assertions.
Retain and adapt meaningful tests of `ObserveMute`, `SetNotice`, `ResolveNotice`, pending
convergence, failed actions and success expiry through their live fields/worker behavior.
Do not delete whole mixed-purpose test files.

**Original retention, superseded by user-directed `audit-completion` on 2026-10-15:**
The initial cleanup kept `Action.ID`, `State.Acks`, `Ack`, `EditAck`, `EditError`,
deduplication, `Action.Edits`, `SettingEdit`, `maxDeferredEdits`, `EditBatch`, `EditsRow`
and batch-aware feedback while action outcomes were designed. GUI outcomes now use
`Action.Complete`; no production caller submits the legacy IDs or batches. Remove the
unused protocol and convert meaningful acknowledgement-based tests to callback outcomes
and observed state. Keep single-edit persistence failure, normal/VR availability checks,
stale restart confirmation, scoped feedback and settled-readback latency coverage.
Retain `Origin` because it clears prior notices, plus `Complete`, GUI session IDs and the
unrelated native `windowsaudio.Request.Ack` revocation handshake.

### 20 and 31. Strict decoding and small shared operations

`storage.Decode` already rejects duplicate keys recursively, unknown struct fields and
trailing documents. Add a table test before moving its callers: valid/defaulted objects,
duplicate top-level/nested/array-contained keys, unknown top-level/nested fields, malformed
or empty input, type mismatch, a second object, a trailing scalar and trailing junk.
Include repeated field names in separate sibling objects as a valid case. Retain existing
behavior for valid JSON `null` at this generic decoder; domain validation remains the
caller's responsibility.

`config.Decode` must still decode into its initialized defaults, then run `Config.Validate`.
`LoadChoices` must still record the state token, normalize source/rehearsal choices, validate
them and replace `Intent` only on success. Reuse `storage.Decode` for the syntax/field checks.
Preserve rejection categories, wrapped underlying JSON errors and configuration/saved-choice
context; a shared decoder may normalize low-level diagnostic wording. Do not branch on
error-message text or add a general error framework. Existing invalid-choice errors must
still reach `StateError`, and failed loads must not silently adopt partial intent.

Remove `config.uniqueKeys`, `config.WriteJournal` and `replaceBytes` only after updating all
callers. The journal forwarding sites are `internal/control/recovery.go`,
`internal/controller/mixer.go` and `internal/config/state.go`; call the existing storage
operations directly. Keep their atomic replacement, permissions and error propagation.

Hardware-bus parsing belongs in `internal/routing`, already imported by controller and
audio. Use one small parser returning the existing 1-based bus number and an `ok` flag;
convert to a zero-based native bus only at existing callers. Current gain/mute consumers
accept exactly `A1`–`A5`; `plugins/audio.hardwareBus` uses `strconv.Atoi` and also accepts
spellings such as `A01` and `A+1`. Keep that numeric parser behavior and retain the strict
two-character guard at gain/mute callers. Tests must pin these differences, invalid names,
out-of-range numbers and the A1/A5 boundaries. Do not broaden or tighten a boundary under
the label of deduplication.

Replace `firstNonEmpty` in Hue, AEC, App Audio and Audio with `cmp.Or`, preserving argument
order and whitespace semantics. Do not create another string utility.

Media, VR, Now Playing, Discord, App Audio, Hue and Stream Deck have the same `Stop` body:
cancel, then wait for done or the supplied context. One `snoofer` function can perform that
operation; each plugin retains its instance fields, worker ownership and cleanup. Do not
introduce a shared instance type or change specialized Audio/AEC/Soundboard shutdown.
Cover cancellation before waiting, completed work and context deadline, and retain the
plugin tests that prove helper goroutines actually join. Do not invent a priority when
both select cases are ready.

**Original thumbnail retention, superseded by user-directed `audit-completion`:**
Now Playing still accepts encoded data, validates dimensions and produces 128px Catmull-Rom
PNGs. Soundboard still loads bounded files, composites GIF disposal and produces 64px
ApproxBiLinear PNGs. Share only their final proportional fit, transparent canvas, scaling
and PNG/base64 encoding through `internal/artwork.Thumbnail(image.Image, size, scaler)`.
Keep input validation, GIF timing/disposal, caching and fallback behavior local; retain
pixel characterization, nonzero source bounds, transparency and source-immutability checks.

### 27. Native build setup

The six sites are `build-aec.ps1`, `build-media.ps1`, `build-monitor.ps1`,
`build-neural-aec.ps1`, `build-soundboard.ps1` and `tools/callback-probe/run.ps1`.
Use one dot-sourced script function for locating vswhere/MSVC, setting up vcvars64 and
writing/running a command file with checked exit codes. Callers retain command bodies,
working/output directories, dependency preparation, compiler flags, artifacts and error
context. No generic build graph, new command language or output layout.

Preserve `OutputDirectory`, staging, model/license copies, pinned dependency recovery and
media async-wait self-tests added by `audit-hardening`. Quote paths with spaces, fail on
missing tools/setup/compiler errors, and keep checked `Push-Location`/`Pop-Location`
ownership. Update the existing fake-command build tests to exercise the shared setup and
prove failure propagation; use isolated fake commands, not the live callback probe.
The final application build still runs only through `scripts/build.ps1`, producing
`bin/snoofer.exe`.

### 28. Native audio buffers

`internal/voicemeeter/callback/audio_buffer.h` already defines the shared C ABI. Add only
small C-compatible `static inline` helpers for buffer shape, a channel's read/write pointers
and pass-through copying. Preserve the structure layout and callback calling convention.

AEC/neural `Valid` check sample/channel counts; callback `valid` additionally rejects any
missing channel pointer before inserts/peak collection. That distinction is safety-critical.
Reuse the shape and per-channel helpers, retaining the callback's full-pointer loop. Copy
with `memmove`, preserve in-place and overlapping buffers, and copy only valid channel
pairs. Do not add allocation, locking, exceptions or work proportional to unvalidated
counts on the audio thread. A small native test must exercise the helpers from C and C++
with count limits, missing pointers, in-place/overlapping buffers and unchanged samples.
Retain callback and AEC failure/fallback behavior; compile the affected native companions
through the canonical build workflow after this checkpoint.

### 29 and 30. Explicit deferral and small removals

The original finding 29 retention is superseded by the user's `audit-completion` instruction.
Discord and Soundboard share `snoofer.OnOff(bool)` at all eight call sites, retaining exactly
`On` and `Off` for display and comparisons. No broader localization or rendering API is added.

Confirmed small removals:

- `Session.Current`, native `GetCurrentSession`/`current` JSON emission and only that
  assertion/fixture field in `sessions_test.go`. Session IDs, command target selection,
  transport, timing, metadata, bounded async operations and artwork remain unchanged.
- Uncalled `renderPresentation`. The `render` wrapper has one presentation-test caller:
  move that test to `renderArtwork` and retain its border/icon assertions. The live
  `presentation`/`renderKey`/frame renderer are not dead code.
- `controls.exitErr`, keeping `cmd.Wait()` and the existing close/join/done sequence with
  an explicit ignored-result comment explaining child exit is handled by session closure.
- The unused `sessionFlagUnlock` constant and uncalled `fakeSessions.setPeak` test helper;
  retain the presence logic and fake peak data used elsewhere.

## Checkpoints and limits

Keep worker cleanup, decoding/storage tests, Go helper consolidation, small removals,
native build setup and native buffer helpers independently reviewable. Run focused tests
at each checkpoint, then repository checks, GUI tests and canonical build. Native helper
self-tests prove buffer behavior, not live-device acceptance. No hardware probe or native
smoke is implied by preparing this plan. Finding 33 and GUI outcome/state redesign are
outside this cleanup.
