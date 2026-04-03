# Standards for Protocol & Schema Bundle v0

These standards apply to implementation work produced from this spec.

## Trust Boundary Standards

- `runecontext/standards/security/trust-boundary-interfaces.md`
  - broker local API plus `protocol/schemas/` and `protocol/fixtures/` are the only allowed shared boundary surfaces
- `runecontext/standards/security/trust-boundary-layered-enforcement.md`
  - protocol changes must preserve broker validation, policy enforcement, and runtime isolation as layered controls
- `runecontext/standards/security/trust-boundary-change-checklist.md`
  - schema/fixture changes are security-sensitive and must stay aligned with trust-boundary docs and guardrails
- `runecontext/standards/security/runner-boundary-check.md`
  - the runner may only consume shared protocol schemas/fixtures and must fail closed on boundary violations

## Determinism and CI Hygiene

- `runecontext/standards/global/deterministic-check-write-tools.md`
  - schema tooling and fixture workflows must default to check-only behavior and write only with explicit opt-in
- `runecontext/standards/ci/worktree-cleanliness.md`
  - CI must not mutate schemas, fixtures, or generated artifacts after the check entrypoint runs
