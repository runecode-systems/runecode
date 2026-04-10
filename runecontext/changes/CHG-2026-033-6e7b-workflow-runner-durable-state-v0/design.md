# Design

## Overview
Implement the untrusted runner orchestration and durable state authority for secure, resumable runs.

## Key Decisions
- Runner is untrusted and never directly executes privileged operations.
- Runner persistence stores untrusted orchestration state only.
- Pause/resume and crash recovery rely on durable state transitions.
- Pending approval blocks only the exact bound scope; unrelated eligible work may continue.
- Multiple pending approvals may coexist and survive restarts.
- Runner-internal durable state remains non-canonical and outside the cryptographic trust root unless exported into canonical protocol objects.
- All real execution remains brokered and policy-authorized.
- Broker-facing run and approval summaries are shared operator-facing contracts; runner state must align with them rather than inventing a second lifecycle vocabulary.
- Runner durable state may retain additional orchestration detail, but broker-visible run truth must remain an explicit translation into authoritative or advisory public fields.
- Authoritative backend/runtime facts come from launcher -> broker projection rather than from runner-local inference.
- Runner must not flatten backend kind, runtime isolation assurance, provisioning/binding posture, and audit posture into one local status string.
- Runner approval wait semantics must preserve the policy distinction between exact-action approvals and stage sign-off, including supersession when the bound stage summary hash changes.
- Broker remains authoritative for shared run truth, approval truth, and operator-facing read models; runner durable state is a resumable journal, not the source of public truth.
- Runner should report typed checkpoints and results to the broker; broker validates and projects them rather than accepting blind runner-owned status upserts.
- Stable logical workflow identities (`run_id`, `stage_id`, `step_id`, `role_instance_id`) survive retries and recovery. Retries and reruns mint separate attempt identities instead of mutating logical scope identity.
- Public run lifecycle stays on the broker lifecycle vocabulary. Runner may use richer internal orchestration states, but partial blocking or branch-local waits should surface through broker detail/coordination models rather than a second public lifecycle enum.
- Runner persistence should use an append-first journal plus snapshots, explicit schema versions, idempotency keys, and deterministic broker-wins reconciliation after restart.

## Ownership Model

### Broker-Owned Canonical State
- Broker remains authoritative for:
  - shared run lifecycle and blocked/recovering posture
  - persisted policy decisions and canonical approval objects
  - operator-facing `RunSummary` / `RunDetail` projection
  - gate/evidence references and artifact linkage

### Runner-Owned Durable State
- Runner durable state should contain only resumable orchestration mechanics such as:
  - workflow cursor / next eligible work
  - stable logical scope references
  - attempt identities and idempotency keys
  - local dependency graph / scheduler bookkeeping
  - pending approval waits and resume tokens
  - in-flight gate attempt bookkeeping
  - local replay checkpoints
- Runner durable state must not become the trust root for:
  - approval validity
  - backend/runtime posture
  - operator-facing lifecycle state
  - policy outcomes

## Identity Model

- `run_id` identifies one run instance.
- `stage_id` identifies one logical stage within the workflow/process definition.
- `step_id` identifies one logical step within a stage.
- `role_instance_id` identifies one logical execution role instance for a run.
- Retries, recovery, and reruns should use separate attempt identities such as:
  - `stage_attempt_id`
  - `step_attempt_id`
  - `gate_attempt_id`
- Stable logical identities should be the same identities used by policy requests, approvals, artifacts, launcher runtime evidence, and broker run-detail summaries.

## Runner -> Broker Contract Recommendation

- Runner should communicate progress through typed broker-facing orchestration reports rather than broker scraping runner-local persistence.
- The write surface should remain operation-specific and event-style, for example around:
  - run initialization / checkpoint reporting
  - step-attempt start and finish
  - approval-wait entered / cleared
  - gate-attempt start and result
  - run lifecycle checkpoint / terminal report
- Broker should validate these reports against canonical state and reject inconsistent transitions fail closed.
- Broker should project accepted reports into read models and durable shared state rather than promoting raw runner event payloads into operator truth without translation.

## Lifecycle Mapping

- Public broker lifecycle remains:
  - `pending`
  - `starting`
  - `active`
  - `blocked`
  - `recovering`
  - `completed`
  - `failed`
  - `cancelled`
- Runner internal orchestration states may be more granular, but they must map deterministically into this vocabulary.
- A run should be publicly `blocked` only when no eligible work can currently progress.
- Partial blocking or branch-local waits should surface through stage/role/coordination detail instead of a second public lifecycle state such as `partially_blocked`.

## Durable State Model

### Persistence Shape
- Use an append-first event journal plus periodic snapshots.
- Every record family and snapshot should carry an explicit schema version.
- Recovery should load the latest compatible snapshot and replay later journal entries deterministically.

### Recommended Journal Families
- `run_started`
- `stage_entered`
- `step_attempt_started`
- `action_request_issued`
- `approval_wait_entered`
- `approval_wait_cleared`
- `gate_attempt_started`
- `gate_attempt_finished`
- `step_attempt_finished`
- `run_terminal`

### Idempotency + Replay
- Every externally visible operation should carry a stable idempotency key.
- Replaying the same journal must not duplicate broker-visible side effects, approvals, or gate evidence linkage.
- Attempt identities should isolate retries from prior failed or superseded attempts.

### Reconciliation
- On restart or resume:
  - broker-owned canonical state wins for shared truth
  - runner journal provides resumable orchestration hints and outstanding local bookkeeping
  - disagreements should resolve through explicit reconciliation rules, never implicit merge heuristics
- Recovery should fail closed on unknown future journal versions or inconsistent canonical broker bindings.

## Approval Wait Semantics

- Exact-action approvals should bind one immutable action request and may be consumed once by the exact matching action.
- Stage sign-off approvals should bind one stage summary hash and become stale when that hash changes.
- Runner should persist enough scope and hash material to resume safely after restart, but approval creation and final approval truth should stay broker-owned and policy-derived.
- Multiple pending approvals may coexist. Runner scheduling should block only the exact bound scope and continue unrelated eligible work when policy and coordination allow.

## Main Workstreams
- Runner contract and packaging constraints.
- Durable state schema and migration rules.
- Propose-to-attest execution loop integration.
- Runner->broker checkpoint/result API alignment.
- Recovery, replay, and reconciliation semantics.
