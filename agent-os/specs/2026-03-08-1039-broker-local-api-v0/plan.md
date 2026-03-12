# Broker + Local API v0

User-visible outcome: isolates and clients communicate through a brokered, schema-validated, rate-limited local API; no isolate-to-isolate direct networking is required.

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-08-1039-broker-local-api-v0/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

Parallelization: docs-only; safe to do anytime.

## Task 2: Broker Responsibilities (MVP)

- Mediate all isolate RPC in a star topology.
- Validate schemas for every request/response.
- Use the shared protocol error envelope and stable reason codes for all rejections/failures (see `agent-os/specs/2026-03-08-1039-protocol-schemas-v0/`).
- Enforce size limits, rate limits, backpressure, and max in-flight requests.
- MVP on-wire encoding is JSON (schema-validated). Keep message types/fields disciplined so post-MVP protobuf encoding is straightforward.

MVP default limits (tunable via explicit config; changes are audited):
- Max message size: 1 MiB (hard reject if exceeded).
- Max structural complexity: depth <= 64; array length <= 10_000; object properties <= 1_000.
- Max in-flight requests per client: 64.
- Max in-flight per role<->role lane: 32.
- Default request deadline: 30s (role-specific overrides allowed; long-running ops must stream progress).
- Streaming limits: chunk <= 64 KiB; idle timeout 15s; total streamed bytes per response 16 MiB (unless explicitly increased by manifest/policy).

Parallelization: can be implemented in parallel with policy engine and protocol schema work once the message envelopes and error taxonomy are stable.

## Task 3: Local Client API

- Provide a local-only API surface for:
  - listing runs
  - approvals (request/approve/deny)
  - artifacts (list/head/download)
  - audit timeline (paged read)
  - logs (stream)

Additional MVP endpoints:
- health/readiness (local-only) for daemon supervision and TUI status
- version/build info (for diagnostics and audit metadata)

Parallelization: can be implemented in parallel with TUI and runner development once the core request/response schemas are defined.

## Task 4: Local Auth

- Bind the API to local IPC only (Unix socket / named pipe).
- MVP (Linux):
  - Use a per-user Unix domain socket under a per-user runtime directory.
  - Enforce strict permissions (dir `0700`, socket `0600`) and a safe umask.
  - Authenticate clients using OS peer credentials (e.g., `SO_PEERCRED`); require same-UID by default.
  - Fail closed if peer credentials cannot be obtained.

Supported MVP deployment shape:
- Single-user local machine; clients connect directly to the broker’s local IPC endpoint (no UDS proxying/forwarding).
- If the runtime environment interferes with peer credential propagation, treat it as unsupported for MVP unless an explicit alternative auth mechanism is enabled.
- Non-MVP note: on platforms without peer credentials, require an explicit local-only auth mechanism (e.g., short-lived token stored in a `0600` file) rather than silently disabling auth.

Parallelization: can be implemented in parallel with TUI and CLI work; depends on a stable local API transport contract.

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
- Ensure gateway roles (e.g., `model-gateway`) can fetch artifact bytes by hash only via the broker/artifact API (never via workspace mounts).
- Enforce role+manifest-based allowlists when serving artifact bytes (fail closed on disallowed data classes).

Parallelization: can be implemented in parallel with the artifact store and policy engine; it depends on stable data-class flow rules and policy decision request/response schemas.

## Acceptance Criteria

- No role requires direct network connectivity to another role.
- Invalid or oversized messages are rejected and audited.
- Local API authentication fails closed; other-user processes cannot connect to the broker API.
- The broker exposes no network-reachable API surface by default.
- CLI/TUI can operate entirely via the local API.
