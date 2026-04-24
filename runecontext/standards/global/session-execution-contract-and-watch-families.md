---
schema_version: 1
id: global/session-execution-contract-and-watch-families
title: Session Execution Contract And Watch Families
status: active
suggested_context_bundles:
    - project-core
    - go-control-plane
    - protocol-foundation
---

# Session Execution Contract And Watch Families

When RuneCode exposes broker-owned session interaction surfaces that may include transcript-only turns, execution-bearing turns, reconnectable live status, and later continue/retry behavior:

- Keep plain transcript append and execution-bearing trigger submission as distinct typed operation families; successful transcript append must not imply execution authorization or replace `SessionExecutionTrigger*` semantics.
- Keep session object lifecycle, aggregate `work_posture`, transcript turn lifecycle, and per-turn execution lifecycle as distinct vocabularies; `SessionSummary` and transcript status fields are not substitutes for `SessionTurnExecution` truth.
- Project broker-owned execution detail through `SessionDetail` and `SessionTurnExecution`, including canonical run, approval, artifact, audit-record, and validated project-substrate bindings when present, rather than requiring clients to reconstruct execution truth from local caches or ad-hoc joins.
- Expose in-flight per-turn execution through a dedicated `SessionTurnExecutionWatch*` family; do not overload `SessionWatch*`, transcript streams, or summary-only surfaces with execution-specific semantics that evolve at a different cadence.
- Keep `execution_state`, `wait_kind`, `wait_state`, and `terminal_outcome` distinct; do not collapse cancellation into `execution_state`, and do not mint client-local terminal values outside the broker-owned contract.
- Keep trigger identity, turn identity, and replay identity distinct. Idempotent retries and continue replay must bind to persisted broker-owned `trigger_id`, selected `turn_id`, and request-hash identity rather than caller-supplied retry fields alone.
- Keep `trigger_source`, `requested_operation`, `approval_profile`, and `autonomy_posture` explicit in trigger requests, acknowledgements, and execution projections; do not infer those controls from message role, TUI mode, or client-local workflow state.
- Keep TUI, CLI, live chat, and autonomous execution as thin adapters over the same broker-owned session execution contracts and watch families rather than allowing each surface to grow a separate execution model.
