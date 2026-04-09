# Tasks

## Git Target Allowlist Model

- [ ] Represent canonical repo identity and allowed operations in the run/stage manifest.
- [ ] Disallow URL-based policy decisions.
- [ ] Model git destinations through the shared typed `DestinationDescriptor` / allowlist-entry pattern rather than a git-only ad hoc destination shape.
- [ ] Define a dedicated `git-remote-ops` approval trigger category for push/tag/PR creation so approval profiles can treat remote state changes explicitly.

Parallelization: can be designed in parallel with policy engine gateway allowlist work; it depends on stable destination descriptor schemas.

## Secretsd-Backed Credentials

- [ ] Issue repo-scoped, operation-scoped short-lived tokens.
- [ ] Add revocation list support for active leases.

Parallelization: can be implemented in parallel with `secretsd` lease work; it depends on stable lease semantics and audit event types.

## Patch Artifact Application + Outbound Verification

- [ ] Consume a signed patch artifact.
- [ ] Apply patch in a sparse/partial checkout by default.
- [ ] Verify outbound diff/tree hash matches the signed patch artifact before push.

Parallelization: can be implemented in parallel with artifact store and protocol schema work; avoid conflicts by agreeing on patch artifact format and signing envelope.

## PR Creation

- [ ] Create PRs via provider APIs.
- [ ] Attach run artifacts (spec links, gate results) as structured metadata.
- [ ] Audit remote git operations with the standard gateway network fields: allowlist id, destination descriptor, bytes, timing, and outcome.

Parallelization: can be implemented in parallel with provider-specific API adapters once the core git-gateway boundary is stable.

## Acceptance Criteria

- [ ] Git operations are impossible from workspace roles.
- [ ] Outbound verification blocks pushes that do not match approved/signed patches.
