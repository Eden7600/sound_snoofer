## Why
A global notice currently marks every Stream Deck button ERROR, and any routing change marks every button PENDING. This misrepresents unrelated controls and causes the entire deck to flash during normal edits.
## What Changes
Scope action feedback and pending readback to each button binding. Preserve independent media/open-controls availability. Derive the image cache from complete visible presentation state, including the exact affected controls.
## Impact
Shared control action feedback, Stream Deck presentation/cache and regression tests. No changes to routing writes, transport guards, HID protocol or TUI navigation.
