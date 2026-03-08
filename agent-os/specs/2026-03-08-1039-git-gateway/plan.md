# Git Gateway (Commit/Push/PR) — Post-MVP

User-visible outcome: RuneCode can create commits and pull requests through a dedicated git-gateway role that verifies outbound changes match signed patch artifacts and enforces repo/branch allowlists.

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-08-1039-git-gateway/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

## Task 2: Git Target Allowlist Model

- Represent canonical repo identity and allowed operations in the run/stage manifest.
- Disallow URL-based policy decisions.

## Task 3: Secretsd-Backed Credentials

- Issue repo-scoped, operation-scoped short-lived tokens.
- Add revocation list support for active leases.

## Task 4: Patch Artifact Application + Outbound Verification

- Consume a signed patch artifact.
- Apply patch in a sparse/partial checkout by default.
- Verify outbound diff/tree hash matches the signed patch artifact before push.

## Task 5: PR Creation

- Create PRs via provider APIs.
- Attach run artifacts (spec links, gate results) as structured metadata.

## Acceptance Criteria

- Git operations are impossible from workspace roles.
- Outbound verification blocks pushes that do not match approved/signed patches.
