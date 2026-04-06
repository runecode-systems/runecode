# Design

## Overview
Implement the brokered local API contract that mediates all isolate RPC, local client access, and broker-managed artifact routing.

## Key Decisions
- Star topology only; no direct isolate-to-isolate communication.
- Schema validation at boundaries; rate limiting and backpressure are mandatory.
- Broker enforces concrete default limits (message size/complexity/in-flight/streaming) with audited overrides.
- The local API is per-user IPC with strict filesystem permissions; authentication fails closed when OS peer credentials are unavailable.
- Approval delivery channels are not authoritative; the broker transports typed signed approval objects and exact hash bindings rather than trusting transport or UI channel identity.
- Audit timeline and verification reads consume derived auditd-owned views and machine-readable verification findings rather than treating the broker, TUI, or artifact store as alternate audit sources of truth.
- MVP remains local-IPC-first, but boundary-visible API contracts must stay topology-neutral so future remote UI or messaging bridges terminate into the same signed approval and approval-authority model.
- Errors use a shared typed envelope and stable reason codes (no ad-hoc error shapes).
- MVP uses JSON on-wire; later transport migration is specified separately so this spec stays focused on the MVP broker/API contract.
- The broker API is defined first as a logical typed contract; transport bindings are an implementation of that contract, not the source of truth.
- MVP uses operation-specific request/response object families rather than a single generic method envelope so request validation, fixture coverage, alternate transport mapping, and future capability review remain explicit.
- Broker read models are public operator-facing contracts. Internal Go structs and daemon-private storage layouts are implementation details and must be translated into topology-neutral API shapes.
- Run inspection is a first-class capability. `list runs` and `get run detail` are foundational control-plane reads, not TUI-only conveniences.
- Approval identity is canonical and hash-bound. Broker-visible approval lifecycle state must reuse the same approval-request identity across policy, runner, TUI, and later transport changes.
- Streaming semantics are uniform across stream families: `stream_id`, monotonic `seq`, exactly one terminal event, and terminal failure carried as a typed protocol error.
- Broker-visible state separates authoritative broker-derived or trusted state from runner advisory state so operator UX can consume real posture without promoting untrusted internals into trusted truth.

## Logical API Model

### Contract Shape
- The logical broker API uses typed request/response families under `protocol/schemas/`.
- Every request and response includes a stable `request_id`.
- Requests and responses are schema-versioned protocol objects, not ad hoc JSON.
- Request families model operations explicitly rather than tunneling arbitrary submethods inside a generic envelope.
- Public object-family names stay topology-neutral and avoid transport- or host-specific naming.

### Core Read Models
The following logical read models are part of the long-lived API foundation for this change:
- `RunSummary`: stable list-facing operator summary for one run.
- `RunDetail`: stable drill-down view for one run.
- `RunStageSummary`: stable per-stage drill-down summary for `RunDetail`.
- `RunRoleSummary`: stable per-role/operator summary for `RunDetail`.
- `RunCoordinationSummary`: explicit read model for waits, locks, conflicts, and later shared-workspace posture.
- `ApprovalSummary`: list/detail-facing summary for one approval lifecycle object.
- `ApprovalBoundScope`: normalized scope metadata for what exact work an approval can unblock or consume.
- `ArtifactSummary`: public broker-facing artifact metadata view built around `ArtifactReference` plus safe operational metadata.
- `AuditTimelinePage`: paged response carrying derived audit operational views.
- `BrokerReadiness`: broker-facing readiness read model, including dependent subsystem posture required for operator UX and local supervision.
- `BrokerVersionInfo`: stable operator diagnostic and compatibility view for build, bundle, and encoding posture.

### Required Request/Response Families
The change must define operation-specific object families for at least:
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

These object families are the semantic API surface that later protobuf/gRPC transport work maps 1:1.

## Run Model

### Run Summary
`RunSummary` is the stable list-facing run model. It should include at least:
- `run_id`
- `workspace_id`
- workflow identity such as `workflow_kind` and/or `workflow_definition_hash`
- `created_at`
- `started_at`
- `updated_at`
- `finished_at`
- `lifecycle_state`
- `current_stage_id`
- `pending_approval_count`
- `approval_profile`
- `backend_kind`
- `assurance_level`
- `blocking_reason_code`
- audit posture summary fields such as `audit_integrity_status`, `audit_anchoring_status`, and `audit_currently_degraded`

Recommended lifecycle vocabulary:
- `pending`
- `starting`
- `active`
- `blocked`
- `recovering`
- `completed`
- `failed`
- `cancelled`

This vocabulary is intentionally richer than one generic status string so future runner, TUI, and concurrency work can distinguish blocked, recovering, and terminal behavior without redefining the API.

### Run Detail
`RunDetail` is the stable drill-down operator read model. It extends `RunSummary` with:
- `stage_summaries`
- `role_summaries`
- `coordination`
- `audit_summary`
- `artifact_counts_by_class`
- `pending_approval_ids`
- relevant active manifest hashes
- latest policy-decision references where useful for operator understanding
- optional runner advisory information kept explicitly non-authoritative

