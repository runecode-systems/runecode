# References

## Product Context

- **Mission:** `runecontext/project/mission.md`
- **Tech stack:** `runecontext/project/tech-stack.md`
- **Roadmap:** `runecontext/project/roadmap.md`
- **Trust boundaries:** `docs/trust-boundaries.md`

## Related Specs

- `runecontext/changes/CHG-2026-007-2315-policy-engine-v0/`
- `runecontext/changes/CHG-2026-008-62e1-broker-local-api-v0/`
- `runecontext/changes/CHG-2026-009-1672-launcher-microvm-backend-v0/`
- `runecontext/changes/CHG-2026-012-f1ef-workflow-runner-workspace-roles-deterministic-gates-v0/`
- `runecontext/changes/CHG-2026-013-d2c9-minimal-tui-v0/`
- `runecontext/changes/CHG-2026-027-71ed-workflow-concurrency-v0/`
- `runecontext/changes/CHG-2026-039-7c2e-interactive-control-plane-ux-contracts-v0/`
- `runecontext/changes/CHG-2026-041-4d8a-approval-review-detail-models-v0/`

## Key Implementation References

- `internal/launcherdaemon/service.go`
- `internal/launcherdaemon/service_backend_posture.go`
- `internal/launcherdaemon/qemu_controller_linux.go`
- `internal/brokerapi/policy_runtime.go`
- `internal/brokerapi/local_api_approval_resolution_flow.go`
- `internal/brokerapi/local_api_approval_detail_support.go`
- `internal/brokerapi/local_api_run_detail_state_authoritative_ops.go`
- `internal/policyengine/evaluate_boundaries_backend.go`
- `internal/policyengine/action_builders.go`
- `protocol/schemas/objects/ApprovalBoundScope.schema.json`
- `protocol/schemas/objects/ApprovalDetail.schema.json`
- `protocol/schemas/objects/RunDetail.schema.json`
- `cmd/runecode-tui/route_approvals.go`
- `cmd/runecode-tui/broker_client.go`

## Similar Implementations

None yet.
