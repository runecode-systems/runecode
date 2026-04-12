# Tasks

## Watch Families

- [x] Define typed run watch/event family expectations.
- [x] Define typed approval watch/event family expectations.
- [x] Define typed session watch/event family expectations.

## Stream Semantics

- [x] Keep stream identity, ordering, and terminal-state rules aligned with shared broker stream semantics.
- [x] Define the minimum live event surface the alpha TUI needs.

## Acceptance Criteria

- [x] The alpha TUI can build live UX on typed watch/event families instead of relying primarily on polling plus logs.
  - Explicitly enforced via `protocol/fixtures/stream/{run-watch.success,approval-watch.success,session-watch.success}.json` plus cross-runtime fixture validation in `internal/protocolschema` and `runner/scripts/protocol-fixtures.test.js`.
- [x] The new watch families extend shared stream semantics rather than creating a second incompatible model.
  - Explicitly enforced via shared stream-sequence invariants (`stream_id` stability, request correlation stability, monotonic `seq`, exactly one final terminal event) against all watch stream fixtures, including negative coverage in `protocol/fixtures/stream/approval-watch.invalid-double-terminal.json`.
