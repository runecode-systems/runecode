# References

## Product Context

- **Mission:** `runecontext/project/mission.md`
- **Tech stack:** `runecontext/project/tech-stack.md`
- **Trust boundaries:** `docs/trust-boundaries.md`

## Relevant Standards

- `runecontext/standards/security/trust-boundary-interfaces.md`
- `runecontext/standards/security/trust-boundary-layered-enforcement.md`
- `runecontext/standards/security/trust-boundary-change-checklist.md`
- `runecontext/standards/security/audit-verification-scope-and-evidence-binding.md`
- `runecontext/standards/security/approval-binding-and-verifier-identity.md`
- `runecontext/standards/global/control-plane-api-contract-shape.md`
- `runecontext/standards/global/local-first-future-optionality.md`
- `runecontext/standards/global/protocol-schema-invariants.md`
- `runecontext/standards/global/protocol-registry-discipline.md`

## Related Specs

- `runecontext/specs/protocol-schema-bundle-v0.md`
- `runecontext/changes/CHG-2026-003-b567-audit-log-v0-verify/`
- `runecontext/changes/CHG-2026-005-cfd0-crypto-key-management-v0/`
- `runecontext/changes/CHG-2026-006-84f0-audit-anchoring-v0/`
- `runecontext/changes/CHG-2026-007-2315-policy-engine-v0/`
- `runecontext/changes/CHG-2026-008-62e1-broker-local-api-v0/`
- `runecontext/changes/CHG-2026-004-acdb-artifact-store-data-classes-v0/`
- `runecontext/changes/CHG-2026-010-54b7-container-backend-v0-explicit-opt-in/`
- `runecontext/changes/CHG-2026-012-f1ef-workflow-runner-workspace-roles-deterministic-gates-v0/`
- `runecontext/changes/CHG-2026-026-98be-image-toolchain-signing-pipeline/`
- `runecontext/changes/CHG-2026-028-647e-windows-microvm-runtime-support/`
- `runecontext/changes/CHG-2026-029-5e5e-macos-virtualization-polish/`
- `runecontext/changes/CHG-2026-030-98b8-isolate-attestation-v0/`
- `runecontext/changes/CHG-2026-033-6e7b-workflow-runner-durable-state-v0/`

## Protocol And Implementation Anchors

- `protocol/schemas/objects/RunSummary.schema.json`
- `protocol/schemas/objects/ActionPayloadBackendPostureChange.schema.json`
- `protocol/schemas/registries/audit_event_type.registry.json`
- `protocol/schemas/registries/error.code.registry.json`
- `cmd/runecode-launcher/main.go`
- `internal/trustpolicy/foundation_runtime.go`
- `internal/brokerapi/local_api_run_approval_types.go`
- `internal/brokerapi/local_api_run_summary_ops.go`

## Similar Implementations

None yet.
