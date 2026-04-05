# Design

## Overview
Implement the core policy evaluator that enforces manifests, role invariants, and explicit approvals.

## Key Decisions
- Deny-by-default everywhere; allow only via signed manifest.
- No automatic fallback to containers; container mode is explicit opt-in.
- MVP policy language is declarative and schema-validated (no general-purpose code execution during evaluation).
- Core security invariants are non-negotiable; any approval policy or UX setting may only tighten policy, never loosen it.
- Network egress is a hard boundary: workspace roles are offline; public egress is only via explicit gateway roles (model inference via `model-gateway`), and non-gateway network egress is not approvable.
- MVP uses checkpoint-style approvals (stage sign-off and explicit posture changes) instead of per-action nags.
- MVP supports a single approval profile (`moderate`); later profile expansion lives in `runecontext/changes/CHG-2026-014-0c5d-approval-profiles-strict-permissive/`.
- Approval requests and decisions are typed, hash-bound to immutable inputs, signed, and time-bounded (TTL/expiry); stale approvals are invalid.
- The user-involvement slider/profile maps ordinary action categories to approval frequency, batching, TTL, and minimum assurance, but it must not lower the fixed assurance floor for a small set of high-blast-radius operations.
- The fixed hard floor covers trust-root changes, security-posture weakening, authoritative restore/import/reconciliation, and deployment/bootstrap authority changes.
- Stage sign-off remains profile-controlled rather than automatically belonging to the fixed hard floor.
- Delivery channel is advisory only; local TUI, remote TUI, or messaging delivery must all converge on the same signed approval contract.
- `ApprovalDecision` is signed by the approval authority after verifying any required user cryptographic assertion rather than by trusting the delivery channel or a free-floating button click.
- Policy decisions and failures use a shared protocol error envelope and stable reason codes.

## Main Workstreams
- Role + Run/Stage Policy Model
- Invariants (Fail Closed)
- Approval Policy (MVP: Moderate)
- Backend Selection Rules
- Decision Outputs

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
