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

Broker local API requirements:
- Local peer authentication fails closed when peer credentials are unavailable or cannot be verified
- Boundary-visible requests and responses use typed schema families rather than ad-hoc JSON payloads
- Broker-mediated streams use explicit typed event families rather than transport-close semantics as the contract

Prohibited bypasses:
- Runner receives secrets via env vars, files, or CLI args
- Ad-hoc JSON outside schema validation
- Runner imports/references trusted paths (`cmd/`, `internal/`)
- Direct socket/file access to trusted daemons bypassing the broker
