# References

## Product Context

- **Mission:** `runecontext/project/mission.md`
- **Roadmap:** `runecontext/project/roadmap.md`
- **Trust boundaries:** `docs/trust-boundaries.md`
- **Repository foundation summary:** `README.md`

## Related Changes

- `runecontext/changes/CHG-2026-012-f1ef-workflow-runner-workspace-roles-deterministic-gates-v0/`
- `runecontext/changes/CHG-2026-007-2315-policy-engine-v0/`
- `runecontext/changes/CHG-2026-008-62e1-broker-local-api-v0/`
- `runecontext/changes/CHG-2026-033-6e7b-workflow-runner-durable-state-v0/`
- `runecontext/changes/CHG-2026-035-c8e1-deterministic-gates-v0/`
- `runecontext/changes/CHG-2026-003-b567-audit-log-v0-verify/`
- `runecontext/changes/CHG-2026-004-acdb-artifact-store-data-classes-v0/`

## Security Standards

- `runecontext/standards/security/policy-evaluation-foundations.md`
- `runecontext/standards/security/approval-binding-and-verifier-identity.md`
- `runecontext/standards/security/runner-durable-state-and-replay.md`
- `runecontext/standards/security/trusted-local-artifact-persistence.md`
- `runecontext/standards/security/trusted-runtime-evidence-and-broker-projection.md`

## Protocol Foundation Standards

- `runecontext/standards/global/protocol-bundle-manifest.md`
- `runecontext/standards/global/protocol-schema-invariants.md`
- `runecontext/standards/global/protocol-registry-discipline.md`
- `runecontext/standards/global/protocol-canonicalization-profile.md`

## Canonical Protocol Surfaces

- `protocol/schemas/manifest.json`
- `protocol/schemas/objects/ActionRequest.schema.json`
- `protocol/schemas/objects/ActionPayloadStageSummarySignOff.schema.json`
- `protocol/schemas/objects/StageSummary.schema.json`
- `protocol/schemas/objects/ActionPayloadGateOverride.schema.json`
- `protocol/schemas/objects/PolicyDecision.schema.json`
- `protocol/schemas/objects/ApprovalRequest.schema.json`
- `protocol/schemas/objects/ApprovalDecision.schema.json`
- `protocol/schemas/objects/RunPlan.schema.json`
- `protocol/schemas/objects/RunStageSummary.schema.json`
- `protocol/schemas/objects/GateEvidence.schema.json`
- `protocol/schemas/objects/GateResultReport.schema.json`
- `protocol/schemas/objects/RunnerResultReport.schema.json`
- `protocol/schemas/objects/AuditEvent.schema.json`
- `protocol/schemas/objects/RunSummary.schema.json`
- `protocol/schemas/objects/RunDetail.schema.json`
- `protocol/schemas/objects/ApprovalSummary.schema.json`
- `protocol/schemas/objects/ApprovalLifecycleDetail.schema.json`
- `protocol/schemas/objects/ApprovalDetail.schema.json`
- `protocol/schemas/objects/RunCoordinationSummary.schema.json`

## Current Runtime Ownership Surfaces

- `internal/policyengine/compile.go`
- `internal/policyengine/compile_context.go`
- `internal/policyengine/evaluate_moderate.go`
- `internal/artifacts/store_approvals.go`
- `internal/brokerapi/local_api_approval_resolution_flow.go`
- `internal/brokerapi/local_api_approval_resolution_stage_signoff.go`
- `internal/brokerapi/local_api_runner_report_ops_gate_validation_evidence.go`
- `internal/brokerapi/local_api_runner_report_ops_override_bindings.go`
- `runner/src/durable-state/types.ts`
- `runner/src/durable-state/helpers.ts`

## Semantic Freeze Artifacts

- `runecontext/changes/CHG-2026-015-cae6-formal-spec-v0-tla-ci-model-checking/workflow-kernel-semantics-freeze-v0.md`
- `runecontext/changes/CHG-2026-015-cae6-formal-spec-v0-tla-ci-model-checking/transition-obligation-matrix-v0.md`
- `runecontext/changes/CHG-2026-015-cae6-formal-spec-v0-tla-ci-model-checking/effective-policy-context-hash-contract-v0.md`

## Similar Implementations

None yet.
