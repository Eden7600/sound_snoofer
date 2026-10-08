# Priority list editor
## Why
Interface, playback and webcam priorities, and the Normal microphone priority, are regular expressions inside the audio plugin's nested JSON. Changing them means hand-editing `snoofer.json`, guessing the exact device names Voicemeeter reports, escaping them, and restarting the plugin.

## What Changes
- **A Routing screen in the GUI** with editable priority lists:
  - **Interfaces:** the ASIO list. Each entry has a driver pattern, a presence pattern and desk/lav channels.
  - **Playback:** device candidates, WDM or ASIO.
  - **Webcam:** fallback mic candidates, WDM.
  - **Mic priority:** the Normal microphone order.
- **List operations:**
  - Each list supports add, remove, move up/down and editing a pattern.
  - Each entry shows what it currently matches: the device name, *No match*, or *Ambiguous: N devices*.
  - The entry in use is marked.
- **Suggestions:** connected devices of the right direction and driver that no entry matches are offered under each list. Adding one generates its pattern:
  - **Exact:** `(?i)^<escaped name>$`.
  - **Device:** for Windows names shaped like `Endpoint (Device)`, the escaped device part matched anywhere, so the entry survives Windows renaming the endpoint.
- **Saving:** every edit is validated with the strict decoder and saved to the audio plugin settings (atomic, revision-checked) before it is applied. It takes effect live through the worker's reload, with no plugin restart.

## Impact
- **Code:**
  - `plugins/audio`: priority view, edit control, save, live configuration source;
  - `internal/config`: pattern generation, list edits;
  - `internal/routing`: candidate match reporting;
  - `app/web`: Routing screen;
  - `scripts/check-gui.cjs`;
  - `docs/ui-contract.md`.
- **Invariants:** unchanged:
  - device matching stays Go regular expressions over device names;
  - the first candidate with exactly one available match wins;
  - ambiguity is skipped;
  - an installed ASIO driver is never evidence of hardware;
  - settings are saved before they are applied.
