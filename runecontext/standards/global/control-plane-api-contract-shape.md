---
schema_version: 1
id: global/control-plane-api-contract-shape
title: Control-Plane API Contract Shape
status: active
suggested_context_bundles:
    - protocol-foundation
    - go-control-plane
---

# Control-Plane API Contract Shape

Boundary-visible control-plane APIs must keep their logical contract explicit, typed, and topology-neutral.

- Define operation-specific request and response object families under `protocol/schemas/`; do not treat ad-hoc JSON or transport-specific method envelopes as the contract source of truth.
- Keep public read models topology-neutral; do not require socket names, local usernames, daemon-private storage layouts, or host-local filesystem paths as part of boundary-visible object identity.
- Use shared typed error envelopes and stable reason-code registries for machine handling; do not rely on transport close, exit status, or scraped prose as the API error contract.
- Keep authoritative and advisory surfaces distinct in run-oriented read models: authoritative control-plane state, compiled plan identity, and approval truth must not be inferred from runner-advisory summaries.
- When local product lifecycle is operator-visible, expose it as a dedicated broker-owned typed posture surface rather than inferring attachability, restart identity, or normal-operation permission from readiness summaries, version/build metadata, transport reachability, socket presence, or bootstrap-local heuristics.
- Keep object lifecycle, projected work posture, and client presence as distinct concepts in operator-facing read models; client attachment state and local convenience memory must not become canonical session or run lifecycle truth.
- Keep plain transcript append, execution-bearing trigger submission, and session summary/watch reads as distinct typed families; do not overload one session write or stream surface to mean both transcript mutation and execution authorization.
- When transcript progress and execution progress can diverge, expose dedicated per-turn execution read models and watch families rather than encoding canonical live execution truth inside aggregate session summaries or transcript lifecycle fields.
- When APIs expose workflow execution planning, surface the immutable compiled contract explicitly rather than requiring clients to reconstruct it from workflow/process inputs or free-form status summaries.
- When surfacing policy-gated work, keep canonical decision identity and machine semantics explicit:
  - expose policy decision hashes or equivalent canonical identifiers where operator UX needs stable identity
  - keep `policy_reason_code`, `approval_trigger_code`, and system `error.code` distinct rather than overloading one status field
  - treat bound-scope summaries as explanatory UX data, not as substitutes for signed request or decision hashes
- When surfacing live activity, operator attention, or other cross-object status views, derive them from typed broker-owned read models and watch-event families rather than CLI scraping, daemon-private storage inspection, or client-local heuristic log parsing.
- When one logical API operation resolves approvals for multiple action kinds, keep the top-level resolve envelope action-generic and move action-specific inputs into typed nested detail objects instead of accumulating unrelated top-level per-action fields
- When surfacing deterministic gates, keep gate identity explicit and stable across planning and reporting: `gate_id`, `gate_kind`, `gate_version`, checkpoint identity, and relevant policy/approval references must not be collapsed into one ambiguous label.
- Stream families must use explicit typed events with stable stream identity, monotonic sequence numbers, and exactly one terminal event.
- Use opaque cursor pagination and explicit ordering semantics for list and timeline operations; do not rely on page-number conventions or undocumented default sort behavior.
- Keep transport bindings, local IPC details, and CLI ergonomics as implementations of the logical API contract rather than the source of that contract.
- Keep operator CLI commands as thin adapters over the same trusted service semantics and typed contracts; do not introduce separate CLI-only allow/deny behavior, approval resolution rules, or revocation semantics that bypass the canonical control-plane path.
