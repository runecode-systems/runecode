---
schema_version: 1
id: security/trust-boundary-interfaces
title: Trust Boundary Interfaces
status: active
suggested_context_bundles:
    - runner-boundary
    - protocol-foundation
---

# Trust Boundary Interfaces

Allowed cross-boundary interfaces:
- Broker local API (only runtime channel between trusted and untrusted)
- Message formats are schema-driven:
  - Schemas: `protocol/schemas/`
  - Fixtures: `protocol/fixtures/`
- Planning truth crosses the boundary only as broker-compiled immutable contracts such as `RunPlan`; workflow/process definitions may inform trusted compilation, but the runner does not become the planning authority
- Runner-originated progress, gate lifecycle, and evidence updates cross the boundary only through typed report families such as `RunnerCheckpointReport`, `RunnerResultReport`, `GateCheckpointReport`, `GateResultReport`, and `GateEvidence`
- Runner-local durable state, approval waits, and internal runtime checkpoints remain advisory mechanics only; they must not cross the boundary as a second source of run, approval, or lifecycle truth

Broker local API requirements:
- Local peer authentication fails closed when peer credentials are unavailable or cannot be verified
- Boundary-visible requests and responses use typed schema families rather than ad-hoc JSON payloads
- Broker-mediated streams use explicit typed event families rather than transport-close semantics as the contract
- Trusted services own executor-binding resolution, gate identity, and plan ordering; the runner may validate shape for self-protection, but must not mint alternate planning authority or alternate gate ordering semantics
- Boundary-visible run state may include runner-advisory summaries, but authoritative status, approval truth, and compiled planning truth remain in the trusted domain
- Restart and resume must reconcile runner-local persistence against broker-canonical plan and approval bindings; stale or superseded plan-bound state fails closed rather than being merged heuristically

Prohibited bypasses:
- Runner receives secrets via env vars, files, or CLI args
- Ad-hoc JSON outside schema validation
- Runner imports/references trusted paths (`cmd/`, `internal/`)
- Direct socket/file access to trusted daemons bypassing the broker
- Runner-local synthesis or mutation of authoritative `RunPlan`, executor registry, approval truth, or deterministic gate identity/order
- Framework-local thread/checkpoint state or runtime-seam persistence becoming the effective public contract or authoritative recovery source across the trust boundary
