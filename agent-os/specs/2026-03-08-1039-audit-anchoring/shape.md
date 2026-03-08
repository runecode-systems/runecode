# Audit Anchoring — Shaping Notes

## Scope

Add optional external anchoring for audit roots and integrate it with verification.

## Decisions

- Anchoring is an explicit step and produces receipts.
- Failures are recorded; no history rewriting.
- Anchoring is the primary mitigation for a fully compromised local audit writer (post-MVP hardening).

## Context

- Visuals: None.
- References: `agent-os/specs/2026-03-08-1039-audit-log-verify-v0/`
- Product alignment: Strengthens tamper-evidence for sharing and forensics.

## Standards Applied

- None yet.
