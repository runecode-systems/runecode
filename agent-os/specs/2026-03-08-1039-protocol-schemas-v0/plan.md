# Protocol & Schema Bundle v0

User-visible outcome: cross-component and cross-isolate communication is structured, schema-validated, and hash-addressable, enabling deterministic policy and audit.

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-08-1039-protocol-schemas-v0/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

Parallelization: docs-only; safe to do anytime.

## Task 2: Define Core Object Model

Define the minimal canonical objects needed for MVP:
- Role manifests and run/stage capability manifests (including explicit opt-ins and an approval profile).
- Artifact references (hash, size, content type, data class, origin).
- Audit events (hash-chained, signed, typed).
- Approval requests/decisions (typed, structured payloads for TUI display and deterministic enforcement).
- Policy decisions (allow/deny/require_human_approval) with reason codes.
- Signed object envelopes (MVP):
  - Standardize signature fields across manifests, audit events, and receipts:
    - `alg` (e.g., `ed25519`)
    - `key_id`
    - `signature` (bytes as base64)
- Shared error taxonomy + envelope (MVP):
  - Define a single `Error` object used across all boundaries:
    - stable `code` (string enum)
    - `category` (validation/auth/policy/transport/storage/timeout/internal)
    - `retryable` (bool)
    - `message` (human-facing; non-sensitive)
    - optional typed `details` with its own `details_schema_id`
  - Require all daemons/components to use this envelope rather than inventing ad-hoc error shapes.
- Model gateway protocol objects:
  - `LLMRequest` and `LLMResponse` (including streaming event shapes where applicable)
  - provider/model selection fields that do not allow arbitrary capability escalation
  - inputs must reference artifacts by hash (no raw prompt blobs crossing boundaries)
  - outputs are untrusted proposals and must be representable as typed artifacts
- Model output features (MVP):
  - streaming: supported; define incremental event types and completion semantics
  - tool calling: supported only as typed proposal objects (never direct execution)
    - `LLMRequest` carries an explicit tool allowlist per request.
    - Tool-call args are schema-validated; unknown/extra fields are rejected.
    - Add conservative limits (e.g., cap tool calls per response and cap total tool-call bytes).
  - structured JSON outputs: required for any machine-consumed output that can drive actions
- Audit events must be gateway-role aware:
  - include role identity and role kind (workspace vs gateway)
  - include egress category metadata for outbound network activity (model, auth, git, web, deps)
  - include allowlist identifiers and stable destination descriptors (without logging secret values)
- Reserved (post-MVP): `ProcessDefinition` (JSON/YAML) as the user-configurable process surface:
  - a schema-validated step graph model (sequential + branching + optional parallel blocks)
  - allowlisted RuneCode step types only (cannot introduce new capabilities)
    - Define the initial allowlist (illustrative):
      - `llm_request`
      - `workspace_read`
      - `workspace_edit`
      - `workspace_test`
      - `gate_run`
      - `approval_checkpoint`
      - `git_gateway_pr_create` (post-MVP; requires git-gateway)
      - `web_research` (post-MVP; requires web-research gateway)
  - Governance note: adding a new allowlisted step type is a capability expansion.
    - requires a schema version bump and a security review
    - must be surfaced in release notes and (when used) recorded in audit metadata
  - per-step provider/model selection and step-level limits
- Reserved (post-MVP) protocol surface for `bridge` providers (local runtimes behind model-gateway):
  - a typed request/response envelope, runtime identity/version fields, and stable error taxonomy
  - Define an explicit "LLM-only" capability mode for bridge runtimes.
    - bridge requests to execute commands, read/write workspace files, or apply patches are denied and treated as policy violations
  - explicit streaming support and backpressure/queueing signals
  - contract-test fixtures for request/response envelopes and error mapping

Parallelization: schema design can proceed in parallel with crypto, broker, and policy work, but the signed/canonicalized envelope rules (JCS profile, `{alg, key_id}`, and the shared error envelope) must be finalized early to avoid churn.

## Task 3: Choose Schema + Validation Strategy

- Use JSON Schema as the single source of truth for MVP:
  - on-wire local RPC messages (broker <-> isolates <-> clients) use JSON (MVP)
  - on-disk manifests and policy documents use JSON
- Generate/derive validators for both Go and TS from the same schema bundle.
- To keep post-MVP protobuf migration feasible (with an optional gRPC facade), restrict schemas to an MVP profile that maps cleanly to protobuf messages:
  - avoid regex-heavy schemas and dynamic keys (`patternProperties` / arbitrary maps) in on-wire messages
  - model unions via an explicit discriminator field (no ambiguous `oneOf` without a tag)
  - keep numeric ranges within I-JSON expectations; represent high-precision numbers as strings
- Fail closed at trust boundaries:
  - reject unknown fields (no permissive parsing)
  - enforce message size limits and structural complexity limits (depth / array length)
