---
schema_version: 1
id: security/trusted-runtime-evidence-and-broker-projection
title: Trusted Runtime Evidence And Broker Projection
status: active
suggested_context_bundles:
    - go-control-plane
    - protocol-foundation
---

# Trusted Runtime Evidence And Broker Projection

When trusted runtime backends report launcher- or isolate-derived state into trusted broker/operator surfaces:

- Treat launcher-produced launch, session, hardening, and terminal records as immutable runtime evidence first and operator-facing read-model inputs second
- Persist runtime evidence before projecting it into broker-visible authoritative state; do not rely on transient launcher streams or in-memory broker caches as the source of truth
- Keep backend lifecycle progression separate from immutable launch evidence; lifecycle updates may refine projected state but must not mutate the meaning of prior evidence records
- Derive broker `RunSummary`, `RunDetail`, and related authoritative runtime projections from persisted runtime evidence plus persisted lifecycle state rather than from runner-only status or audit-only heuristics
- Keep operator-visible runtime posture split into separate axes such as `backend_kind`, `isolation_assurance_level`, `provisioning_posture`, and audit verification posture; do not collapse these into one overloaded status field
- Keep instance-scoped backend posture read models explicit and broker-projected: expose the active runtime `instance_id`, selected and preferred backend kinds, reduced-assurance state, pending approval state, and linked policy/approval references through typed control-plane surfaces rather than client-local toggles or launcher-private side channels
- Broker-owned runtime audit families should cover both pre-session launch outcomes and session lifecycle outcomes; `runtime_launch_admission` and `runtime_launch_denied` are first-class operator-facing evidence surfaces, not implied side effects of later session events
- Broker-owned runtime audit events must reference persisted evidence digests and canonical runtime identity fields rather than launcher-private host paths, hypervisor argv, transport allocation details, or scraped stderr text
- Runtime audit `operation_id` and audit-emission dedupe identity must be derived from the evidence that defines the event. Launch events should stay launch-evidence scoped, while session lifecycle events should include the launch, hardening, and session evidence needed to avoid collisions or drift across retries and later projections
- Optional runtime audit payload fields should be omitted when empty rather than serialized as empty strings that violate typed schema semantics or create ambiguous operator meaning
- When broker applies an approval-gated runtime posture change, emit a broker-owned posture-application audit event that references the current runtime `instance_id` together with the persisted action, policy-decision, and approval digests that authorized the change
- The broker remains the owner of operator-facing runtime audit emission; launcher/runtime services supply typed evidence and lifecycle updates but must not become a second public audit authority
- Secure-session summaries and isolate binding state should remain transport-neutral and topology-neutral; durable identity is the per-session isolate key/binding context, not the host transport endpoint
- Backend implementations may expose implementation-specific evidence details such as hypervisor provenance or acceleration kind only in detailed evidence surfaces; shared operator read models should remain backend-class-oriented and stable across Linux, macOS, Windows, and future runtime implementations
- Tests should cover restart-time reconstruction, evidence/lifecycle consistency, deduplicated audit emission, and fail-closed handling when runtime evidence is invalid or incomplete
