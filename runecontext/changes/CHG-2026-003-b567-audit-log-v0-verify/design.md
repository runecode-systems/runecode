# Design

## Overview
Create a tamper-evident audit subsystem owned by `auditd` with an instance-global append-only ledger of signed audit-event envelopes, deterministic segment sealing, authoritative local verification, and typed verification/reporting outputs that later changes can build on without changing the substrate.

## Key Decisions
- The primary audit ledger is instance-global, not per-run. Per-run audit views and export bundles are derived from the shared ledger rather than acting as the source of truth.
- `auditd` owns the authoritative append-only ledger, recovery logic, segment lifecycle, and sealing flow. SQLite or similar query indexes are derived and rebuildable only.
- Audit payload families such as `AuditEvent` and `AuditReceipt` are unsigned protocol payloads wrapped in `SignedObjectEnvelope`; detached signatures cover the RFC 8785 JCS payload bytes.
- `SignedObjectEnvelope` remains single-signature in this foundation. When multiple attestations are needed, RuneCode emits separate signed objects or receipts rather than accumulating multiple independent attestations on one envelope.
- The canonical identity of an audit record is the SHA-256 digest of the RFC 8785 JCS bytes of the full signed envelope. This digest is reused for chain links, segment leaves, receipt targeting, import references, and verification output.
- Chain continuity is scoped to a stable emitter stream identity rather than directly to a signing key. Key rotation must not break the logical event chain.
- Audit segments contain signed `AuditEvent` envelopes only. `AuditReceipt`, `AuditSegmentSeal`, and later verification artifacts are sidecar evidence objects that reference segment or record digests rather than becoming leaves in the same segment they attest.
- Segment sealing is a first-class signed object (`AuditSegmentSeal`), not an in-band event appended to the segment being sealed.
- Segment roots use an ordered Merkle construction over signed record digests with domain-separated leaf and internal-node hashing, plus a separate exact segment-file hash over the raw framed bytes.
- Segment seals chain linearly via the previous segment-seal digest so ledger history remains explainable across retained, archived, imported, and restored segments.
- Audit storage must remain encrypted-at-rest by default with no silent plaintext fallback; storage protection posture is recorded and surfaced as audit evidence.
- Verification is authoritative in trusted Go code, fails closed on invalid signatures, chain breaks, malformed segments, or invalid receipts, and distinguishes integrity failure from degraded-but-still-explainable posture.
- Missing anchors are not an integrity failure by default; they are reported as degraded posture until later policy requires stronger anchoring for selected workflows.
- Machine-readable verification output is a first-class protocol family and artifact data class (`audit_verification_report`), not just human-oriented CLI output.
- Audit views must support later policy, TUI, gateway, secrets, concurrency, import/restore, and attestation work without needing schema rewrites for object references or degraded-posture reasons.

## Ledger Model

### Authoritative Layers
- Authoritative source of truth:
  - append-only segment files owned by `auditd`
  - signed `AuditSegmentSeal` objects
  - signed `AuditReceipt` sidecar evidence objects
- Derived/query layers:
  - rebuildable local index storage for timeline, run, stage, and posture queries
  - machine-readable verification reports stored as artifacts for later review
- Export/review layers:
  - optional run-scoped or segment-scoped export bundles
  - optional artifact-store copies of sealed audit evidence for inspection or transport

### Segment Contents
- Open and sealed segments contain only framed signed `AuditEvent` envelopes.
- Segments are immutable once sealed.
- Receipt and seal objects are stored alongside the ledger as separate signed evidence objects keyed by digest.

### Segment Lifecycle
- Segment lifecycle states are explicit: `open`, `sealed`, `anchored`, `imported`, and `quarantined`.
- `open` segments may recover from a torn trailing frame after crash/restart.
- `sealed` segments are immutable and any structural inconsistency fails closed.
- `anchored` indicates one or more valid receipts over the segment seal or its root commitment.
- `imported` indicates the segment originated from backup/import rather than being first recorded locally.
- `quarantined` indicates the segment cannot be admitted as valid ledger history until operator action resolves the inconsistency.

## Audit Object Model

### Signed Event Shape
- `AuditEvent` becomes an unsigned payload family wrapped in `SignedObjectEnvelope`.
- `AuditEvent` retains payload and event metadata, but not inline `signatures`.
- `previous_event_hash` points to the digest of the previous signed event envelope in the same emitter stream.
- `seq` is strictly monotonic per emitter stream.
- `occurred_at` is the emitter timestamp and remains advisory metadata; append order is the authoritative ledger chronology.
- Event-level manifest and bundle binding must use exact hash references for trust-relevant context. Advisory version strings may still be surfaced for UX or diagnostics, but verification and receipt linkage rely on immutable manifest and bundle hashes rather than version labels alone.

### Stream Identity
- Each signer-visible chain uses a stable `emitter_stream_id` or equivalent chain identity.
- Stream identity survives key rotation so later verifier rules can validate chain continuity across historical and active verifier records.
- Verifier rules bind an event's signing key to the stream identity and reject unexpected signer-to-stream changes unless justified by admissible signer evidence or rotation state.

