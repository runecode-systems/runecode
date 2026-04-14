## Applicable Standards
- `standards/product/roadmap-conventions.md`
- `standards/security/trust-boundary-interfaces.md`
- `standards/security/trust-boundary-layered-enforcement.md`
- `standards/security/trust-boundary-change-checklist.md`
- `standards/security/policy-evaluation-foundations.md`
- `standards/security/approval-binding-and-verifier-identity.md`
- `standards/security/audit-verification-scope-and-evidence-binding.md`
- `standards/security/trusted-runtime-evidence-and-broker-projection.md`
- `standards/security/runner-boundary-check.md`
- `standards/global/control-plane-api-contract-shape.md`
- `standards/global/source-quality-enforcement-layering.md`
- `standards/global/local-first-future-optionality.md`
- `standards/global/protocol-schema-invariants.md`
- `standards/global/protocol-registry-discipline.md`

## Resolution Notes
- Migrated from the legacy spec standards list and refreshed to canonical RuneContext standard paths.
- Expanded to include the trust-boundary, policy, approval-binding, audit-evidence, broker-contract, and protocol-discipline standards needed for a reduced-assurance container backend that still reuses the primary broker/policy/runtime foundation.
- Added standards coverage for trusted runtime evidence and broker projection because container mode must remain a shared operator/audit surface, not a backend-specific side channel.
- Added runner-boundary and source-quality coverage because this change spans launcher, broker, protocol, and TUI code while preserving the repo's trust-boundary and gate discipline.
