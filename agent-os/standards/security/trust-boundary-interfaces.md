# Trust Boundary Interfaces

Allowed cross-boundary interfaces:
- Broker local API (only runtime channel between trusted and untrusted)
- Message formats are schema-driven:
  - Schemas: `protocol/schemas/`
  - Fixtures: `protocol/fixtures/`

Prohibited bypasses:
- Runner receives secrets via env vars, files, or CLI args
- Ad-hoc JSON outside schema validation
- Runner imports/references trusted paths (`cmd/`, `internal/`)
- Direct socket/file access to trusted daemons bypassing the broker
