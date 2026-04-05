# Tasks

## Runner Contract

- [ ] Implement untrusted runner orchestration with stable broker-facing contracts.
- [ ] Keep LangGraph internal and non-canonical.

## Durable State

- [ ] Implement persisted run-state transitions and step-attempt tracking.
- [ ] Implement explicit crash recovery and idempotency rules.
- [ ] Persist approval-wait state with enough scope detail to resume safely after restart.
- [ ] Support multiple concurrent pending approvals, dedupe/supersession by approval-request identity, and explicit statuses (`pending`, `approved`, `denied`, `expired`, `superseded`, `cancelled`, `consumed`).

## Execution Loop

- [ ] Enforce propose, validate, authorize, execute, and attest transitions.
- [ ] Keep approvals typed, bounded, and resumable.
- [ ] Block only the scope bound to a pending approval and continue unrelated eligible work when resources and policy allow.
- [ ] Ensure approval consumption is bound to the exact request scope/hash so unrelated work cannot accidentally consume or satisfy the wrong approval.

## Acceptance Criteria

- [ ] Runs can pause/resume and recover safely after process failure.
- [ ] Runs can accumulate multiple pending approvals while unrelated eligible work continues.
- [ ] Runner cannot bypass policy or direct execution boundaries.
