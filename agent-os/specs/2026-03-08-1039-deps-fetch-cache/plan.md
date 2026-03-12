# Deps Fetch + Offline Cache — Post-MVP

User-visible outcome: workspace roles remain offline while dependencies can still be fetched via a dedicated gateway role that produces a read-only cache artifact.

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-08-1039-deps-fetch-cache/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

Parallelization: docs-only; safe to do anytime.

## Task 2: Lockfile-Only Inputs

- Define inputs as lockfile artifacts only (no workspace access).

Parallelization: can be implemented in parallel with artifact store work; it depends on stable lockfile artifact schemas and data-class flow rules.

## Task 3: Registry Allowlist + Fetcher Role

- Allow egress only to approved package registries.
- Emit a read-only dependency cache artifact.

Parallelization: can be implemented in parallel with policy engine gateway allowlist work; it depends on stable destination descriptor schemas and broker limits.

## Task 4: Workspace Consumption

- Attach cache artifacts read-only to workspace roles.
- Ensure cache usage is recorded in audit.

Parallelization: can be implemented in parallel with workspace role work; it depends on stable artifact attachment semantics.

## Acceptance Criteria

- Workspace roles can build/test without any internet access.
- Dependency fetch behavior is auditable and policy-controlled.
