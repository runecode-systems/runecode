# Tasks

## Broker Responsibilities (MVP)

- [ ] Mediate all isolate RPC in a star topology.
- [x] Validate schemas for every request/response.
- [x] Use the shared protocol error envelope and stable reason codes for all rejections/failures (see `runecontext/specs/protocol-schema-bundle-v0.md`).
- [x] Enforce size limits, rate limits, backpressure, and max in-flight requests.
- [x] MVP on-wire encoding is JSON (schema-validated). Keep message types/fields disciplined so post-MVP protobuf encoding is straightforward.
- [x] Broker approval handling must consume signed approval artifacts and exact hash bindings; local IPC client identity or delivery channel must never be treated as sufficient authorization on its own.
- [x] Define the broker API as a transport-neutral logical contract first and treat local IPC as the first transport binding, not the source of truth.
- [x] Use operation-specific typed request/response families rather than a single generic method envelope.
- [x] Require a stable `request_id` on every request and response.
- [x] Expand the shared protocol error-code coverage for broker/API auth, validation, not-found, limit, timeout, and approval-state failures.

MVP default limits (tunable via explicit config; changes are audited):
- [x] Max message size: 1 MiB (hard reject if exceeded).
- [x] Max structural complexity: depth <= 64; array length <= 10_000; object properties <= 1_000.
- [x] Max in-flight requests per client: 64.
- [x] Max in-flight per role<->role lane: 32.
- [x] Default request deadline: 30s (role-specific overrides allowed; long-running ops must stream progress).
- [x] Streaming limits: chunk <= 64 KiB; idle timeout 15s; total streamed bytes per response 16 MiB (unless explicitly increased by manifest/policy).

Parallelization: can be implemented in parallel with policy engine and protocol schema work once the message envelopes and error taxonomy are stable.

## Local Client API

- [x] Provide a local-only API surface for:
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
- [x] health/readiness (local-only) for daemon supervision and TUI status
- [x] version/build info (for diagnostics and audit metadata)
- [x] Approval endpoints return and consume typed signed `ApprovalRequest` and `ApprovalDecision` artifacts plus structured status metadata.
- [x] Approval endpoints support listing and polling multiple pending approvals and their bound scopes without implying a whole-system pause.
- [ ] Keep broker approval inspection and resolution aligned with the policy split between exact-action approvals and stage sign-off approvals.
- [x] Define protocol object families for the core read models and operations, including:
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
- [x] Define `RunSummary` as the stable list-facing run model with lifecycle state, current stage, pending approval count, active profile, backend kind, assurance level, and audit posture summary.
- [x] Define `RunDetail` as the stable drill-down run model with stage summaries, role summaries, coordination state, audit summary, artifact counts, pending approvals, and explicit separation of authoritative vs advisory state.
- [ ] Tighten `RunSummary` / `RunDetail` posture semantics so:
  - `backend_kind` identifies the selected backend class and stays topology-neutral
  - `assurance_level` refers only to runtime isolation assurance
  - provisioning/binding posture remains separate from both `backend_kind` and audit posture
  - audit posture remains represented through explicit audit summary fields rather than overloaded into runtime assurance
- [ ] Keep partial blocking, waits, and coordination detail in `RunDetail`, `RunStageSummary`, `RunRoleSummary`, and `RunCoordinationSummary` rather than expanding the public lifecycle enum for every coordination mode.
- [x] Define the broker run lifecycle vocabulary explicitly and reuse it across the TUI and future runner integration.
- [x] Define approval identity as the canonical approval-request identity shared with policy and runner state rather than a transport/session-local identifier.
- [x] Define explicit approval status vocabulary and bound-scope metadata so blocked work and later supersession/consumption semantics remain machine-readable.
- [ ] Treat `ApprovalBoundScope` as normalized UX metadata and keep signed request identity, stage-summary hash, and compiled policy-context hash as the authoritative bindings.
- [x] Define artifact public read models around `ArtifactReference` without exposing daemon-private storage paths or host-local implementation details.
- [x] Reuse or directly map audit operational-view and verification-summary contracts rather than inventing a second broker-specific audit vocabulary.
- [x] Use opaque cursor-based pagination for paged reads and specify ordering semantics per operation.
- [ ] Keep authoritative backend/runtime facts launcher/broker-derived and avoid deriving runtime isolation assurance from audit verification posture or runner-local status alone.
- [ ] Define typed runner->broker workflow checkpoint/result write families when runner orchestration integration lands, keeping them operation-specific and broker-validated.

