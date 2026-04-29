## Applicable Standards
- `standards/product/roadmap-conventions.md`
- `standards/security/trust-boundary-interfaces.md`
- `standards/security/trust-boundary-layered-enforcement.md`
- `standards/security/policy-evaluation-foundations.md`
- `standards/security/approval-binding-and-verifier-identity.md`
- `standards/security/runner-durable-state-and-replay.md`
- `standards/global/control-plane-api-contract-shape.md`
- `standards/global/protocol-schema-invariants.md`
- `standards/global/deterministic-check-write-tools.md`

## Resolution Notes
Migrated from the legacy spec standards list and refreshed to canonical RuneContext standard paths, then refocused to cover deterministic authoring, non-authoritative accelerator posture, and safe extension of the contract-first workflow substrate.

This now also includes the rule that later generic workflow authoring must build on `CHG-2026-050-e3f8-workflow-definition-contract-binding-v0` and must not reopen the shared typed git request, signed patch artifact, or exact-action approval contracts.

It also now includes the CHG-049 inheritance that:
- later custom workflow adoption uses an explicit separate registration/catalog path
- custom workflows must not override or shadow product-shipped built-in workflow identities
- draft-like workflow authoring must preserve artifact-first plus explicit promote/apply semantics where applicable
