---
applyTo: "docs/source-quality.md,.source-quality-baseline.json,.source-quality-config.json,.golangci.yml,runner/eslint.config.*,tools/checksourcequality/**"
---

Use these references for source-quality review comments:

- `/docs/source-quality.md`
- `/docs/trust-boundaries.md`
- `/justfile`
- `/.golangci.yml`
- `/.source-quality-baseline.json`
- `/.source-quality-config.json`

Use this file as the detailed source-quality review guide when its scope overlaps with broader Go, runner, or CI instruction files.

When reviewing changes in this scope, focus on:

- Hard-fail rules stay deterministic, check-only, and consistent with `just lint` / `just ci`.
- Threshold, baseline, and suppression changes remain narrow, reviewable, and do not quietly weaken Tier 1 protections.
- Comment-quality checks avoid high-noise false positives and keep explanatory prose, annotation prefixes, and rationale comments usable.
- Tool-internal checker changes preserve correctness, false-positive/false-negative behavior, test coverage, and actionable output.
- Complex trust-boundary, policy, protocol-validation, secrets, and audit logic still triggers requests for maintained docs/specs/ADRs when inline comments are not enough.
- Output remains actionable: include rule name, file path, local context, observed vs expected values, and a concrete remediation direction.
- Reject source-quality suppressions that do not include concrete rationale or a reviewed exception path in the checked-in config/baseline surfaces.

Prefer comments that protect maintainability and auditability over style-only nitpicks.
