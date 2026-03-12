# Artifact Store + Data Classes v0 — Shaping Notes

## Scope

Implement a content-addressed artifact store and a minimal data classification system with enforced role-to-role flow rules.

## Decisions

- Artifacts are hash-addressed and immutable.
- Unknown or ambiguous artifacts are classified as the most restrictive class (fail-closed).
- Artifact contents are stored on encrypted-at-rest storage by default (no silent plaintext mode).
- Artifact retention/GC is required to avoid unbounded growth.
- `approved_file_excerpts` are only created via explicit human approval; unapproved excerpts use a more restrictive class (`unapproved_file_excerpts`) and are not eligible for third-party egress.
- Promotions are hardened: approvals are explicit, reviewable, rate-limited, and revocable via policy (no history rewriting).
- Derived evidence is stored as explicit artifacts with their own data class (e.g., `audit_verification_report`).

## Context

- Visuals: None.
- References: `agent-os/specs/2026-03-08-1039-protocol-schemas-v0/`
- Product alignment: Enables explicit data movement and enforces blast-radius limits.

## Standards Applied

- None yet.