- Canonicalization for hashing/signing (MVP requirement):
  - Use RFC 8785 (JSON Canonicalization Scheme, JCS) for canonical bytes.
  - Prohibit floats/NaN/Infinity in hashed/signed objects; use integers or strings.
  - Encode bytes as base64 strings; timestamps as RFC 3339 strings; durations as integer milliseconds.
  - Hash/sign inputs are the canonical JSON bytes produced by JCS.
  - Implementation guidance (MVP):
    - Validate canonicalization correctness using RFC 8785 reference test vectors (cross-language golden fixtures).
    - Prefer using the RFC 8785 reference vectors/implementations from `https://github.com/cyberphone/json-canonicalization` as the baseline.
      - Consider vendoring the canonicalizer code for Go/TS to reduce supply-chain risk (fixtures enforce correctness either way).
    - Canonicalization operates on plain JSON values; do not depend on language-specific object serializers.
    - If a third-party canonicalizer is used, pin versions and require golden fixture parity in CI.
- Add field-level data classification metadata in schemas (`public | sensitive | secret`) to support structural redaction/boundary enforcement.

Parallelization: can be implemented in parallel with audit/artifact subsystems as long as fixtures and canonicalization rules are agreed.

## Task 6: On-Wire Encoding Migration Plan (Post-MVP)

- Keep the logical object model stable and documented independent of encoding.
- Prefer protobuf message encoding for on-wire local RPC post-MVP without requiring gRPC:
  - define `.proto` message definitions that map 1:1 to the logical model
  - keep golden fixtures and cross-language tests so JSON and protobuf encodings are behaviorally equivalent
  - continue using local IPC transports (UDS / named pipes / vsock / virtio-serial); do not introduce a network API by default
  - keep message framing, size limits, deadlines/timeouts, and backpressure as explicit requirements regardless of transport
- gRPC is optional (post-MVP) and must remain local-only:
  - prefer gRPC over Unix domain sockets (Unix) and OS-native local IPC (e.g., named pipes on Windows) where supported
  - do not use TCP by default
  - if TCP loopback is used for compatibility, require one of:
    - mTLS with pinned/trusted local certificates, or
    - a strong, short-lived local token mechanism (stored with strict filesystem permissions)
  - binding safety is a security requirement: never bind privileged APIs to non-loopback interfaces
- Do not change hashing/signing semantics for persisted/signed objects (canonicalization remains defined by this spec).

Parallelization: design-only; can be done anytime after MVP schema rules are stable.

## Task 4: Versioning + Compatibility Rules

- Every top-level object includes explicit `schema_id` and `schema_version` fields.
- Manifest hashes bind to the specific schema version used for validation/canonicalization.
- Compatibility model (MVP):
  - no "loose" parsing at trust boundaries (unknown fields are rejected)
  - changes require a schema version bump
  - older schema versions remain verifiable (verifier keeps old schemas)
- If the verifier encounters an unsupported schema version, verification fails closed with a clear reason code.

Approval profile versioning note:
- MVP supports a single approval profile value (`moderate`). Adding new profiles (e.g., `strict`, `permissive`) is a schema version bump and is post-MVP.

Approval profile semantics note:
- Approval profiles must never convert `deny -> allow`.
- Approval profiles only affect whether an otherwise-allowed action returns `allow` vs `require_human_approval`.
- Unknown profile values fail closed.

Parallelization: can be implemented in parallel with verifier work; it depends on a stable schema bundle/version registry.

## Task 5: Reference Fixtures

- Add small, checked-in example manifests and events that validate against schemas.
- Include both a “microVM stage” and a “container stage (explicit opt-in)” fixture.
- Include an MVP approval profile fixture (`moderate`) embedded in the run/stage manifest.
- Include a minimal `LLMRequest`/`LLMResponse` fixture that uses only `spec_text` inputs.
- Include fixtures for:
  - streaming event sequences (including interruption/cancellation)
  - tool-call proposal outputs (schema-valid)
  - structured JSON output validation (schema pass/fail cases)
  - bridge provider envelope + error taxonomy examples (post-MVP)
  - ProcessDefinition example (post-MVP; validates but cannot expand capabilities)
- Add canonicalization + hashing fixtures:
  - canonical JSON bytes (golden)
  - expected hash outputs
  - (where relevant) expected signature verification outcomes

Fixture governance (MVP):
- Fixtures live under `protocol/fixtures/` and are treated as security-sensitive contract artifacts.
- Regeneration is explicit; tooling must not auto-update fixtures during `just ci`.
- Any fixture update must be reviewable:
  - canonicalized JSON
  - volatile fields stripped/canonicalized (timestamps, content-length, auth headers)
  - clear change rationale (vendor drift vs intentional capability expansion)

Parallelization: fixtures can be created in parallel across subsystems (audit/policy/gateway) as long as they validate against the same schema bundle and canonicalization rules.

## Acceptance Criteria

- Go and TS validate the same fixtures deterministically and reject the same invalid inputs.
- Canonical bytes and hash inputs are stable across platforms (golden fixtures pass in CI).
- Schema versions are explicit and bound to hashes; verification fails closed on unknown versions.
- All cross-boundary messages used in MVP are schema-defined and validated.
- Shared error envelope fixtures exist and are validated consistently across Go and TS.
- The schema/profile avoids constructs that would make post-MVP protobuf migration impractical.
