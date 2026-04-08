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

Treat a change as risky if it weakens any of these layers, even if boundary-check still passes.
