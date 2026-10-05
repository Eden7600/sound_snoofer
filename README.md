# Snoofer

A Windows tray application with optional compiled Audio, VR, Stream Deck and Windows media plugins. Core owns the tray, unified TUI, plugin lifecycle and semantic controls. Audio retains Sound Snoofer's Voicemeeter routing, recording and recovery behavior.

## Run

Launch `bin/snoofer.exe`; choose **Open controls** from its tray icon. Closing controls leaves Snoofer running. **Quit Snoofer** releases resources without resetting mixer routing or stopping recording.

Configuration defaults to `snoofer.json` beside the executable. Plugin dependency changes are confirmed together and restart the application. A new configuration starts with every plugin disabled. Personal settings were manually converted in `bin/snoofer.json`, with operational journals at their original paths.

Arrows navigate, Tab switches Settings/Plugins, Enter edits or confirms, Escape cancels, +/− adjusts numeric controls, R retries failed plugins, Q closes controls. Normal audio remains editable while VR overrides it. Stream Deck layouts are edited in the same TUI with Save/Cancel.

## Build and validate

Use the Go version in go.mod. Default audio builds require installed MSVC x64 tools and the Windows SDK for the callback companion; core-only builds do not. Do not redistribute Voicemeeter's vendor DLL.

```powershell
.\scripts\build.ps1
.\scripts\build.ps1 -Tags core -Output .local/snoofer-core.exe
.\scripts\build.ps1 -Tags no_audio -Output .local/snoofer-media-deck.exe
.\scripts\check.ps1
.\bin\snoofer.exe --check --config .\bin\snoofer.json
```

Keep `snoofer-audio-monitor.dll` beside audio-enabled executables. Build scripts use the Windows GUI subsystem to avoid a blank terminal. They do not stop an in-use application.

For interactive development preview, copy the envelope and audio sidecars into a repository-local scratch directory, update state_path, and use `--dry-run --config <copied-envelope>`. Preview avoids audio writes but can save UI settings; do not interactively smoke-test against personal state.

## Guides

- [Application and configuration](docs/application-guide.md)
- [Plugin authoring and composition](docs/plugins.md)
- [Troubleshooting](docs/troubleshooting.md)
- [Recording setup](docs/recording.md)
- [Implementation and validation](openspec/changes/modular-snoofer/implementation.md)
- [Remaining hardware acceptance](openspec/changes/modular-snoofer/tasks.md)

Audio maintenance commands use `snoofer --config <envelope> audio <command>`. These commands run audio alone, without VR process policy; use the tray for the complete application.

Callback progress proves processing, not audible output. Physical Volt/VR/Stream Deck acceptance is tracked separately from automated tests.

## License

[GNU AGPL version 3](LICENSE), AGPL-3.0-only.
