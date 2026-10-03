# Verification

- Tests, vet, Windows build and strict OpenSpec validation passed.
- Model tests cover physical ASIO preservation, recognized virtual names/IDs, truncated names, case folding, empty arrays and nonmutation. CLI text/JSON and TUI rendering tests pass.
- Preview-only terminal smoke: navigated to Devices at 80 columns; observed webcam, SteelSeries, Bluetooth and Volt entries with no VB-CABLE or Voicemeeter virtual ASIO entries. Quit cleanly without mixer writes.
- Read-only rebuilt CLI JSON inventory likewise contains the ten physical-driver entries and omits the eight observed virtual-driver entries.
