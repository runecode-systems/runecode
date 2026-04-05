# Tasks

## Broker Responsibilities (MVP)

- [ ] Mediate all isolate RPC in a star topology.
- [ ] Validate schemas for every request/response.
- [ ] Use the shared protocol error envelope and stable reason codes for all rejections/failures (see `runecontext/specs/protocol-schema-bundle-v0.md`).
- [ ] Enforce size limits, rate limits, backpressure, and max in-flight requests.
- [ ] MVP on-wire encoding is JSON (schema-validated). Keep message types/fields disciplined so post-MVP protobuf encoding is straightforward.
- [ ] Broker approval handling must consume signed approval artifacts and exact hash bindings; local IPC client identity or delivery channel must never be treated as sufficient authorization on its own.

MVP default limits (tunable via explicit config; changes are audited):
- [ ] Max message size: 1 MiB (hard reject if exceeded).
- [ ] Max structural complexity: depth <= 64; array length <= 10_000; object properties <= 1_000.
- [ ] Max in-flight requests per client: 64.
- [ ] Max in-flight per role<->role lane: 32.
- [ ] Default request deadline: 30s (role-specific overrides allowed; long-running ops must stream progress).
- [ ] Streaming limits: chunk <= 64 KiB; idle timeout 15s; total streamed bytes per response 16 MiB (unless explicitly increased by manifest/policy).

Parallelization: can be implemented in parallel with policy engine and protocol schema work once the message envelopes and error taxonomy are stable.

## Local Client API

- [ ] Provide a local-only API surface for:
  - listing runs
  - approvals (request/approve/deny)
  - artifacts (list/head/download)
  - audit timeline (paged read)
  - logs (stream)

Additional MVP endpoints:
- [ ] health/readiness (local-only) for daemon supervision and TUI status
- [ ] version/build info (for diagnostics and audit metadata)
- [ ] Approval endpoints return and consume typed signed `ApprovalRequest` and `ApprovalDecision` artifacts plus structured status metadata.
- [ ] Approval endpoints support listing and polling multiple pending approvals and their bound scopes without implying a whole-system pause.

Parallelization: can be implemented in parallel with TUI and runner development once the core request/response schemas are defined.

## Local Auth

- [ ] Bind the API to local IPC only (Unix socket / named pipe).
- [ ] MVP (Linux):
  - Use a per-user Unix domain socket under a per-user runtime directory.
  - Enforce strict permissions (dir `0700`, socket `0600`) and a safe umask.
  - Authenticate clients using OS peer credentials (e.g., `SO_PEERCRED`); require same-UID by default.
  - Fail closed if peer credentials cannot be obtained.

Supported MVP deployment shape:
- [ ] Single-user local machine; clients connect directly to the broker's local IPC endpoint (no UDS proxying/forwarding).
- [ ] If the runtime environment interferes with peer credential propagation, treat it as unsupported for MVP unless an explicit alternative auth mechanism is enabled.
- [ ] Non-MVP note: on platforms without peer credentials, require an explicit local-only auth mechanism (e.g., short-lived token stored in a `0600` file) rather than silently disabling auth.

Parallelization: can be implemented in parallel with TUI and CLI work; depends on a stable local API transport contract.

### Follow-On Local Transport Spec

- [ ] Post-MVP protobuf/gRPC transport work now lives in `runecontext/changes/CHG-2026-021-8d6d-local-ipc-protobuf-transport-v0/`.

## Artifact Routing Integration

- [ ] Implement broker-mediated artifact routing using the artifact store.
- [ ] Enforce the data-class flow matrix.
- [ ] Ensure gateway roles (e.g., `model-gateway`) can fetch artifact bytes by hash only via the broker/artifact API (never via workspace mounts).
- [ ] Enforce role+manifest-based allowlists when serving artifact bytes (fail closed on disallowed data classes).

Parallelization: can be implemented in parallel with the artifact store and policy engine; it depends on stable data-class flow rules and policy decision request/response schemas.

## Acceptance Criteria

- [ ] No role requires direct network connectivity to another role.
- [ ] Invalid or oversized messages are rejected and audited.
- [ ] Local API authentication fails closed; other-user processes cannot connect to the broker API.
- [ ] The broker exposes no network-reachable API surface by default.
- [ ] CLI/TUI can operate entirely via the local API.
- [ ] Approval-related API surfaces carry typed signed approval artifacts; transport identity alone never authorizes high-risk actions.
