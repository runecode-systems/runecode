# Tasks

## Runner Contract

- [ ] Implement untrusted runner orchestration with stable broker-facing contracts.
- [ ] Implement runner execution as consumption of one broker-compiled immutable `RunPlan` rather than runner-local workflow planning.
- [ ] Introduce the `RunPlan` identity and bind runner journal, checkpoints, and results to it.
- [ ] Keep LangGraph internal and non-canonical.
- [ ] Align runner-facing run lifecycle state with the shared broker run-summary/run-detail vocabulary instead of defining a separate UI-only status model.
- [ ] Keep broker as the authoritative owner of shared run truth and operator-facing projections; runner durable state must remain explicit advisory orchestration state only.
- [ ] Define typed runner->broker orchestration report families for checkpoints and results rather than relying on broker scraping runner-local persistence.
- [ ] Ensure broker validates runner-reported transitions and rejects inconsistent state updates fail closed.
- [ ] Keep the runner thin:
  - plan loader
  - broker client
  - journal/snapshot store
  - scheduler
  - executor adapters
  - report emitter
- [ ] Explicitly avoid runner-local authorization, runner-local approval truth, and runner-local workflow/gate planning semantics.

## Durable State

- [ ] Implement persisted run-state transitions and step-attempt tracking.
- [ ] Implement explicit crash recovery and idempotency rules.
- [ ] Persist active plan identity and fail closed on stale or superseded plan bindings during replay.
- [ ] Persist approval-wait state with enough scope detail to resume safely after restart.
- [ ] Support multiple concurrent pending approvals, dedupe/supersession by approval-request identity, and explicit statuses (`pending`, `approved`, `denied`, `expired`, `superseded`, `cancelled`, `consumed`).
- [ ] Use the canonical approval-request identity shared with broker approval APIs as the stable approval identifier.
- [ ] Persist whether a pending approval is an exact-action approval or a stage sign-off and retain the canonical bound action hash or stage summary hash needed to validate consumption after restart.
- [ ] Persist enough run summary and drill-down state for the broker to produce stable `RunSummary` and `RunDetail` read models without scraping runner internals.
- [ ] Keep runner-only orchestration detail explicitly separate from any authoritative broker-exposed run posture.
- [ ] Treat authoritative backend/runtime facts as launcher-derived broker state, not runner-owned truth.
- [ ] Do not collapse `backend_kind`, runtime isolation assurance, provisioning/binding posture, and audit posture into one runner-local status field.
- [ ] Use one stable logical identity model across runner state:
  - `run_id`
  - `stage_id`
  - `step_id`
  - `role_instance_id`
  - separate retry/recovery/gate attempt identities
- [ ] Implement durable state as a versioned append-first journal plus snapshots rather than a mutable monolithic status blob.
- [ ] Put explicit schema-versioning and migration rules on journal and snapshot record families.
- [ ] Use stable idempotency keys so journal replay does not duplicate broker-visible lifecycle updates, approvals, or gate evidence linkage.
- [ ] Define deterministic reconciliation rules where broker canonical shared state wins and runner local state supplies only resumable orchestration hints.

## Execution Loop

- [ ] Enforce propose, validate, authorize, execute, and attest transitions.
- [ ] Execute only work that is explicitly present in the active `RunPlan`.
- [ ] Keep approvals typed, bounded, and resumable.
- [ ] Block only the scope bound to a pending approval and continue unrelated eligible work when resources and policy allow.
- [ ] Ensure approval consumption is bound to the exact request scope/hash so unrelated work cannot accidentally consume or satisfy the wrong approval.
- [ ] Supersede stale stage sign-off approvals when the bound stage summary hash changes before consumption.
- [ ] Ensure broker-exposed blocked-state and pending-approval summaries can be derived deterministically from runner state without inventing a second approval model.
- [ ] Ensure runner/orchestrator state integrates with launcher-backed runtime state without redefining launch/session/attachment semantics locally.
- [ ] Map runner internal orchestration states deterministically into the shared public broker lifecycle vocabulary.
- [ ] Treat partial blocking as run-detail coordination/state detail rather than minting a second public lifecycle enum.
- [ ] Report step-attempt starts/finishes, approval waits, gate attempts/results, and terminal checkpoints through typed broker-facing report contracts.
- [ ] Reject execution when plan-bound executor, gate, or scope bindings do not match the active trusted plan.

## Acceptance Criteria

- [ ] Runs can pause/resume and recover safely after process failure.
- [ ] Runs can accumulate multiple pending approvals while unrelated eligible work continues.
- [ ] Runner cannot bypass policy or direct execution boundaries.
- [ ] Broker remains the only shared source of operator-facing run truth after runner restart or recovery.
- [ ] Stable logical workflow identities survive retries and restarts while attempt identities track reruns separately.
- [ ] Runs cannot silently drift from the broker-compiled `RunPlan`; stale or superseded plan bindings fail closed.
