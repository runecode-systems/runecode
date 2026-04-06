# Tasks

## Broker Responsibilities (MVP)

- [ ] Mediate all isolate RPC in a star topology.
- [ ] Validate schemas for every request/response.
- [ ] Use the shared protocol error envelope and stable reason codes for all rejections/failures (see `runecontext/specs/protocol-schema-bundle-v0.md`).
- [ ] Enforce size limits, rate limits, backpressure, and max in-flight requests.
- [ ] MVP on-wire encoding is JSON (schema-validated). Keep message types/fields disciplined so post-MVP protobuf encoding is straightforward.
- [ ] Broker approval handling must consume signed approval artifacts and exact hash bindings; local IPC client identity or delivery channel must never be treated as sufficient authorization on its own.
- [ ] Define the broker API as a transport-neutral logical contract first and treat local IPC as the first transport binding, not the source of truth.
- [ ] Use operation-specific typed request/response families rather than a single generic method envelope.
- [ ] Require a stable `request_id` on every request and response.
- [ ] Expand the shared protocol error-code coverage for broker/API auth, validation, not-found, limit, timeout, and approval-state failures.

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
  - getting run detail
  - listing approvals
  - getting approval detail
  - resolving approvals using typed signed decision artifacts
  - artifacts (list/head/download)
  - audit timeline (paged read)
  - audit verification reads
  - logs (stream)

Additional MVP endpoints:
- [ ] health/readiness (local-only) for daemon supervision and TUI status
- [ ] version/build info (for diagnostics and audit metadata)
- [ ] Approval endpoints return and consume typed signed `ApprovalRequest` and `ApprovalDecision` artifacts plus structured status metadata.
- [ ] Approval endpoints support listing and polling multiple pending approvals and their bound scopes without implying a whole-system pause.
- [ ] Define protocol object families for the core read models and operations, including:
  - `RunSummary`, `RunDetail`, `RunStageSummary`, `RunRoleSummary`, `RunCoordinationSummary`
  - `ApprovalSummary`, `ApprovalBoundScope`
  - `ArtifactSummary`
  - `BrokerReadiness`, `BrokerVersionInfo`
  - `RunListRequest` / `RunListResponse`
  - `RunGetRequest` / `RunGetResponse`
  - `ApprovalListRequest` / `ApprovalListResponse`
  - `ApprovalGetRequest` / `ApprovalGetResponse`
  - `ApprovalResolveRequest` / `ApprovalResolveResponse`
  - `ArtifactListRequest` / `ArtifactListResponse`
  - `ArtifactHeadRequest` / `ArtifactHeadResponse`
  - `ArtifactReadRequest`
  - `AuditTimelineRequest` / `AuditTimelineResponse`
  - `AuditVerificationGetRequest` / `AuditVerificationGetResponse`
  - `LogStreamRequest`
  - `ReadinessGetRequest` / `ReadinessGetResponse`
  - `VersionInfoGetRequest` / `VersionInfoGetResponse`
- [ ] Define `RunSummary` as the stable list-facing run model with lifecycle state, current stage, pending approval count, active profile, backend kind, assurance level, and audit posture summary.
- [ ] Define `RunDetail` as the stable drill-down run model with stage summaries, role summaries, coordination state, audit summary, artifact counts, pending approvals, and explicit separation of authoritative vs advisory state.
- [ ] Define the broker run lifecycle vocabulary explicitly and reuse it across the TUI and future runner integration.
- [ ] Define approval identity as the canonical approval-request identity shared with policy and runner state rather than a transport/session-local identifier.
- [ ] Define explicit approval status vocabulary and bound-scope metadata so blocked work and later supersession/consumption semantics remain machine-readable.
- [ ] Define artifact public read models around `ArtifactReference` without exposing daemon-private storage paths or host-local implementation details.
- [ ] Reuse or directly map audit operational-view and verification-summary contracts rather than inventing a second broker-specific audit vocabulary.
- [ ] Use opaque cursor-based pagination for paged reads and specify ordering semantics per operation.

Parallelization: can be implemented in parallel with TUI and runner development once the core request/response schemas are defined.

## Local Auth

- [ ] Bind the API to local IPC only (Unix socket / named pipe).
- [ ] MVP (Linux):
  - Use a per-user Unix domain socket under a per-user runtime directory.
  - Enforce strict permissions (dir `0700`, socket `0600`) and a safe umask.
  - Authenticate clients using OS peer credentials (e.g., `SO_PEERCRED`); require same-UID by default.
  - Fail closed if peer credentials cannot be obtained.
- [ ] Treat peer-credential checks as transport admission/authentication, not as the authorization primitive for high-risk actions.
- [ ] Do not require host-local usernames, socket paths, or OS-specific handles in boundary-visible request/response contracts except as optional diagnostics.

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
- [ ] Keep artifact download broker-mediated and streamed with uniform broker stream semantics rather than transport-specific ad hoc chunking.
- [ ] Keep MVP artifact download as full-object hash-addressed read; any future range support must be additive.

Parallelization: can be implemented in parallel with the artifact store and policy engine; it depends on stable data-class flow rules and policy decision request/response schemas.

## Streaming Semantics

- [ ] Define uniform stream-event rules across broker stream families:
  - stable `stream_id`
  - monotonic `seq`
  - exactly one terminal event
  - terminal status explicit in-band
  - failed terminal events carry the shared typed error envelope
- [ ] Define initial typed stream families for at least log streaming and artifact-byte reads.
- [ ] Keep stream semantics transport-neutral so later protobuf/gRPC work only remaps encoding.

Parallelization: can be implemented in parallel with log and artifact work once the shared stream contract is frozen.

## Acceptance Criteria

- [ ] No role requires direct network connectivity to another role.
- [ ] Invalid or oversized messages are rejected and audited.
- [ ] Local API authentication fails closed; other-user processes cannot connect to the broker API.
- [ ] The broker exposes no network-reachable API surface by default.
- [ ] CLI/TUI can operate entirely via the local API.
- [ ] Approval-related API surfaces carry typed signed approval artifacts; transport identity alone never authorizes high-risk actions.
- [ ] Run list and run detail are first-class typed broker reads rather than TUI-only derived shortcuts.
- [ ] Broker read models remain topology-neutral and do not leak daemon-private storage or transport implementation details.
- [ ] Later protobuf transport work can map 1:1 to the logical broker API without changing operation semantics, status vocabularies, or stream semantics.
