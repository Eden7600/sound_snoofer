# Tasks

- [x] 1. Review and commit proposal, design and requirements; pass strict validation before implementation.
- [x] 2. Add focused failing browser regressions for backoff/recovery, retained errors, refused/replayed actions and render failure scheduling.
- [x] 3. Implement recovery state, separate banner and shared send refusal; update UI contract without changing provider/action outcome semantics.
- [x] 4. Run frontend type/lint/format, focused and full browser checks; review both viewport sizes and commit the validated source change.
- [x] 5. Coordinate the canonical build and native acceptance after stable integration, preserving the previous launch configuration. Do not mark this complete from browser tests alone.

2026-10-15 native acceptance: a canonical diagnostic WebView2 fixture used a temporary
Desktop.State rejection wrapper. The connection banner retained the existing notice and
navigation; Record was explicitly refused with zero captured actions. Retry intervals grew
from 201 ms through 414/804/1605/3206 ms to approximately five seconds (5004–5016 ms).
Recovery cleared only the banner, resumed meters and 106–122 ms polling, retained the notice
and refusal, and sent no replay. EOF cleanup exited 0. This establishes native binding-fault
behavior, independently of the earlier approval timeout and ordinary EOF shutdown tests.
The normal canonical build passed and the prior bin/snoofer.json launch was restored, completing task 5.
