# Traceability Map (Formal Spec v0 / CHG-2026-015)

This artifact maps modeled workflow-kernel concepts and invariants to their protocol contracts, trusted runtime owners, runner durability surfaces, and governing changes/standards so TLC failures are actionable without code archaeology.

## CI/TLA Invariant Routing Table

| Invariant ID (TLA/CI) | Modeled concept | Protocol schema families | Trusted Go ownership modules | Runner durability modules | Related changes | Standards anchors |
| --- | --- | --- | --- | --- | --- | --- |
| `INV-APPROVAL-SCOPE-ISOLATION` | An approval bound for one run/action cannot authorize another run/action. | `ActionRequest`, `ApprovalRequest`, `ApprovalDecision`, `ApprovalDetail`, `ApprovalSummary` | `internal/artifacts/store_approvals.go`, `internal/brokerapi/local_api_approval_resolution_flow.go` | `runner/src/durable-state/types.ts`, `runner/src/durable-state/helpers.ts` | `CHG-2026-015`, `CHG-2026-007`, `CHG-2026-008`, `CHG-2026-033` | `approval-binding-and-verifier-identity.md`, `policy-evaluation-foundations.md`, `trust-boundaries.md` |
| `INV-APPROVAL-SINGLE-CONSUME` | One approval can be consumed at most once; accepted-vs-consumed remains distinct. | `ApprovalRequest`, `ApprovalDecision`, `ApprovalLifecycleDetail`, `ApprovalDetail` | `internal/artifacts/store_approvals.go`, `internal/brokerapi/local_api_approval_resolution_flow.go` | `runner/src/durable-state/types.ts` | `CHG-2026-015`, `CHG-2026-033` | `approval-binding-and-verifier-identity.md`, `runner-durable-state-and-replay.md` |
| `INV-STAGE-SIGNOFF-SUPERSESSION` | Stale/superseded stage-sign-off approvals cannot be consumed. | canonical `StageSummary` family (owned by this change), `ActionPayloadStageSummarySignOff`, `RunPlan`, `RunStageSummary` (derived only) | `internal/brokerapi/local_api_approval_resolution_stage_signoff.go`, `internal/brokerapi/local_api_approval_resolution_flow.go` | `runner/src/durable-state/types.ts` | `CHG-2026-015`, `CHG-2026-012`, `CHG-2026-033` | `protocol-schema-invariants.md`, `protocol-canonicalization-profile.md`, `trust-boundaries.md` |
| `INV-BROKER-AUTHORITY-OVER-RUNNER` | Broker-authoritative truth cannot be replaced by runner-advisory durable state. | `RunnerResultReport`, `RunDetail`, `RunSummary`, `RunCoordinationSummary` | `internal/brokerapi/local_api_runner_report_ops_gate_validation_evidence.go`, `internal/brokerapi/local_api_approval_resolution_flow.go` | `runner/src/durable-state/types.ts`, `runner/src/durable-state/helpers.ts` | `CHG-2026-015`, `CHG-2026-008`, `CHG-2026-033` | `runner-durable-state-and-replay.md`, `trusted-local-artifact-persistence.md`, `trust-boundaries.md` |
| `INV-GATE-OVERRIDE-BINDING` | Gate override continuation must bind exact failed gate result + attempt + policy context. | `ActionPayloadGateOverride`, `GateResultReport`, `GateEvidence`, `RunnerResultReport`, `RunPlan` | `internal/brokerapi/local_api_runner_report_ops_gate_validation_evidence.go`, `internal/policyengine/compile.go`, `internal/policyengine/evaluate_moderate.go` | `runner/src/durable-state/types.ts` | `CHG-2026-015`, `CHG-2026-035`, `CHG-2026-007`, `CHG-2026-008` | `policy-evaluation-foundations.md`, `protocol-schema-invariants.md`, `protocol-registry-discipline.md` |
| `INV-PUBLIC-LIFECYCLE-NO-SHADOW-ENUM` | Partial blocking/coordination state must not mint a second public lifecycle vocabulary. | `RunSummary`, `RunDetail`, `RunCoordinationSummary` | `internal/brokerapi/local_api_approval_resolution_flow.go` | `runner/src/durable-state/types.ts` | `CHG-2026-015`, `CHG-2026-012`, `CHG-2026-008` | `control-plane-api-contract-shape.md`, `trust-boundaries.md` |
| `INV-AUDIT-OBLIGATION-ON-AUTH-TRANSITION` | Authoritative transitions in the v0 matrix cannot advance without required audit facts. | `AuditEvent`, `ApprovalLifecycleDetail`, `RunDetail`, `GateEvidence` | `internal/artifacts/store_approvals.go`, `internal/brokerapi/local_api_approval_resolution_flow.go`, `internal/brokerapi/local_api_runner_report_ops_gate_validation_evidence.go` | `runner/src/durable-state/helpers.ts` | `CHG-2026-015`, `CHG-2026-003`, `CHG-2026-004` | `audit-verification-scope-and-evidence-binding.md`, `trusted-local-artifact-persistence.md` |

## Failure Triage (Fast Path)

1. Match failing TLC invariant name to **Invariant ID** above.
2. Check corresponding schema families first (`protocol/schemas/objects/*` and `protocol/schemas/manifest.json`) to confirm contract intent.
3. Check listed trusted modules for authoritative state transition logic.
4. Check listed runner durability modules only for advisory/replay assumptions (never as authority source).
5. Validate against linked standards and related change contracts before adjusting model bounds or weakening invariants.

## Maintenance Rule

When any kernel invariant, schema family, or owning module changes, update this file in the same PR as:
- TLA invariant/config updates,
- protocol manifest/schema updates,
- trusted/runtime ownership refactors.

This keeps CI model-check failures directly traceable to owning contracts.
