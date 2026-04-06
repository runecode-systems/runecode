## Summary
RuneCode establishes an instance-global, append-only audit ledger owned by `auditd`, with typed audit payloads wrapped in detached RFC 8785 JCS-signed envelopes, Merkle-sealed segments, authoritative local verification, and machine-readable verification reports for later review.

## Problem
RuneCode's security model depends on evidence-backed execution, explicit approvals, typed policy decisions, gateway mediation, artifact flow controls, and later isolate attestation and anchoring. A basic append-only log is not enough: the system needs an audit foundation that remains verifiable across signer rotation, segmentation, import/restore, future concurrency, later anchoring, and later proof-oriented work without rewriting the substrate.

The current migrated placeholder does not yet lock the contracts that future changes depend on most heavily:
- detached signed-object placement for audit records
- canonical record identity and chain semantics
- instance-global ledger ownership versus per-run primary logs
- segment sealing and anchoring targets
- typed cross-object references and signer-evidence linkage
- machine-readable verification results and degraded-posture reporting

If those choices are deferred, later changes such as audit anchoring, broker local API, minimal TUI, workflow concurrency, secrets lease auditing, and isolate attestation will either duplicate authority or require a second audit-model rewrite.

## Proposed Change
- Audit Ledger + Evidence Model.
- Typed Audit Event Contract.
- Append-Only Writer + Recovery Rules.
- Segment Sealing + Lifecycle Model.
- Verification + Machine-Readable Findings.
- Redaction Boundaries + Audit Views.

## Why Now
This work remains scheduled for `v0.1.0-alpha.2` and is foundational for the rest of the trusted control-plane roadmap. Locking the audit substrate now avoids rework in `CHG-2026-004`, `CHG-2026-005`, `CHG-2026-006`, `CHG-2026-007`, `CHG-2026-008`, `CHG-2026-013`, `CHG-2026-027`, `CHG-2026-030`, and `CHG-2026-031`, all of which assume durable, reviewable, fail-closed audit evidence.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.
- Trusted Go verification remains authoritative for audit admission and verification; runner-side checks remain parity/supporting only.

## Out of Scope
- Implementing every future audit payload family owned by later changes.
- External anchoring targets and non-local anchoring transports.
- Re-introducing legacy Agent OS planning paths as canonical references.

## Impact
Keeps Audit Log v0 + Verify reviewable as a RuneContext-native foundational change and locks the long-lived audit substrate now, so later features mostly add typed payloads, verifier rules, and UI/API consumers rather than reworking ledger ownership, segment sealing, or verification semantics.
