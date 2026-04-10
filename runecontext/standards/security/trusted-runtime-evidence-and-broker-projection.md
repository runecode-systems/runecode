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
- Broker-owned runtime audit events must reference persisted evidence digests and canonical runtime identity fields rather than launcher-private host paths, hypervisor argv, transport allocation details, or scraped stderr text
- The broker remains the owner of operator-facing runtime audit emission; launcher/runtime services supply typed evidence and lifecycle updates but must not become a second public audit authority
- Secure-session summaries and isolate binding state should remain transport-neutral and topology-neutral; durable identity is the per-session isolate key/binding context, not the host transport endpoint
- Backend implementations may expose implementation-specific evidence details such as hypervisor provenance or acceleration kind only in detailed evidence surfaces; shared operator read models should remain backend-class-oriented and stable across Linux, macOS, Windows, and future runtime implementations
- Tests should cover restart-time reconstruction, evidence/lifecycle consistency, deduplicated audit emission, and fail-closed handling when runtime evidence is invalid or incomplete
