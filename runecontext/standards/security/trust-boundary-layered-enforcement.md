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
  - Broker-compiled immutable `RunPlan` contracts and trusted executor-binding resolution
  - Deterministic gate identity and ordering owned by the trusted control plane
  - Runtime isolation backends (microvm/container)
- Route allow/deny/approval-required semantics through the shared trusted policy engine boundary; component-local checks may validate structure or integrity, but must not fork authorization semantics
- Route planning semantics through the trusted control plane as well; runner-local code may schedule or checkpoint work from a compiled plan, but must not reinterpret workflow/process inputs into a second planning authority
- Preserve role-family separation at the trust boundary:
  - workspace roles remain offline
  - public egress is only via explicit gateway roles evaluated against typed destination descriptors and signed allowlist inputs
- No role may combine workspace read-write access, public egress authority, and long-lived secrets custody in one trust surface
- Treat raw URLs, transport identity, peer credentials, and local process context as insufficient authorization inputs for policy-gated egress or approval consumption
- Treat runner-advisory state as descriptive and replayable rather than authoritative; broker read models may project it, but canonical approval state, gate outcomes, and control-plane truth must still be derived from trusted persistence and typed policy/broker logic

Treat a change as risky if it weakens any of these layers, even if boundary-check still passes.
