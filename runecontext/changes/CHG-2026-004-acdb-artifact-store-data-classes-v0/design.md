# Design

## Overview
Implement a content-addressed artifact store and a minimal data classification system with enforced role-to-role flow rules.

## Key Decisions
- Artifacts are hash-addressed and immutable.
- Unknown or ambiguous artifacts are classified as the most restrictive class (fail-closed).
- Artifact contents are stored on encrypted-at-rest storage by default (no silent plaintext mode).
- Artifact retention/GC is required to avoid unbounded growth.
- `approved_file_excerpts` are only created via explicit human approval; unapproved excerpts use a more restrictive class (`unapproved_file_excerpts`) and are not eligible for third-party egress.
- Promotions are hardened: approvals are explicit, reviewable, rate-limited, and revocable via policy (no history rewriting).
- Derived evidence is stored as explicit artifacts with their own data class (e.g., `audit_verification_report`), but the artifact store does not become the authoritative source of truth for subsystems that define stronger primary ledgers. In particular, `runecontext/changes/CHG-2026-003-b567-audit-log-v0-verify/` owns the authoritative audit ledger; artifact-store copies of audit evidence are export/review products, not the primary audit record.
- JSON hashing/signing paths use RFC 8785 JCS canonical bytes, implemented through a pinned vendored snapshot behind local wrappers so runtime behavior matches the protocol contract without introducing a fragile upstream module dependency. Current trusted usage supports top-level object or array JSON values only, matching RuneCode's signed and persisted protocol surfaces.
- Public broker/API artifact views are derived operator-facing contracts built around `ArtifactReference`; daemon-private storage layout such as blob paths or local storage roots is not part of the boundary-visible artifact model.

## Main Workstreams
- Define MVP Data Classes
- Content-Addressed Artifact Store (CAS)
- Flow Matrix Enforcement
- Quotas + Limits (Minimal)
- Garbage Collection + Retention (Minimal)

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
