# Mic activity
## Why
Automatic mic selection only knows whether a mic is connected, not whether it delivers any signal. In a multi-mic ASIO setup, a lav with a dead battery or an unplugged desk channel still counts as present, so Auto keeps picking a mic nobody can hear.

Every available mic is also wired into Voicemeeter all the time: both ASIO channels are patched and the webcam is assigned.

## What Changes
- **Opt-in configuration.** A new `profiles.activity` block turns the behavior on:
  - `check`: N, the number of top options to check;
  - `silence_db`: the silence threshold;
  - `silent_after_s`: how long a mic must stay below it.

  Without the block, behavior is unchanged.
- **Wiring.** With activity on, Snoofer wires only:
  - the first N currently available options in the Normal mic priority;
  - the mic in use.

  Other desk/lav ASIO patches and the webcam input are cleared.
- **Metering.** Wired mics are measured from Voicemeeter's pre-fader input levels, so mic mute does not read as silence. No Windows capture streams are opened.
  - A mic below `silence_db` for `silent_after_s` is latched Silent.
  - Signal above the threshold for 300 ms makes it Active again.
  - A failed or missing reading is Unknown, never Silent.
- **Selection.** With the Normal source on Auto, the priority walk skips Silent mics.
  - If every option is Silent, the priority ignores silence. Silence never selects Off.
  - When a higher-priority mic becomes Active again, Auto returns to it.
  - An explicitly chosen mic is never changed.
- **Display.** Mic target options show *Silent* next to latched mics, in the GUI and as option labels on the deck.

## Supersedes
`selectable-voice-routing` says "Silence SHALL NOT trigger source changes", and its dead-lav scenario keeps a silent lav selected. This change replaces that rule **only** for the Normal source on Auto with `profiles.activity` configured. Explicit choices, VR profile choices and configurations without activity keep the old rule.

## Impact
- **Code:**
  - `internal/config` (activity schema);
  - `internal/voicemeeter` (pre-fader input levels);
  - `internal/control` (activity latch on the worker);
  - `internal/routing` (wired set, silence-aware priority);
  - `plugins/audio` (option labels).
- **Invariants:**
  - Unchanged: mic stack disabled or Off still clears every managed mic assignment and patch. Routing stays deterministic, because the latched state is an input to planning and the planner never reads levels itself.
  - Native reads stay on the worker.
