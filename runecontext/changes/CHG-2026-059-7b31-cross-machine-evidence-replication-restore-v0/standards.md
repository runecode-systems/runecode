## Applicable Standards
- `standards/product/roadmap-conventions.md`
- `standards/security/trust-boundary-interfaces.md`
- `standards/security/trust-boundary-layered-enforcement.md`
- `standards/security/policy-evaluation-foundations.md`
- `standards/security/approval-binding-and-verifier-identity.md`
- `standards/security/secret-lease-lifecycle-and-binding.md`
- `standards/security/audit-verification-scope-and-evidence-binding.md`
- `standards/global/control-plane-api-contract-shape.md`
- `standards/global/deterministic-check-write-tools.md`

## Resolution Notes
This change extends the verification-plane foundation into cross-machine durability, restore, and publication-durability gating while preserving the existing trust-boundary split between trusted control-plane ownership and the untrusted runner.

It also relies on the shared exact-action approval discipline and durable prepared and execute lifecycle shape already frozen for other hard-floor remote-state-mutation lanes so evidence replication and publication recovery do not invent a weaker transport-local exception path.
