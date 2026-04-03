---
schema_version: 1
id: source-quality-guardrails-v0
title: Source Quality Guardrails v0
originating_changes:
  - CHG-2026-001-57d6-agent-os-to-runecontext-migration-umbrella
revised_by_changes: []
---

# Source Quality Guardrails v0

## Summary

RuneCode enforces language-aware, risk-tiered source-quality guardrails that prioritize maintainability and auditability without boilerplate-comment mandates.

## Durable Current-State Outcomes

- `docs/source-quality.md` defines policy scope, language-aware documentation expectations, and anti-boilerplate comment guidance.
- Deterministic Tier 1/Tier 2 enforcement and protected-surface expectations are documented.
- Source-quality checks are integrated into canonical gates (`just lint`, `just ci`) using deterministic check-only behavior.
- Reviewed exception and ratchet mechanisms are represented in checked-in policy/config surfaces.
- Source-quality review expectations are centralized via dedicated instruction coverage.

## Policy Invariants

- Complexity and decomposition pressure are preferred over comment-volume metrics.
- Suppressions require concrete rationale; security-sensitive Tier 1 suppressions follow stricter reviewed paths.
- Architecture rationale for trust-boundary, policy, protocol, secrets, and audit behavior belongs in maintained docs/specs when inline comments are insufficient.

## Related Artifacts

- `docs/source-quality.md`
- `.github/instructions/source-quality.instructions.md`
- `.source-quality-baseline.json`
- `.source-quality-config.json`

## Related Standards

- `runecontext/standards/global/source-quality-enforcement-layering.md`
- `runecontext/standards/global/language-aware-source-docs.md`
- `runecontext/standards/global/deterministic-check-write-tools.md`
- `runecontext/standards/ci/worktree-cleanliness.md`
