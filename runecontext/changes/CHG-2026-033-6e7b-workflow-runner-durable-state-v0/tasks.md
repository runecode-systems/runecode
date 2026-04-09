# Tasks

## Runner Contract

- [ ] Implement untrusted runner orchestration with stable broker-facing contracts.
- [ ] Keep LangGraph internal and non-canonical.
- [ ] Align runner-facing run lifecycle state with the shared broker run-summary/run-detail vocabulary instead of defining a separate UI-only status model.

## Durable State

- [ ] Implement persisted run-state transitions and step-attempt tracking.
- [ ] Implement explicit crash recovery and idempotency rules.
- [ ] Persist approval-wait state with enough scope detail to resume safely after restart.
- [ ] Support multiple concurrent pending approvals, dedupe/supersession by approval-request identity, and explicit statuses (`pending`, `approved`, `denied`, `expired`, `superseded`, `cancelled`, `consumed`).
- [ ] Use the canonical approval-request identity shared with broker approval APIs as the stable approval identifier.
- [ ] Persist whether a pending approval is an exact-action approval or a stage sign-off and retain the canonical bound action hash or stage summary hash needed to validate consumption after restart.
- [ ] Persist enough run summary and drill-down state for the broker to produce stable `RunSummary` and `RunDetail` read models without scraping runner internals.
- [ ] Keep runner-only orchestration detail explicitly separate from any authoritative broker-exposed run posture.
- [ ] Treat authoritative backend/runtime facts as launcher-derived broker state, not runner-owned truth.
- [ ] Do not collapse `backend_kind`, runtime isolation assurance, provisioning/binding posture, and audit posture into one runner-local status field.

## Execution Loop

- [ ] Enforce propose, validate, authorize, execute, and attest transitions.
- [ ] Keep approvals typed, bounded, and resumable.
- [ ] Block only the scope bound to a pending approval and continue unrelated eligible work when resources and policy allow.
- [ ] Ensure approval consumption is bound to the exact request scope/hash so unrelated work cannot accidentally consume or satisfy the wrong approval.
- [ ] Supersede stale stage sign-off approvals when the bound stage summary hash changes before consumption.
- [ ] Ensure broker-exposed blocked-state and pending-approval summaries can be derived deterministically from runner state without inventing a second approval model.
- [ ] Ensure runner/orchestrator state integrates with launcher-backed runtime state without redefining launch/session/attachment semantics locally.

## Acceptance Criteria

- [ ] Runs can pause/resume and recover safely after process failure.
- [ ] Runs can accumulate multiple pending approvals while unrelated eligible work continues.
- [ ] Runner cannot bypass policy or direct execution boundaries.
