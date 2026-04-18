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
Migrated from the legacy spec standards list and refreshed to canonical RuneContext standard paths, then expanded to cover shared process-contract, policy/approval binding, and runner replay semantics required for extensible workflow composition.

This now also includes the rule that extensible workflows must reuse shared typed git request, signed patch artifact, and exact-action approval contracts rather than inventing process-local remote-mutation semantics.
