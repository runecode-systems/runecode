---
applyTo: "runner/**/*.{ts,js,json},protocol/**/*.{ts,json},docs/trust-boundaries.md"
---

Use these references for trust-boundary and runner review comments:

- `/docs/trust-boundaries.md`
- `/runner/package.json`
- `/runner/scripts/boundary-check.js`
- `/runner/scripts/boundary-check.test.js`
- `/justfile`
- `/.github/workflows/ci.yml`

When reviewing changes in this scope, focus on:

- No trust-boundary bypasses from the runner into trusted Go code under `cmd/` or `internal/`.
- No ad-hoc cross-boundary message formats outside `protocol/schemas/` and approved fixtures.
- Boundary-check scripts and tests remain aligned with path, import-rule, and schema access changes.
- Runner Node engine range and script commands stay valid (`>=22.22.1 <25`).
- Tests preserve fail-closed behavior for boundary guardrails.

Raise a high-confidence issue if a change weakens trust-boundary enforcement or bypasses boundary checks.
