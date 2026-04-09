---
schema_version: 1
id: security/trust-boundary-layered-enforcement
title: 'Trust Boundary: Layered Enforcement'
status: active
suggested_context_bundles:
    - runner-boundary
---

# Trust Boundary: Layered Enforcement

- CI boundary-check is a best-effort static guardrail, not a security boundary
- Authoritative enforcement lives in:
  - Broker local API auth, request limits, and schema validation
  - Typed broker error handling and explicit stream termination semantics
  - Deterministic policy decisions
  - Runtime isolation backends (microvm/container)
- Route allow/deny/approval-required semantics through the shared trusted policy engine boundary; component-local checks may validate structure or integrity, but must not fork authorization semantics
- Preserve role-family separation at the trust boundary:
  - workspace roles remain offline
  - public egress is only via explicit gateway roles evaluated against typed destination descriptors and signed allowlist inputs
- No role may combine workspace read-write access, public egress authority, and long-lived secrets custody in one trust surface
- Treat raw URLs, transport identity, peer credentials, and local process context as insufficient authorization inputs for policy-gated egress or approval consumption

Treat a change as risky if it weakens any of these layers, even if boundary-check still passes.
