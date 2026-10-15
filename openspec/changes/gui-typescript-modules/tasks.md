# Tasks

- [x] 1. `docs(gui)`: specify the behavior-preserving TypeScript migration and validation.
- [x] 2. `build(gui)`: add pinned tooling and typed shared modules; preserve pure model tests and
  reproducible icons; verify type/lint/format and model checks.
- [x] 3. `refactor(gui)`: extract the existing screen builders and provider view types in logical
  screen groups; verify strict checking and retain current behavior.
- [x] 4. `refactor(gui)`: switch the entry point, browser setup, Tailwind sources and canonical
  embedded build to the esbuild bundle; remove superseded source files and verify browser coverage.
- [x] 5. `test(gui)`: finish strict checking/formatting of browser tests and make all frontend gates
  default in `check.ps1`; prove representative violations fail the gates.
- [ ] 6. Validate the full repository check, fixture drift, GUI suite budget, OpenSpec, canonical
  build and native desktop smoke; inspect packaged screens and restore the previous launch.