### Authoritative vs Advisory Run State
- Broker-derived trusted state and trusted daemon state are first-class in the API.
- Runner-reported orchestration details may be exposed only as explicit advisory metadata.
- The API must not require clients to trust untrusted runner persistence as the source of run truth.

## Approval Model

### Canonical Approval Identity
- Approval identity must be derived from the canonical approval-request identity, not from transport session state, UI-generated IDs, or broker-local counters.
- The canonical identity for broker list/get/resolve surfaces should be the approval-request payload hash or another directly equivalent canonical approval-request identity shared with policy and runner state.

### Approval Summary
`ApprovalSummary` should include at least:
- `approval_id`
- `status`
- `requested_at`
- `expires_at`
- `decided_at`
- `consumed_at`
- `approval_trigger_code`
- `changes_if_approved`
- `approval_assurance_level`
- `presence_mode`
- `bound_scope`
- `policy_decision_hash`
- `superseded_by_approval_id`
- request and decision artifact digests where applicable

Recommended status vocabulary:
- `pending`
- `approved`
- `denied`
- `expired`
- `cancelled`
- `superseded`
- `consumed`

### Approval Bound Scope
`ApprovalBoundScope` should normalize the scope that clients need for safe UX and later machine interpretation, including where applicable:
- `workspace_id`
- `run_id`
- `stage_id`
- `step_id`
- `role_instance_id`
- `action_kind`
- `policy_decision_hash`

Clients should not have to scrape freeform approval details just to answer what exact work is blocked or what exact work would be unblocked by approval.

### Approval Resolution
- Approval resolution must consume typed signed approval artifacts and exact hash bindings.
- `ApprovalResolveRequest` is not a free-floating `approve=true` or `deny=true` button message.
- The delivery channel or local client identity is not the authorization primitive for high-risk actions.
- Broker-facing approval APIs are primarily for inspection and resolution of policy-derived approvals, not for creating arbitrary freeform approval requests from untrusted clients.

## Artifact Model

### Public Artifact View
- `ArtifactReference` remains the stable hash-addressed core contract.
- `ArtifactSummary` is the public broker read model that wraps safe operational metadata around `ArtifactReference`.
- Public artifact views must not expose host-local blob paths, storage roots, or daemon-private file layout.

`ArtifactSummary` should include at least:
- `reference`
- `created_at`
- `created_by_role`
- `run_id`
- `stage_id`
- `step_id`
- `approval_of_digest`
- `approval_decision_hash`

### Artifact Operations
- Listing supports filters by scope and class rather than forcing clients to scan full stores.
- Metadata reads use `head`-style request/response contracts.
- Byte reads are streamed and remain broker-mediated with policy and manifest checks applied on every request.
- MVP download scope is full-object hash-addressed read; future range-read support must be an additive extension rather than a shape rewrite.

## Audit Model

### Derived Views Only
- The broker exposes derived audit views and verification summaries; it does not create an alternate audit truth model.
- The broker should reuse or directly map the audit operational-view and verification-summary shapes already established by audit work rather than inventing a second public audit vocabulary.

### Audit Operations
- Audit timeline reads are paged, typed, and cursor-based.
- Audit verification responses surface both the machine-readable report and the derived summary needed by operator UX.
- Audit surfaces expose anchored, unanchored, degraded, and failed posture explicitly through typed fields and reason codes, not scraped human prose.

## Streaming Model

### Uniform Stream Semantics
- Stream families carry a stable `stream_id`.
- Events use strictly monotonic `seq` values.
- Every stream has exactly one terminal event.
- Terminal events explicitly report terminal status rather than relying on transport close semantics.
- Failed terminal events carry the shared typed `runecode.protocol.v0.Error` object.

### Initial Stream Families
The initial stream model should cover at least:
- `LogStreamEvent`
- `ArtifactReadEvent`

The stream contract must be designed so later `RunWatchEvent` or `ApprovalWatchEvent` additions are additive and do not require transport redesign.

## Pagination And Ordering
- Paged reads use opaque cursors rather than page-number semantics.
- Ordering is explicit per operation.
- Audit timeline ordering and artifact/approval listing ordering must be specified so future TUI and CLI work do not infer inconsistent defaults.

Recommended defaults:
- approvals: pending-first, newest-first within status groups
- artifacts: newest-first by creation/update time unless an operation defines stronger semantics
- audit timeline: explicit operational ordering defined by the derived audit view contract

## Health, Readiness, and Version

### Broker Readiness
- `BrokerReadiness` is more than process liveness.
- It should summarize broker-local readiness plus dependent readiness required for operator UX and local supervision.
- Audit-derived readiness dimensions already defined by auditd must remain visible through the broker rather than collapsed to one boolean.

### Broker Version Info
`BrokerVersionInfo` should include at least:
- product version
- build revision
- build time
- protocol bundle version
- protocol bundle manifest hash
- API family/version
- supported transport encodings

Product build identity and logical API identity are different concerns and should remain separate in the contract.

## Main Workstreams
- Broker Responsibilities (MVP)
- Local Client API
- Local Auth
- Artifact Routing Integration

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
