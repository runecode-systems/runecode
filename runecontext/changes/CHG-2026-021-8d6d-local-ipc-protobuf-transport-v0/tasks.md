# Tasks

## Proto Mapping for the Existing Logical Model

- [ ] Define `.proto` messages that map 1:1 to the existing logical local-API object model.
- [ ] Preserve existing error envelope, hashes, and schema-versioning rules.
- [ ] Keep persisted-object hashing and signing semantics defined by the logical JSON model, including RFC 8785 JCS canonicalization for persisted JSON objects.
- [ ] Preserve `request_id` on request/response flows and `stream_id` + `seq` on stream flows.
- [ ] Preserve typed run, approval, artifact, audit, readiness, and version read models without transport-specific semantic forks.
- [ ] Preserve any dedicated broker-owned product lifecycle posture surface and its stable product-instance identity, lifecycle generation, attach mode, and reason-code semantics 1:1 when that logical contract is present.
- [ ] Preserve cursor pagination and ordering semantics from the logical API.
- [ ] Preserve the logical stream contract of exactly one terminal event with typed terminal failure.
- [ ] Preserve any runner->broker checkpoint/result write families and any gate-evidence-related logical families 1:1 when those logical contracts are added.

## Local IPC Transport Requirements

- [ ] Keep the transport local-only by default.
- [ ] Keep framing, limits, deadlines, streaming backpressure, and max in-flight posture explicit regardless of encoding.
- [ ] Preserve deterministic broker enforcement for size and complexity limits.
- [ ] Keep socket paths, runtime directories, named-pipe identities, and similar transport bindings out of boundary-visible product identity and out of the canonical repo-scoped product lifecycle model.

## Optional Local-Only gRPC Profile

- [ ] Define any optional local-only gRPC profile without widening the trust boundary.

## Migration and Compatibility Rules

- [ ] Keep migration from JSON encoding explicit and reviewable.
- [ ] Preserve compatibility rules for existing logical contracts and persisted objects.
- [ ] Do not merge, rename, or reinterpret logical operations as part of transport migration.
- [ ] Do not reinterpret authoritative-versus-advisory run-state semantics, approval scope semantics, or gate-attempt/evidence semantics during transport mapping.
- [ ] Do not reinterpret canonical `runecode` lifecycle semantics, repo-scoped product instance identity, or bootstrap-vs-transport boundaries during transport migration.

## Acceptance Criteria

- [ ] Protobuf stays an alternate local transport encoding rather than a new protocol.
- [ ] Local IPC trust-boundary rules and persisted RFC 8785 JCS canonicalization semantics remain unchanged.
- [ ] A protobuf client and JSON client target the same logical broker API semantics and observe the same run, approval, cursor, and stream behavior.
- [ ] Transport migration preserves the canonical repo-scoped product lifecycle and `runecode` user-surface semantics rather than leaking transport identity into product identity.
