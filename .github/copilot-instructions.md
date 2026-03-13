# Copilot PR Review Instructions

When reviewing pull requests in this repository:

- Treat these files as source of truth for conventions:
  - `/docs/trust-boundaries.md`
  - `/docs/source-quality.md`
  - `/CONTRIBUTING.md`
  - `/justfile`
  - `/.github/workflows/ci.yml`
  - `/agent-os/standards/index.yml`
  - `/agent-os/standards/product/roadmap-conventions.md`
- Prioritize findings in this order: security, correctness, reliability, portability, maintainability.
- De-prioritize style-only comments unless they hide a functional risk or violate a documented convention.
- For each finding, include severity (`Critical`, `High`, `Medium`, `Low`), file path, impact, and a concrete fix recommendation.
- Prefer evidence from this repository and cite relevant file paths when possible.

Project context:

- Primary runtime and language: Go 1.25 (`go.mod`).
- Untrusted workflow runner: Node + TypeScript in `runner/` (Node `>=22.22.1 <25`).
- Canonical CI parity command: `just ci`.

Review expectations:

- Preserve the trust boundary between trusted Go components and the untrusted runner:
  - Do not allow runner-side access to trusted `cmd/` or `internal/` code.
  - Keep cross-boundary message contracts schema-driven via `protocol/schemas/` and approved fixtures.
- Never suggest changes that expose secrets or sensitive values in logs, errors, fixtures, tests, or generated artifacts.
- If build, lint, test, or boundary behavior changes, ensure parity remains with:
  - `just ci`
  - `go test ./...`
  - `cd runner && npm run lint`
  - `cd runner && npm test`
  - `cd runner && npm run boundary-check`
- Treat source-quality enforcement surfaces as protected review areas:
  - `/justfile`
  - `/tools/checksourcequality/`
  - `/.source-quality-baseline.json`
  - `/.source-quality-config.json`
  - `/.golangci.yml`
  - `/runner/eslint.config.*`
  - `/docs/source-quality.md`
  - `/.github/instructions/`
- For detailed source-quality review criteria, follow `/docs/source-quality.md` and `/.github/instructions/source-quality.instructions.md`.
- Distinguish hard-fail source-quality rules from review guidance:
  - hard-fail rules should stay deterministic and tool-friendly,
  - higher-judgment questions such as whether a subsystem needs a spec/doc/ADR update should remain review guidance.
- Reject source-quality suppressions that lack concrete rationale or a reviewed exception path.
- If a trust-boundary, policy, protocol-validation, secrets, or audit change needs paragraph-level explanation, ask for a maintained doc/spec/ADR update instead of only requesting more inline comments.
- If roadmap or spec docs change, verify they follow `agent-os/standards/product/roadmap-conventions.md`.