Parallelization: can be implemented in parallel with TUI and runner development once the core request/response schemas are defined.

## Local Auth

- [x] Bind the API to local IPC only (Unix socket / named pipe).
- [ ] MVP (Linux):
  - Use a per-user Unix domain socket under a per-user runtime directory.
  - Enforce strict permissions (dir `0700`, socket `0600`) and a safe umask.
  - Authenticate clients using OS peer credentials (e.g., `SO_PEERCRED`); require same-UID by default.
  - Fail closed if peer credentials cannot be obtained.
- [x] Treat peer-credential checks as transport admission/authentication, not as the authorization primitive for high-risk actions.
- [x] Do not require host-local usernames, socket paths, or OS-specific handles in boundary-visible request/response contracts except as optional diagnostics.

Supported MVP deployment shape:
- [x] Single-user local machine; clients connect directly to the broker's local IPC endpoint (no UDS proxying/forwarding).
- [x] If the runtime environment interferes with peer credential propagation, treat it as unsupported for MVP unless an explicit alternative auth mechanism is enabled.
- [x] Non-MVP note: on platforms without peer credentials, require an explicit local-only auth mechanism (e.g., short-lived token stored in a `0600` file) rather than silently disabling auth.

Parallelization: can be implemented in parallel with TUI and CLI work; depends on a stable local API transport contract.

### Follow-On Local Transport Spec

- [x] Post-MVP protobuf/gRPC transport work now lives in `runecontext/changes/CHG-2026-021-8d6d-local-ipc-protobuf-transport-v0/`.

## Artifact Routing Integration

- [x] Implement broker-mediated artifact routing using the artifact store.
- [x] Enforce the data-class flow matrix.
- [ ] Ensure gateway roles (e.g., `model-gateway`) can fetch artifact bytes by hash only via the broker/artifact API (never via workspace mounts).
- [x] Enforce role+manifest-based allowlists when serving artifact bytes (fail closed on disallowed data classes).
- [ ] Keep broker policy-facing request/response surfaces ready to carry canonical `ActionRequest` / `PolicyDecision` identities without transport-specific reshaping.
- [x] Keep artifact download broker-mediated and streamed with uniform broker stream semantics rather than transport-specific ad hoc chunking.
- [x] Keep MVP artifact download as full-object hash-addressed read; any future range support must be additive.

Parallelization: can be implemented in parallel with the artifact store and policy engine; it depends on stable data-class flow rules and policy decision request/response schemas.

## Streaming Semantics

- [x] Define uniform stream-event rules across broker stream families:
  - stable `stream_id`
  - monotonic `seq`
  - exactly one terminal event
  - terminal status explicit in-band
  - failed terminal events carry the shared typed error envelope
- [x] Define initial typed stream families for at least log streaming and artifact-byte reads.
- [x] Keep stream semantics transport-neutral so later protobuf/gRPC work only remaps encoding.

Parallelization: can be implemented in parallel with log and artifact work once the shared stream contract is frozen.

## Acceptance Criteria

- [ ] No role requires direct network connectivity to another role.
- [ ] Invalid or oversized messages are rejected and audited.
- [x] Local API authentication fails closed; other-user processes cannot connect to the broker API.
- [x] The broker exposes no network-reachable API surface by default.
- [ ] CLI/TUI can operate entirely via the local API.
- [x] Approval-related API surfaces carry typed signed approval artifacts; transport identity alone never authorizes high-risk actions.
- [x] Run list and run detail are first-class typed broker reads rather than TUI-only derived shortcuts.
- [x] Broker read models remain topology-neutral and do not leak daemon-private storage or transport implementation details.
- [ ] Broker run read models keep backend kind, runtime isolation assurance, provisioning/binding posture, and audit posture as distinct operator-facing concepts.
- [x] Later protobuf transport work can map 1:1 to the logical broker API without changing operation semantics, status vocabularies, or stream semantics.
