---
applyTo: "runner/**/*.{ts,js,json},protocol/**/*.{ts,json},docs/trust-boundaries.md"
---

Use these references for trust-boundary and runner review comments:

- `/docs/trust-boundaries.md`
- `/docs/source-quality.md`
- `/.github/instructions/source-quality.instructions.md`
- `/runner/package.json`
- `/runner/scripts/boundary-check.js`
- `/runner/scripts/boundary-check.test.js`
- `/.source-quality-config.json`
- `/justfile`
- `/.github/workflows/ci.yml`

When reviewing changes in this scope, focus on:

- No trust-boundary bypasses from the runner into trusted Go code under `cmd/` or `internal/`.
- No ad-hoc cross-boundary message formats outside `protocol/schemas/` and approved fixtures.
- Boundary-check scripts and tests remain aligned with path, import-rule, and schema access changes.
- Runner Node engine range and script commands stay valid (`>=22.22.1 <25`).
- Tests preserve fail-closed behavior for boundary guardrails.
- Use `/docs/source-quality.md` as policy context; dedicated source-quality review coverage lives in `/.github/instructions/source-quality.instructions.md`.
- Tier 1 runner guardrail modules keep required top-of-file module docs and do not weaken source-quality enforcement through casual suppressions or config drift.
- When boundary rationale spans multiple files or behaviors, prefer updates to maintained docs/specs over only adding more inline comments.

Raise a high-confidence issue if a change weakens trust-boundary enforcement or bypasses boundary checks.
