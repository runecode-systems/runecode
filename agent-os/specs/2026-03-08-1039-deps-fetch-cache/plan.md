# Deps Fetch + Offline Cache — Post-MVP

User-visible outcome: workspace roles remain offline while dependencies can still be fetched via a dedicated role that produces a read-only cache artifact.

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-08-1039-deps-fetch-cache/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

## Task 2: Lockfile-Only Inputs

- Define inputs as lockfile artifacts only (no workspace access).

## Task 3: Registry Allowlist + Fetcher Role

- Allow egress only to approved package registries.
- Emit a read-only dependency cache artifact.

## Task 4: Workspace Consumption

- Attach cache artifacts read-only to workspace roles.
- Ensure cache usage is recorded in audit.

## Acceptance Criteria

- Workspace roles can build/test without any internet access.
- Dependency fetch behavior is auditable and policy-controlled.
