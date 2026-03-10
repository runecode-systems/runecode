# Trust Boundary: Layered Enforcement

- CI boundary-check is a best-effort static guardrail, not a security boundary
- Authoritative enforcement lives in:
  - Broker local API auth + schema validation
  - Deterministic policy decisions
  - Runtime isolation backends (microvm/container)

Treat a change as risky if it weakens any of these layers, even if boundary-check still passes.
