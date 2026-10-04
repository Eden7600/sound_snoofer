# Proposal

## Why

Frequently connecting and disconnecting microphones and playback devices forces manual Voicemeeter reconfiguration. Sound Snoofer should select the best available device independently for each configured input or output and restore preferred devices when they return.

## What Changes

- Introduce a Windows Go application with device discovery, a deterministic routing planner, and a foreground reconciliation loop.
- Support Banana now and Potato through runtime edition detection and capability validation.
- Configure independent ordered device preferences: Volt 2 before webcam microphone; AirPods before SteelSeries speakers. Match public device names with user-configured Go regex patterns, scoped by direction and driver type.
- Provide read-only inventory, dry-run planning, one-shot application, and explicit live watching.
- Debounce device changes, apply only necessary device assignments, verify results, and recover after Voicemeeter restarts.
- Keep unrelated mixer settings and existing Element processing paths intact. Default to leaving an assignment unchanged when no configured candidate is available, and explain that state.
- Defer Element profile switching, ASIO insert processing, automatic app launching, Windows default-device changes, tray UI, and startup integration to later changes. Fixed routes retain WDM support. Studio mode selects Volt ASIO on A1, maps mono inputs 1 and 2 to separate stereo strips, allocates playback to the lowest free output, migrates playback sends, and continuously enforces semantic source rules.

## Capabilities

### New Capabilities

- `device-discovery`: Enumerate input/output devices with identity, direction, driver type, and availability evidence, and report the running Voicemeeter edition.
- `priority-routing`: Validate user configuration and select independent preferred input/output devices deterministically.
- `studio-routing`: Coordinate ASIO ownership, channel patching, dynamic playback allocation, and persistent virtual-input playback rules.
- `routing-reconciliation`: Preview, apply, verify, and watch device assignments with debouncing and recovery.

### Modified Capabilities

None; this active initial change now includes the user-confirmed ASIO and declarative routing scope.

## Impact

New Go module, command-line entry point, Windows Remote API adapter, configuration model, pure planner, and controller. Depends on the user's installed Voicemeeter Remote DLL; OpenSpec is development tooling only. Applying an assignment can interrupt audio, particularly when changing A1. Configuration must explicitly identify managed slots; no personal routing configuration is activated automatically.
