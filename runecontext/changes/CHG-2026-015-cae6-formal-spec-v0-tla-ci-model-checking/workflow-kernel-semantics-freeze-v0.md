# Workflow-Kernel Semantics Freeze (v0)

This artifact freezes the canonical workflow-kernel semantics that subsequent protocol, runtime, formal-spec, and CI model-checking work must treat as authoritative for `CHG-2026-015`.

## 1) Approval lifecycle and consumption semantics

Canonical lifecycle vocabulary:

- `pending`
- `approved`
- `denied`
- `expired`
- `cancelled`
- `superseded`
- `consumed`

Frozen semantics:

- `approved` means a valid signed decision has been accepted for the current bound inputs.
- `consumed` means broker-trusted continuation application has occurred.
- Approval consumption is broker-only.
- When consumption occurs, it MUST be atomic with trusted application of the exact bound continuation.
- Terminal states are closed and terminal for the approval object:
  - `consumed`
  - `denied`
  - `expired`
  - `cancelled`
  - `superseded`

Implementation note: some current broker operations may still accept decision and consume in one trusted transaction, but that path is interpreted as an atomic path over this full lifecycle model, not as a collapse of `approved` and `consumed` semantics.

## 2) Stage sign-off binding semantics

Frozen contract:

- Stage sign-off binds the canonical `runecode.protocol.v0.StageSummary` object.
- `stage_summary_hash` means the RFC 8785 JCS hash of canonical `StageSummary` bytes.
- `summary_revision` is monotonic metadata only; it is non-authoritative by itself.
- `RunStageSummary` remains a derived/read-model surface, not a sign-off trust root.

Supersession rule:

- For stage sign-off approvals, stale requests are superseded when canonical stage-summary binding changes for the same logical stage scope under the active plan.
- The active plan scope is represented in sign-off binding via `plan_id` when present in request details/payload.

## 3) Gate-evidence authority semantics

Frozen authority split:

- Runner-reported gate result/evidence is advisory input only.
- Broker validates gate identity, gate attempt identity, trusted plan placement, and normalized input digest bindings.
- Broker materializes canonical trusted gate-evidence artifact/reference after validation.
- Any provided `gate_evidence_ref` must match the broker-canonicalized evidence digest.

## 4) Public lifecycle vs partial-blocking semantics

Frozen lifecycle rule:

- Public run lifecycle is `blocked` only when no eligible work can progress.
- Partial blocking remains detail/coordination state, surfaced via run detail and coordination models, not a second public lifecycle enum.

## 5) Closed v0 transition-obligation matrix

The closed `v0` matrix for authoritative transition audit obligations is defined in:

- `transition-obligation-matrix-v0.md`

This matrix is the required transition-obligation baseline for subsequent formal model and CI checks in this change.
