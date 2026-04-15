# Protocol Schemas

- `protocol/schemas/manifest.json` is the authoritative bundle manifest for protocol object families and shared registries.
- `protocol/schemas/meta/manifest.schema.json` validates `protocol/schemas/manifest.json`.
- `protocol/schemas/meta/registry.schema.json` validates `protocol/schemas/registries/*.registry.json`.

## Status Semantics

- `mvp` means the object family is in MVP bundle scope. Some `mvp` families are intentionally narrow anchors until their owning spec task lands; those entries include a manifest `note` describing the pending task. In the current bundle, `ApprovalRequest`, `ApprovalDecision`, `PolicyDecision`, and `Error` are the main constrained MVP anchors.
- `reserved` means the family is reserved for post-MVP extension work and must not expand capabilities without a later schema/task update.

## Current Lifecycle Coverage

- `PrincipalIdentity`, `RoleManifest`, and `CapabilityManifest` now carry the shared identity and lifecycle fields needed to bind requests, approvals, and audit records to active manifest context.
- `ApprovalRequest` and `ApprovalDecision` are now signed object payload families (detached signatures over RFC 8785 JCS payload bytes), bind immutable hash inputs, enforce explicit expiry semantics, and constrain MVP approval profiles to `moderate`.
- Approval payloads include explicit assurance and posture hooks (`approval_assurance_level`, `presence_mode`, and decision-side posture fields) so trusted verification does not rely on delivery-channel semantics.
- `VerifierRecord` is the first-class verifier descriptor family for deterministic key identity, public-key metadata, logical scope/purpose, owner attribution, closed posture enums, creation metadata, and lifecycle status.
- Approval trigger codes remain registry-owned values with fail-closed runtime validation; object schemas intentionally avoid hardcoding the full registry so new codes can land without a schema family bump.
- Timestamp ordering such as `requested_at < expires_at` remains a runtime validation rule even though both timestamps are required in the serialized protocol object.
- Signature key identity is topology-neutral and deterministic: `key_id` uses the profile label `key_sha256` and `key_id_value` carries exactly 64 lowercase hex characters derived from canonical public-key bytes.
- Trusted runtime validation now treats the pair (`key_id`, `key_id_value`) as a single verifier identity and fails closed when they are inconsistent with verifier-record public keys.

## Verification Placement

- Trusted Go verification remains authoritative for trust admission and signature acceptance.
- Runner-side TypeScript schema verification remains fixture/parity/supporting behavior and is non-authoritative.

## Artifact Data Classes v0

- `ArtifactReference.data_class` is an explicit MVP taxonomy, not an open-ended free-form label.
- Current classes are:
  - `spec_text`
  - `unapproved_file_excerpts`
  - `approved_file_excerpts`
  - `diffs`
  - `build_logs`
  - `audit_events`
  - `audit_verification_report`
  - `audit_receipt_export_copy`
  - `web_query` (reserved)
  - `web_citations` (reserved)
- `web_query` and `web_citations` are reserved for future role work and remain fail-closed unless explicitly enabled by later signed-manifest policy surfaces.

## Audit Ledger + Evidence Model Foundation

- Authoritative audit truth is the `auditd`-owned instance-global append-only ledger (`AuditEvent` signed envelopes in segment files) plus signed sidecar evidence objects (`AuditReceipt`, `AuditSegmentSeal`, and `AuditVerificationReport`).
- Query/index stores are rebuildable local-read derivatives and are never a second source of truth.
- Artifact-store copies of audit evidence are optional export/review copies and do not replace ledger authority.
- Canonical audit-record identity is `sha256(JCS(SignedObjectEnvelope))`, modeled as `AuditRecordDigest` for reuse across event chaining, receipt targeting, segment first/last references, seal chaining, import/restore references, and verifier findings.
- `SignedObjectEnvelope` remains a single-signature envelope contract for this foundation. Additional attestations are modeled as separate signed objects/receipts rather than multiple independent signatures on one envelope.
- Open and sealed segment leaves are limited to signed `AuditEvent` envelopes; receipts, seals, and verification reports are sidecar evidence keyed by digest.
- Anchor receipts keep the shared `AuditReceipt(kind=anchor)` envelope and `AuditSegmentSeal` subject model. MVP runtime verification currently admits `anchor_kind=local_user_presence_signature` (with local witness kind) while schema contracts stay additive for post-MVP anchor kinds in `CHG-2026-025`.

## Append-Only Writer + Recovery Contract Foundation

- `auditd` is the authoritative append-only writer and recovery owner; trusted admission surfaces must fail closed unless all four checks succeed at write time:
  - schema validation
  - event-contract catalog validation
  - signer-evidence validation
  - detached signature verification
- Framed segment-file contract (`AuditSegmentFile`) models:
  - header (`format`, `segment_id`, lifecycle state, `auditd` writer identity)
  - repeated record frames (`record_digest`, `byte_length`, canonical signed-envelope bytes)
  - explicit lifecycle marker (`open`/`sealed`/`quarantined`) for deterministic recovery
- Recovery rules are fail-closed by contract:
  - open segments may truncate a torn trailing frame before sealing
  - sealed segments never permit silent repair
  - inconsistent sealed segments are quarantined
- Storage posture evidence must assert encrypted-at-rest default with no silent plaintext fallback; explicit dev-only degraded posture must be recorded and surfaced.
- Local readiness is broker-local-API consumption only and must include all dimensions before `ready=true`:
  - recovery complete
  - append position stable
  - current segment writable
  - verifier material available
  - derived index caught up

## Artifact Policy Family v0

- `ArtifactPolicy` provides a schema-level anchor for artifact-store and data-flow controls:
  - hash-only cross-role handoffs
  - CAS interface contract (`put/get/head`) with deterministic hashing profile
  - encrypted-at-rest-default storage posture with explicit dev-only plaintext override semantics
  - approval-promotion hardening requirements (explicit human approval, mint-new-reference posture, size/rate limits, full-content + origin metadata visibility)
  - manifest-driven producer/consumer flow matrix
  - approved-excerpt revocation denylist by artifact hash
  - per-role and per-step quotas
  - retention and deterministic GC/export/restore controls with audit requirements

## Schema Document IDs

- Object-schema `$id` values under `https://runecode.dev/protocol/schemas/...` are canonical schema identifiers for tooling and reference resolution.
- These `$id` values are not a network fetch contract. Validation and CI use the checked-in schema bundle as the source of truth.
