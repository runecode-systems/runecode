# Protocol & Schema Bundle v0

User-visible outcome: cross-boundary and cross-isolate communication is structured, schema-validated, versioned, and hash-addressable, with typed approvals, stable identities, auditable streaming/model outputs, and reserved extension points for future auth and bridge-provider work.

## Task 1: Save Spec Documentation

Maintain `agent-os/specs/2026-03-08-1039-protocol-schemas-v0/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

Parallelization: docs-only; safe to do anytime.

## Task 2: Define Canonical Object Families + Code Registries

Define the shared top-level object families that all downstream specs must use rather than inventing ad-hoc payloads:
- role manifests
- run/stage capability manifests
- principal/role identity objects
- shared digest/hash reference objects
- artifact references + provenance receipts
- audit events + audit receipts
- approval requests + approval decisions
- policy decisions
- `LLMRequest` / `LLMResponse` + streaming event objects
- signed object envelopes
- shared error envelopes
- post-MVP reserved workflow/process-definition objects that remain schema-governed and cannot expand capabilities without a schema bump + security review

Define registry boundaries so machine-consumed codes do not bleed together:
- `error.code` for transport/validation/storage/auth/runtime failures
- `policy_reason_code` for deterministic policy outcomes
- `approval_trigger_code` for why human approval is requested
- `audit_event_type` for event classification
- object-family-specific enums only where shared registries are insufficient

Registry rules:
- each registry has its own namespace and documentation owner
- codes are stable, machine-consumed identifiers; human-readable text lives in separate message/detail fields
- no downstream spec may reuse `error.code` values as policy reasons or approval triggers

Top-level object requirements:
- every persisted or cross-boundary top-level object includes `schema_id` and `schema_version`
- `schema_id` is the stable object-family identifier; `schema_version` is an exact version string, not a hint for permissive parsing
- protocol-owned `schema_id` values use the format `runecode.protocol.v0.<ObjectFamily>`
- runtime `schema_id` values are protocol identifiers and must not be replaced by JSON Schema `$id` file/tooling identifiers
- unknown object families or versions fail closed
- maintain an authoritative schema registry manifest at `protocol/schemas/manifest.json` listing each schema file, `schema_id`, `schema_version`, object-family owner, and status (`mvp` or `reserved`)

Manifest requirements:
- role manifests and run/stage capability manifests carry explicit opt-ins, approval profile selection, and allowlist references as signed inputs
- manifests never rely on implicit defaults for security-sensitive capability expansion

Digest/hash reference requirements (MVP):
- all content-addressed references use a shared digest object with:
  - `hash_alg`
  - `hash`
- MVP requires `hash_alg = sha256` for shared protocol hashes; unknown hash algorithms fail closed

Signed object envelope requirements (MVP):
- standardize signature fields across manifests, audit events, provenance/approval receipts, and related signed objects:
  - `alg` (for example `ed25519`)
  - `key_id`
  - `signature` (bytes as base64)
- use a detached payload signing model: `signature` covers the RFC 8785 JCS canonical bytes of the payload object before any signature wrapper fields are attached
- implementations must never sign language-specific serialized forms or self-referential objects that include their own `signature` field

Implementation findings + decisions after the initial Task 2 review:
- `SignedObjectEnvelope.payload` is constrained to object payloads that carry `schema_id` and `schema_version`; the detached wrapper no longer accepts arbitrary JSON scalars or nulls.
- `SignedObjectEnvelope.payload` is classified as `secret` at the wrapper layer so nested secret-bearing payload families fail safe even before broker-side schema introspection recurses into the payload schema.
- MVP signature algorithms are explicitly allowlisted at the schema layer; Task 2 pins the shared signature block to `ed25519` rather than accepting arbitrary `alg` strings.
- Conservative structural limits (`maxLength`, `maxItems`, `maxProperties`) are part of the Task 2 schema bundle so trust-boundary validators inherit fail-closed bounds before Task 10 adds deeper transport-specific limits.
- Task 2 now adds field-level `x-data-class` metadata (`public | sensitive | secret`) to shared schemas so later broker-side redaction/rejection work has a stable annotation shape.
- Task 2 adds property-level descriptions across object schemas so generated tooling and reviewers get field-level protocol documentation rather than title-only object summaries.
- Shared digest and signature fragments are centralized via reusable JSON Schema definitions/references instead of copy-pasted inline shapes.
- `protocol/schemas/manifest.json` is authoritative in both directions: verification rejects omitted files, stray files on disk, and manifest paths that escape `protocol/schemas/`.
- Go verification for Task 2 compiles every checked-in schema against JSON Schema draft 2020-12 in addition to manifest/registry invariant tests.
- Task 2 validates `$ref` targets as well as inline schema nodes so shared digest/signature definitions cannot silently lose bounds, descriptions, or classification metadata through indirection.
- Task 2 treats shared registry code values as pairwise non-overlapping across all bundle registries to keep machine-consumed namespaces fail-closed even when short codes are reused accidentally.
- `PrincipalIdentity.role_kind` is now schema-constrained: `role_instance` actors must declare it, while `user` and `local_client` identities must not attach role kinds prematurely; daemon and external-runtime semantics stay extensible for Task 3.
- Task 2 adds meta-schemas for the schema manifest and registry files plus CI validation for those documents so Go/TS tooling can share a machine-readable contract for bundle metadata.
- Task 2 keeps `ApprovalRequest`, `ApprovalDecision`, `PolicyDecision`, and `Error` in MVP bundle scope but marks them as minimal family anchors via manifest notes; their owning tasks still add the remaining shared fields under explicit schema-versioned follow-up work.
- Task 2 documents that schema-document `$id` URIs are canonical identifiers for tooling and reference resolution, not a requirement to fetch live network content.
- Empty `policy_reason_code`, `approval_trigger_code`, and `audit_event_type` registries remain intentionally reserved until downstream policy, approval, and audit specs define concrete values.

Parallelization: finalize the object-family list and code-registry split early so policy, broker, audit, and gateway work do not diverge.

## Task 3: Define Identity, Manifest, and Lifecycle Semantics

Define a shared principal identity object used across requests, approvals, audit events, leases, and receipts:
- actor kind (user, daemon, role instance, local client, external runtime)
- role kind where applicable (workspace, gateway, auth, model, git, web, deps)
- stable instance/session identifiers
- run/stage identifiers where applicable
- active manifest hash / capability hash references
- signing key identifier or verifier reference when relevant

Manifest + lifecycle rules:
- action requests, approvals, policy decisions, and leases bind to the active manifest hash and relevant schema versions
- a manifest change, approval-profile change, allowlist change, or policy input change creates a new hash identity and cannot silently inherit old approvals/leases
- components must revalidate or reissue requests when bound inputs change
- stale session state fails closed rather than being "best effort" reused

Gateway identity requirements:
- gateway-role audit events include role identity, role kind, and stable destination descriptors
- `secretsd` and other trusted daemons rely on the shared principal identity object rather than ad-hoc caller metadata
- any future bridge runtime is modeled as an external-runtime/gateway principal and inherits shared gateway invariants even before provider-specific bridge specs are implemented

Parallelization: can be designed in parallel with policy and broker work once the principal identity object and manifest-hash binding rules are agreed.

## Task 4: Define Approval Object Model (MVP, Profile-Ready)

Define typed approval objects owned by this protocol spec:
- `ApprovalRequest`
- `ApprovalDecision`
- approval receipt/hash conventions used by audit and runner pause/resume flows

`ApprovalRequest` must include:
- `manifest_hash`
- `action_request_hash`
- `relevant_artifact_hashes`
- requester principal identity
- `approval_trigger_code`
- structured details payload (typed; no freeform-only prompts)
- explicit expiry / TTL metadata (required in the serialized protocol object; no implicit default at the protocol layer)
- a clear statement of what changes if approved

`ApprovalDecision` must include:
- referenced approval request hash
- approver principal identity
- decision outcome (`approve | deny | expired | cancelled`)
- decision timestamp and any expiry/consumption semantics
- optional structured restrictions/notes that are machine-checkable
- optional references to the policy decision or stage-manifest summary the human is acting on

Approval invariants:
- approvals are hash-bound to immutable inputs
- if any bound hash changes while approval is pending, the request is stale and invalid
- approval objects never expand capability beyond what policy + signed manifests already allow
- the protocol spec owns shared object shapes and binding rules; approval profile specs own additional profile semantics

Approval profile rules (MVP):
- MVP supports a single profile value: `moderate`
- unknown approval profile values fail closed
- adding new profile values (for example `strict` and `permissive`) is post-MVP and requires a schema version bump for the affected object families

Parallelization: finalize approval objects early because runner, policy, TUI, and audit all depend on the same contract.

## Task 5: Define Shared Error Envelope + Versioning Rules

Define a single `Error` object used across trust boundaries:
- stable `code`
- `category` (`validation | auth | policy | transport | storage | timeout | internal`)
- `retryable` (bool)
- `message` (human-facing and non-sensitive)
- optional typed `details` with its own `details_schema_id`

Define a shared `PolicyDecision` object family:
- decision outcome: `allow | deny | require_human_approval`
- stable `policy_reason_code`
- structured decision details / required-approval payloads
- hashes of evaluated inputs (manifest, action request, relevant artifacts, policy inputs)
- the protocol spec owns the shared `PolicyDecision` shape and fields; `agent-os/specs/2026-03-08-1039-policy-engine-v0/` owns evaluation semantics, precedence rules, and policy-specific detail schemas

Versioning + compatibility rules:
- every top-level object hash binds to the exact `schema_id` + `schema_version` used for validation and canonicalization
- trust boundaries reject unknown fields; there is no loose parsing mode
- any field addition, removal, rename, semantic change, or enum expansion that matters across the boundary requires a schema version bump for that object family
- verifiers retain old schemas so previously persisted objects remain verifiable
- if a verifier encounters an unsupported version, it fails closed with a stable `error.code`
- MVP runtime posture is same-bundle only: communicating components participating in one active local session must use the same schema bundle version
- schema-bundle upgrades are coordinated local restarts, not live mixed-version negotiation
- session/open audit metadata records the schema bundle version used by each daemon/client participating in the session
- rollback posture: previously persisted objects at older versions remain verifiable, but operational rollback requires downgrading components and reissuing new runtime objects under the supported schema bundle

Code taxonomy rules:
- error codes, policy reason codes, and approval trigger codes each have separate registries
- registry additions are security-sensitive because they may affect enforcement, TUI rendering, or automation behavior
- release notes and fixture changes must make registry additions/reinterpretations explicit

Parallelization: can proceed in parallel with verifier work once the shared envelope and versioning contract are fixed.

## Task 6: Define Model Protocol Objects + Streaming Semantics

Model gateway protocol objects:
- `LLMRequest`
- `LLMResponse`
- streaming event objects for incremental output and terminal status

`LLMRequest` requirements:
- provider/model selection fields that do not allow arbitrary capability escalation
- inputs reference artifacts by hash; no raw prompt blobs cross boundaries
- explicit tool allowlist per request
- tool-call argument objects are schema-validated and reject unknown/extra fields
- conservative limits cap tool calls per response and total tool-call bytes
- output schema references for any machine-consumed structured output
- request limits (bytes, tool-call count, timeouts, streaming posture)

MVP default model-protocol limits:
- max `LLMRequest` payload size (excluding referenced artifact bytes): 256 KiB
- max tool calls proposed per response: 8
- max total tool-call argument bytes per response: 64 KiB
- max structured-output payload bytes per final response or proposal object: 256 KiB
- default total streamed bytes per response: 16 MiB
- default streaming chunk/event payload size: 64 KiB
- default streaming idle timeout: 15 seconds

`LLMResponse` requirements:
- outputs are untrusted proposals and must be representable as typed artifacts
- tool calling is supported only as schema-validated proposal objects; never direct execution
- structured JSON outputs are required for any machine-consumed output that can drive actions

Streaming semantics:
- define ordered event types for start, incremental content, proposal emission, structured-output candidates, and terminal status
- every streamed response has sequence numbers and exactly one terminal event
- terminal events distinguish success, interruption/cancellation, and failure
- the terminal event identifies the authoritative final response object/hash (or the final typed error)
- partial events are auditable but do not implicitly authorize side effects
- define how streaming interruptions, deadlines, and broker-enforced truncation surface in the protocol
- if the broker enforces truncation, timeout, or cancellation, the broker emits the terminal status/event with broker attribution rather than masquerading as gateway-originated output

Audit requirements for model traffic:
- audit events must be gateway-role aware
- include egress category metadata for outbound network activity (model, auth, git, web, deps)
- include allowlist identifiers and stable destination descriptors without logging secret values

Parallelization: schema design can proceed in parallel with gateway, broker, and audit work, but streaming terminal semantics and request/response object shapes must be fixed early.

## Task 7: Define Artifact References, Provenance, and Audit Linkage

Define artifact references with the minimum shared fields needed across subsystems:
- digest reference (`hash_alg`, `hash`)
- size
- content type
- data class
- provenance reference

Replace vague origin metadata with typed provenance:
- define an artifact provenance object or receipt that links an artifact to:
  - producing principal identity
  - run/stage/session identifiers where applicable
  - producing audit event hash or receipt hash
  - source artifact hashes when derived from other artifacts
  - creation timestamp / schema version metadata
- provenance must be machine-checkable and stable enough for audit + verification tooling

Audit event requirements:
- audit events are typed, hash-chained, signed, and schema-versioned
- event objects reference related artifacts, principals, decisions, and receipts by hash
- downstream specs may add event-type-specific fields, but they must extend the shared audit object family rather than inventing ad-hoc event shapes

Parallelization: can be implemented in parallel with artifact store and audit work once provenance and audit linkage fields are agreed.

## Task 8: Define Auth Extension Object Families (Post-MVP Reserved)

Reserve provider-agnostic auth object families now so later auth/provider specs do not invent ad-hoc control messages:
- `AuthRequest`
- `AuthChallenge`
- `AuthCompletion`
- `AuthLeaseGrant` / equivalent lease-handoff receipt
- typed auth failure/error detail objects where needed

Reserved auth-family requirements:
- support browser-loopback, device-code, and refresh-style flows at the object-family level
- include provider identity, flow kind, correlation/state binding, requested scopes, and callback/challenge descriptors where applicable
- model the handoff to `secretsd` / `model-gateway` via lease references and receipts, not raw token values
- no auth object that can reach TUI, runner, audit fixtures, or general broker clients may contain raw secrets
- provider-specific OAuth parameters stay in provider specs; this protocol spec owns the shared object families, invariants, and versioning expectations only

Parallelization: design-only for MVP; reserve the object families now so post-MVP auth-gateway and provider work inherit consistent contracts.

## Task 9: Define Bridge Runtime Extension Object Families (Post-MVP Reserved)

Reserve bridge-provider object families for local runtimes behind `model-gateway`:
- `BridgeRuntimeIdentity`
- `BridgeCompatibilityProbe`
- `BridgeSessionPosture`
- `BridgeTokenChallenge` or equivalent refresh-needed signal
- typed bridge RPC error mapping objects

Reserved bridge-family requirements:
- runtime identity/version must be explicit and auditable
- compatibility probe results must be typed and record required features, tested-range posture, and pass/fail status
- bridge posture must expose whether the runtime is in explicit `LLM-only` mode
- token delivery modes must avoid env vars and raw secret logging
- provider specs choose the exact RPC mapping/protocol details; this protocol spec owns the shared object families, invariants, and versioning expectations only

Security invariants:
- bridge requests to execute commands, read/write workspace files, or apply patches are denied and treated as policy violations
- no listening network ports by default; prefer local child-process IPC
- these bridge invariants are shared broker/policy enforcement requirements, not optional provider-specific behavior

Parallelization: design-only for MVP; reserve the bridge object families now so subscription-backed providers align to the same posture and audit model later.

## Task 10: Choose Schema + Validation Strategy

- Use JSON Schema draft 2020-12 as the single source of truth for MVP:
  - on-wire local RPC messages (broker <-> isolates <-> clients) use JSON (MVP)
  - on-disk manifests and policy documents use JSON
- Generate or derive validators for both Go and TS from the same schema bundle.
- Cross-language schema validation and canonicalization results must be identical; shared golden fixtures are the authoritative contract when implementations disagree.
- Maintain the authoritative schema registry manifest at `protocol/schemas/manifest.json` and generate or verify validator inputs against it deterministically.
- Keep post-MVP protobuf migration feasible by restricting schemas to an MVP profile that maps cleanly to protobuf messages:
  - avoid regex-heavy schemas and dynamic keys (`patternProperties` / arbitrary maps) in on-wire messages
  - model unions via an explicit discriminator field
  - keep numeric ranges within I-JSON expectations; represent high-precision numbers as strings
- Add schema-authoring lint/tooling that fails on constructs forbidden by the MVP profile so protobuf-hostile patterns are rejected during authoring, not discovered later.
- Fail closed at trust boundaries:
  - reject unknown fields
  - enforce message size limits and structural complexity limits (depth / array length)
  - keep auth/bridge reserved families under the same fail-closed discipline even before they are implemented
- Canonicalization for hashing/signing (MVP requirement):
  - use RFC 8785 (JSON Canonicalization Scheme, JCS) for canonical bytes
  - prohibit floats/NaN/Infinity in hashed/signed objects; use integers or strings
  - encode bytes as base64 strings; timestamps as RFC 3339 strings; durations as integer milliseconds
  - hash/sign inputs are the canonical JSON bytes produced by JCS
  - validate canonicalization correctness using RFC 8785 reference test vectors and cross-language golden fixtures
  - canonicalization operates on plain JSON values; do not depend on language-specific serializers
  - if a third-party canonicalizer is used, pin versions and require golden-fixture parity in CI
- Add field-level data classification metadata in schemas (`public | sensitive | secret`) to support structural redaction and boundary enforcement.
- The broker is the canonical enforcement point for schema-driven secret/sensitive field rejection or stripping at the trusted/untrusted boundary; producer-side enforcement remains defense-in-depth.

Parallelization: can be implemented in parallel with audit/artifact subsystems as long as the schema profile and canonicalization rules are fixed.

## Task 11: Reference Fixtures + Cross-Language Validation

Add checked-in fixtures that validate against schemas and capture both success and fail-closed behavior:
- role manifest and run/stage capability manifest fixtures
  - include a microVM-stage example and a container-stage explicit-opt-in example
  - include the MVP `moderate` approval profile in the relevant manifest fixtures
- signed payload fixtures that prove detached-signature input construction (`JCS(payload)` before wrapper fields)
- principal identity examples across user, daemon, role-instance, and external-runtime actors
- approval request/decision fixtures, including expiry and stale-input invalidation cases
- policy decision fixtures that reference approval triggers and policy reason codes separately
- shared error envelope fixtures and invalid-code/invalid-details cases
- `LLMRequest` / `LLMResponse` fixtures using only `spec_text` inputs for MVP
- streaming event-sequence fixtures, including success, interruption/cancellation, timeout, and failure
- artifact provenance / receipt fixtures linking artifacts back to producing audit events
- auth reserved-family fixtures (shape-only, no secret material)
- bridge reserved-family fixtures for runtime identity, compatibility probe pass/fail, posture, and error mapping
- schema-bundle session/open fixtures that record bundle versions per participating component/client
- canonicalization + hashing fixtures:
  - canonical JSON bytes (golden)
  - expected hash outputs
  - expected signature verification outcomes where relevant

Fixture governance:
- fixtures live under `protocol/fixtures/` and are treated as security-sensitive contract artifacts
- regeneration is explicit; tooling must not auto-update fixtures during `just ci`
- any fixture update must be reviewable and explain whether the change is a bug fix, drift correction, or intentional capability expansion
- fixture sets must cover both Go and TS consumers and reject the same invalid inputs
- Go and TS test runners iterate the same manifest-defined fixture set; fixture count and fixture IDs must match across languages
- provide an explicit repo-local fixture update workflow/command separate from `just ci`; CI verifies outputs but never regenerates them implicitly

Parallelization: fixtures can be created in parallel across subsystems as long as they validate against the same schema bundle and canonicalization rules.

## Task 12: On-Wire Encoding Migration Plan (Post-MVP)

- Keep the logical object model stable and documented independent of encoding.
- Prefer protobuf message encoding for on-wire local RPC post-MVP without requiring gRPC:
  - define `.proto` message definitions that map 1:1 to the logical model
  - keep golden fixtures and cross-language tests so JSON and protobuf encodings are behaviorally equivalent
  - continue using local IPC transports (UDS / named pipes / vsock / virtio-serial); do not introduce a network API by default
  - keep message framing, size limits, deadlines/timeouts, and backpressure as explicit requirements regardless of transport
- gRPC is optional (post-MVP) and must remain local-only:
  - prefer gRPC over Unix domain sockets (Unix) and OS-native local IPC (for example named pipes on Windows) where supported
  - do not use TCP by default
  - if TCP loopback is used for compatibility, require either mTLS with pinned/trusted local certificates or a strong short-lived local token mechanism stored with strict filesystem permissions
  - binding safety is a security requirement: never bind privileged APIs to non-loopback interfaces
- Do not change hashing/signing semantics for persisted or signed objects; canonicalization remains defined by this spec.

Parallelization: design-only; can be done anytime after MVP schema rules are stable.

## Acceptance Criteria

- Go and TS validate the same fixtures deterministically and reject the same invalid inputs.
- Go and TS produce the same canonical bytes for the same logical payloads; golden fixtures are authoritative when implementations disagree.
- Every persisted or cross-boundary top-level object family used in MVP has explicit `schema_id` and `schema_version` fields.
- Protocol-owned `schema_id` values follow a stable namespaced convention and are listed in `protocol/schemas/manifest.json`.
- Shared digest/hash references use an explicit digest object and pin `sha256` as the MVP hash algorithm.
- Signed object envelopes define the exact signing input as the JCS canonical bytes of the detached payload.
- Shared approval request/decision schemas exist and cover binding, expiry, and stale-input invalidation semantics.
- Shared principal identity objects are used consistently across manifests, approvals, leases, audit events, and bridge/auth extension families.
- All communicating components in a live local session use the same schema bundle version, and session/open audit metadata records those versions.
- Streaming responses have ordered event types, sequence rules, and exactly one terminal event with deterministic success/interruption/failure semantics.
- Broker-enforced truncation/timeouts produce broker-attributed terminal status rather than ambiguous gateway output.
- Artifact provenance is typed and sufficient to link produced artifacts back to principals and audit events.
- Shared error envelope fixtures exist and are validated consistently across Go and TS.
- Error codes, policy reason codes, and approval trigger codes are documented as separate registries.
- Concrete MVP defaults exist for model request size, tool-call limits, structured-output size, and streamed-byte limits.
- Broker boundary enforcement uses schema field classification metadata as the canonical secret/sensitive redaction or rejection mechanism.
- Reserved auth and bridge object families are defined strongly enough that downstream specs can extend them without inventing ad-hoc cross-boundary shapes.
- The schema/profile avoids constructs that would make post-MVP protobuf migration impractical.
