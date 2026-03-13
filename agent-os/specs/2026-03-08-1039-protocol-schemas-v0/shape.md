# Protocol & Schema Bundle v0 - Shaping Notes

## Scope

Refine the schema foundation for manifests, identities, approvals, artifacts/provenance, audit events, policy decisions, and the model-gateway request/response boundary so downstream MVP and post-MVP specs inherit stable shared contracts instead of inventing ad-hoc payloads.

## Decisions

- The protocol spec owns shared object families and code registries; downstream specs extend semantics, workflows, and provider-specific behavior without redefining common cross-boundary shapes.
- Protocol-owned top-level object families use stable namespaced `schema_id` values and are tracked in an authoritative schema registry manifest.
- Cross-boundary communication is schema-validated and logged; no freeform text triggers privileged actions.
- No freeform prompt blobs cross trust boundaries; model requests reference artifacts by hash and remain typed (`LLMRequest` / `LLMResponse`).
- Shared content-addressed references use explicit digest objects; MVP pins `sha256`.
- MVP model protocol supports streaming and tool calling, but only as typed artifacts/proposals; bridge/tool outputs never imply execution.
- Streaming must have explicit ordered event types and exactly one terminal event so broker, audit, and gateway implementations stay deterministic.
- If the broker terminates or truncates a stream, that terminal status is broker-attributed rather than treated as gateway output.
- Approvals are first-class typed objects with hash binding, expiry, and stale-input invalidation semantics.
- Approval TTL/expiry is explicit in the serialized protocol object; the protocol layer does not rely on implicit defaults.
- Principal identity is a shared protocol concern; requests, approvals, leases, receipts, and audit events use the same identity model.
- Signed/canonicalized objects require deterministic serialization; canonicalization is RFC 8785 JCS validated via cross-language golden fixtures.
- Signed objects use a standardized signature envelope including `{alg, key_id, signature}` and sign the detached payload's JCS canonical bytes.
- The shared signature block is schema-allowlisted to MVP-safe algorithms and the detached payload wrapper requires object payloads with explicit schema identity/version metadata.
- The detached payload wrapper is classified fail-safe as `secret` until broker-side recursive schema introspection can apply nested field classifications precisely.
- Error codes, policy reason codes, and approval trigger codes are distinct registries; no downstream spec should conflate them.
- Shared registries use separate namespaces and Task 2 verification now treats cross-registry code reuse as a fail-closed error rather than relying on namespace disambiguation alone.
- Artifact origin must be replaced by typed provenance or receipt objects that link artifacts to producing principals, stages, and audit events.
- Schemas carry field-level data classification metadata (`public | sensitive | secret`) to support structural redaction and boundary enforcement.
- Shared schemas include conservative structural bounds, field-level descriptions, and bidirectional manifest verification so stray files, escaped manifest paths, and undocumented property contracts fail closed.
- Shared `$ref` definitions are verified with the same invariant checks as inline schemas so reusable digest/signature fragments cannot drift silently.
- MVP-scoped placeholder families remain explicit via manifest notes; dedicated later tasks add their remaining fields under schema-versioned follow-up work instead of silently widening Task 2 shells.
- The broker is the canonical enforcement point for schema-driven secret/sensitive field rejection or stripping at the trusted/untrusted boundary.
- MVP favors a single schema source of truth that can be validated in both Go and TS.
- MVP uses JSON Schema draft 2020-12 and JSON-on-wire; the logical object model remains encoding-agnostic so on-wire RPC can migrate post-MVP to protobuf over local IPC.
- Bundle metadata (`manifest.json` and registry files) also has machine-readable meta-schemas, and schema-document `$id` URIs are canonical identifiers rather than a network fetch contract.
- MVP runtime posture is same-schema-bundle only; upgrades are coordinated restarts rather than mixed-version live negotiation.
- Auth and bridge-provider object families are reserved now at the shared-contract level, but provider-specific OAuth/RPC details stay in dedicated later specs.

## Context

- Visuals: None.
- References:
  - `agent-os/product/mission.md`
  - `agent-os/product/roadmap.md`
  - `agent-os/product/tech-stack.md`
  - `docs/trust-boundaries.md`
  - `agent-os/specs/2026-03-10-1530-approval-profiles-v0/`
  - `agent-os/specs/2026-03-12-1030-auth-gateway-role-v0/`
  - `agent-os/specs/2026-03-11-1920-openai-chatgpt-subscription-provider-v0/`
  - `agent-os/specs/2026-03-11-1921-github-copilot-subscription-provider-v0/`
- Product alignment: reinforces least-privilege defaults, brokered schema-validated trust boundaries, typed approvals, and tamper-evident auditability without over-specifying provider details too early.

## Standards Applied

- `security/trust-boundary-interfaces` - all shared schemas and fixtures are part of the allowed cross-boundary surface.
- `security/trust-boundary-layered-enforcement` - protocol objects must support broker validation, policy enforcement, and isolation backends without weakening any layer.
- `security/trust-boundary-change-checklist` - protocol changes are security-sensitive and must stay aligned with docs, fixtures, and guardrails.
- `security/runner-boundary-check` - the runner may only consume shared protocol schemas/fixtures, so the object model must be explicit and fail closed.
- `global/deterministic-check-write-tools` - schema and fixture generation/update flows must stay deterministic and explicit.
- `ci/worktree-cleanliness` - fixture/schema workflows must be check-only in CI and must not mutate the repo implicitly.
