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
This change exists to freeze the workflow-definition and binding substrate needed for the first productive workflow pack without coupling it to later authoring and accelerator work.

That includes preserving one typed authority model for workflow identity, executor reuse, gate semantics, approval binding, audit linkage, and git remote-mutation composition.