### Common Structure
- `AuditEvent` includes a reusable `scope` block for workspace/run/stage/step correlation.
- `AuditEvent` includes a reusable `correlation` block for session/operation and parent-operation correlation.
- `AuditEvent` carries explicit manifest/bundle bindings for the exact active trust context when applicable, including active role/capability manifest hashes and protocol bundle manifest hash or equivalent immutable contract reference.
- `AuditEvent` uses a typed-reference model instead of parallel `related_*_hashes` arrays:
  - `subject_ref`
  - `cause_refs`
  - `related_refs`
- Typed refs identify the referenced object family, digest, and reference role so later policy, approval, verifier, segment, artifact, lease, and import objects can all be linked without schema rewrites.
- `AuditEvent` carries `signer_evidence_refs` when signer admissibility depends on prior trusted evidence such as isolate-session bindings or later attestation objects.

### Payload Governance
- `audit_event_type` remains a stable machine code registry.
- A separate machine-readable audit-event contract catalog binds each event type to:
  - allowed payload schema IDs and version posture
  - allowed signer purposes/scopes
  - required scope/correlation fields
  - allowed or required gateway context
  - admissible subject/cause reference shapes
- Payload schema IDs and receipt payload schema IDs must resolve to checked-in protocol schemas; unknown payload families fail closed.

### Receipts And Seals
- `AuditReceipt` becomes an unsigned payload family wrapped in `SignedObjectEnvelope`.
- `AuditReceipt` targets a generic signed-object subject rather than only one event digest.
- Receipt kinds move to an explicit `audit_receipt_kind` registry so anchoring, import, restore, and later reconciliation receipts stay typed and reviewable.
- `AuditSegmentSeal` is introduced as a new signed payload family carrying:
  - segment identity and lifecycle posture
  - first/last event digests
  - event count
  - ordered Merkle root
  - exact raw segment-file hash
  - previous segment-seal digest
  - protocol bundle manifest hash / equivalent contract binding
  - signer/evidence summary needed for deterministic replay and explanation

## Physical Segment Format
- Segment files use an explicit framed format rather than ad-hoc newline parsing.
- The format includes a segment header and repeated record frames containing:
  - canonical record digest
  - byte length
  - canonical signed-envelope bytes
- Canonical signed-envelope bytes are RFC 8785 JCS bytes of the persisted `SignedObjectEnvelope` object.
- Open-segment recovery may truncate an incomplete trailing frame only before the segment is sealed.
- Sealed segments never permit partial-frame repair; any mismatch between frame digest, canonical bytes, or sealed metadata quarantines the segment and fails verification closed.

## Verification Model

### Authoritative Verification
- Trusted Go verification remains authoritative and is used by `auditd`, the broker, and any trusted daemon boundary that consumes audit evidence.
- Verification checks:
  - segment framing and file-hash integrity
  - ordered Merkle root and seal correctness
  - per-stream sequence monotonicity and chain continuity
  - detached signature validity against admissible verifier records
  - signer evidence validity for signers whose authority depends on prior binding evidence
  - payload schema validity and event-contract catalog compatibility
  - receipt validity when receipts are present
  - import/restore provenance consistency when imported segments or receipts are involved

### Verification Output
- `AuditVerificationReport` is a first-class protocol object stored as `audit_verification_report`.
- Verification output separates status dimensions rather than collapsing everything into one result:
  - integrity status
  - anchoring status
  - storage posture status
  - segment lifecycle posture
  - degraded reasons
  - hard failures
- Machine-readable findings use stable reason codes and severities so policy, TUI, and later formal/proof work can consume them without scraping prose.

### Historical Validity
- Verification distinguishes:
  - cryptographically valid
  - historically admissible at event time
  - currently degraded or revoked by later verifier posture
- Rotation, revocation, or compromise metadata must not rewrite history; they change future acceptance posture and may tighten historical explanations when `suspect_since` or equivalent evidence warrants it.

## Redaction And Audit Views
- Secrets are kept out of audit payloads by construction rather than by best-effort post-hoc scrubbing.
- Default operational views may redact sensitive fields, but the signed payload contract still records enough typed evidence for later trusted verification.
- Gateway context and secrets-lease lifecycle events must remain attributable without logging secret values.

## Dependencies And Follow-On Alignment
- `CHG-2026-004-acdb-artifact-store-data-classes-v0/` consumes audit verification reports and optional export copies, but does not own the authoritative ledger.
- `CHG-2026-005-cfd0-crypto-key-management-v0/` supplies signer/verifier identity, key-rotation, and signer-evidence rules used by audit verification.
- `CHG-2026-006-84f0-audit-anchoring-v0/` anchors `AuditSegmentSeal` commitments rather than raw ad-hoc file hashes.
- `CHG-2026-008-62e1-broker-local-api-v0/` exposes derived audit views and readiness, not alternate sources of truth.
- `CHG-2026-013-d2c9-minimal-tui-v0/` consumes machine-readable verification status and findings.
- `CHG-2026-027-71ed-workflow-concurrency-v0/` depends on the instance-global ledger and shared scope/correlation model for concurrent run visibility.
- `CHG-2026-030-98b8-isolate-attestation-v0/` upgrades signer-evidence validation without replacing the event/segment substrate.

## Main Workstreams
- Audit Ledger + Evidence Model
- Typed Audit Event Contract
- Append-Only Writer + Recovery Rules
- Segment Sealing + Lifecycle Model
- Verification + Machine-Readable Findings
- Redaction Boundaries + Audit Views

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, typed contracts, or trusted state, the plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
