# Design
## Data to the GUI
The audio plugin publishes `audio.priorities` (Kind status, SurfaceOnly). Its `ViewData` holds:
- **`Devices`:** the non-virtual inventory from the last full snapshot (name, direction, driver, available). ASIO entries are drivers, labelled as such and never shown as connected.
- **`Lists`:** `interfaces`, `playback`, `webcam` and `microphones`. Each entry carries its fields, its match result (`Matches []string`) and whether it is in use.
  - Interfaces report matches for both the presence pattern and the driver pattern.
  - Microphone entries are option IDs (desk, lav, webcam, off) and report whether each is currently an option.
- **`Suggestions`** per list: unmatched candidate devices, each with `Exact` and `Device` patterns.

Matching reuses the routing helpers, with the same direction, driver and availability rules as planning, so the editor and the planner cannot disagree.

## Edit control
`audio.priority-edit` (Kind text, SurfaceOnly) accepts JSON `{list, op, index, value, field}`, following the pattern of `appaudio.edit`.
- **Operations:**
  - `add`: value is a candidate JSON or a mic ID.
  - `remove`.
  - `move`: value is `up` or `down`.
  - `set`: field is one of `pattern`, `driver`, `asio_pattern`, `presence_pattern`, `desk`, `lav`.
- **Validation:** the plugin decodes its current audio `config`, applies the edit to a copy, re-encodes it, and runs `config.Decode`. It saves only when that succeeds.
  - Lists the schema requires keep at least one entry, so removing the last Interface is refused.
- **Feedback:**
  - The control's `Value` is a hash of the saved audio config, so the GUI can see when an edit has settled.
  - A failed edit is reported on the control's `Status`, and the previous settings stay active.

## Live apply
Today the worker's `Load` returns the startup configuration. Instead, the plugin keeps its current runtime configuration behind a mutex.
1. A successful save rebuilds the runtime configuration with the same preparation used at start: profiles default, policy and soundboard hooks.
2. The plugin stores the rebuilt configuration and sends the worker `Reload`.
3. Reload keeps today's semantics: validation against the edition, revoking defaults protection, re-planning.
4. If reload fails, the previous configuration keeps running and the GUI shows the notice.

Edits are serialized by an edit mutex. The worker remains the only writer to Voicemeeter.

## Pattern generation
- `config.ExactPattern(name)` is `(?i)^` + `regexp.QuoteMeta(name)` + `$`.
- `config.DevicePattern(name)` is `(?i)` + `QuoteMeta(inner)` when the name ends in `(inner)` with a non-empty inner part, and `""` otherwise.
- Both are checked against the inventory before being offered.
- The GUI never builds regular expressions itself.

## GUI
- Routing sits after Audio in the sidebar (icon `route`).
- Each list is a card of numbered rows. A row holds:
  - an editable pattern (Enter or Apply);
  - a match badge: In use (active), No match (neutral) or Ambiguous (attention);
  - Up, Down and Remove icon buttons.
- Interface rows also have a presence pattern field and Desk/Lav channel inputs (0 = none).
- A Suggestions row under each list offers each unmatched device with Exact and Device buttons.
- The Mic priority card is an ordered list of option IDs, with Add offering the missing IDs.
- Text inputs keep focus and content across live updates, as on the other screens.
