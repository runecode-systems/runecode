## Applicable Standards
- `standards/product/roadmap-conventions.md`
- `standards/global/control-plane-api-contract-shape.md`
- `standards/global/project-substrate-contract-and-lifecycle.md`
- `standards/global/protocol-canonicalization-profile.md`
- `standards/security/trust-boundary-interfaces.md`
- `standards/security/trust-boundary-layered-enforcement.md`
- `standards/security/audit-verification-scope-and-evidence-binding.md`
- `standards/security/trusted-runtime-evidence-and-broker-projection.md`
- `standards/global/protocol-schema-invariants.md`
- `standards/global/protocol-registry-discipline.md`

## Resolution Notes
Migrated from the legacy spec standards list and refreshed to canonical RuneContext standard paths, then expanded to cover the stronger foundation clarified in planning:

- signed runtime-image and toolchain identity should reuse the existing canonicalization and verifier-record model rather than introducing a second runtime-only trust format
- runtime-image identity, launch evidence, and audit projection must remain typed, digest-addressed, and host-path-free
- platform-specific or backend-specific realization details must stay behind the trusted launcher boundary rather than forking public contracts
- runtime identity and validated project-substrate identity must remain distinct even when later evidence binds both
- performance and scaling concerns should be handled through verified local caches and backend-private optimizations, not by creating separate trust or launch architectures for smaller versus larger deployments
