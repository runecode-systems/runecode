# Broker + Local API v0

User-visible outcome: isolates and clients communicate through a brokered, schema-validated, rate-limited local API; no isolate-to-isolate direct networking is required.

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-08-1039-broker-local-api-v0/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

## Task 2: Broker Responsibilities (MVP)

- Mediate all isolate RPC in a star topology.
- Validate schemas for every request/response.
- Enforce size limits, rate limits, backpressure, and max in-flight requests.
- MVP on-wire encoding is JSON (schema-validated). Keep message types/fields disciplined so post-MVP protobuf encoding is straightforward.

## Task 3: Local Client API

- Provide a local-only API surface for:
  - listing runs
  - approvals (request/approve/deny)
  - artifacts (list/head/download)
  - audit timeline (paged read)
  - logs (stream)

## Task 4: Local Auth

- Bind the API to local IPC only (Unix socket / named pipe).
- MVP (Linux):
  - Use a per-user Unix domain socket under a per-user runtime directory.
  - Enforce strict permissions (dir `0700`, socket `0600`) and a safe umask.
  - Authenticate clients using OS peer credentials (e.g., `SO_PEERCRED`); require same-UID by default.
  - Fail closed if peer credentials cannot be obtained.
- Non-MVP note: on platforms without peer credentials, require an explicit local-only auth mechanism (e.g., short-lived token stored in a `0600` file) rather than silently disabling auth.

### Post-MVP note: protobuf and optional gRPC

- Prefer protobuf message encoding over the existing local IPC transports (UDS / named pipes).
- gRPC is optional; if adopted for ergonomics:
  - do not use TCP by default
  - prefer gRPC over Unix domain sockets (Unix) or OS-native local IPC (e.g., named pipes on Windows) where supported
  - if TCP loopback is used for compatibility, require mTLS or a strong short-lived local token
  - binding safety is a security requirement: never bind privileged APIs to non-loopback interfaces

## Task 5: Artifact Routing Integration

- Implement broker-mediated artifact routing using the artifact store.
- Enforce the data-class flow matrix.

## Acceptance Criteria

- No role requires direct network connectivity to another role.
- Invalid or oversized messages are rejected and audited.
- Local API authentication fails closed; other-user processes cannot connect to the broker API.
- The broker exposes no network-reachable API surface by default.
- CLI/TUI can operate entirely via the local API.
