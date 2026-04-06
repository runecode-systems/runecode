# Tasks

## Audit Ledger + Evidence Model

- [ ] Define the authoritative audit storage model:
  - instance-global append-only ledger owned by `auditd`
  - authoritative segment files + sidecar evidence objects as the source of truth
  - rebuildable query/index storage only for local reads and timeline views
  - optional export/review copies in the artifact store without making CAS the primary ledger
- [ ] Define the canonical audit-record digest as `sha256(JCS(SignedObjectEnvelope))` and reuse it consistently for:
  - `previous_event_hash`
  - Merkle leaves
  - receipt subject targeting
  - segment first/last record references
  - import/restore provenance references
  - verification findings and reports
- [ ] Define the signed audit object families used by this change:
  - `AuditEvent` payload family wrapped in `SignedObjectEnvelope`
  - `AuditReceipt` payload family wrapped in `SignedObjectEnvelope`
  - `AuditSegmentSeal` payload family wrapped in `SignedObjectEnvelope`
  - `AuditVerificationReport` protocol object stored as `audit_verification_report`
- [ ] Keep `SignedObjectEnvelope` single-signature for this foundation and model additional attestations as separate signed objects or receipts rather than multiple independent signatures on one envelope.
- [ ] Keep open and sealed segments limited to signed `AuditEvent` envelopes only.
- [ ] Store receipts, segment seals, and verification reports as first-class sidecar evidence keyed by digest rather than in-band segment leaves.

Parallelization: foundational; this must be settled before writer, verifier, or anchoring implementation branches diverge.

## Typed Audit Event Contract

- [ ] Update `AuditEvent` to remove inline `signatures` and treat it as a pure payload family.
- [ ] Define chain continuity on a stable emitter-stream identity rather than directly on a signing key:
  - add `emitter_stream_id` or equivalent
  - keep `seq` strictly monotonic per stream
  - define verifier rules for gaps, duplicates, rollbacks, unexpected signer changes, and admissible rotation
- [ ] Treat `previous_event_hash` as the digest of the previous signed event envelope in the same emitter stream.
- [ ] Keep `occurred_at` as signed emitter time but make append order the authoritative ledger chronology.
- [ ] Bind trust-relevant event context with exact hashes rather than advisory version labels alone:
  - active role/capability manifest hashes where applicable
  - protocol bundle manifest hash or equivalent immutable bundle contract reference
  - keep version strings optional/advisory for UX and diagnostics only
- [ ] Replace parallel `related_*_hashes` arrays with a typed-reference model:
  - `subject_ref`
  - `cause_refs`
  - `related_refs`
- [ ] Add shared context blocks that later features can reuse without schema churn:
  - `scope` for workspace/run/stage/step context
  - `correlation` for session/operation/parent-operation context
- [ ] Add `signer_evidence_refs` for signers whose admissibility depends on prior trusted evidence such as isolate-session bindings or later attestation.
- [ ] Keep audit events gateway-role aware so network activity is attributable without logging secrets:
  - model egress events
  - auth egress events in `runecontext/changes/CHG-2026-018-5900-auth-gateway-role-v0/`
  - later gateway specs (`runecontext/changes/CHG-2026-002-33c5-git-gateway-commit-push-pr/`, `runecontext/changes/CHG-2026-023-59ac-web-research-role/`, `runecontext/changes/CHG-2026-024-acde-deps-fetch-offline-cache/`) extend the same event contract with typed payloads
- [ ] Record secrets lease lifecycle events as first-class audit events without logging secret values.
- [ ] Define isolate session/binding events so MVP TOFU posture is explicit and later attestation can upgrade the same model rather than replace it.
- [ ] Add or reference a machine-readable audit-event contract catalog that binds each `audit_event_type` to:
  - allowed payload schema IDs
  - allowed signer purposes/scopes
  - required scope/correlation fields
  - allowed/required subject and cause refs
  - gateway context rules

Parallelization: can proceed alongside schema-bundle and crypto work once the detached-event contract is locked.

## Append-Only Writer + Recovery Rules

- [ ] Implement `auditd` as the append-only writer and recovery owner for the authoritative ledger.
- [ ] Enforce schema validation, event-contract validation, signer-evidence validation, and signature verification at write/admission time.
- [ ] Define the framed physical segment format:
  - segment header
  - repeated record frames containing record digest, byte length, and canonical signed-envelope bytes
  - explicit open/sealed lifecycle markers or metadata needed for deterministic recovery
- [ ] Define crash-recovery rules:
  - open segments may truncate a torn trailing frame before sealing
  - sealed segments never permit silent repair
  - inconsistent sealed segments are quarantined and fail closed
- [ ] Store audit data on encrypted-at-rest storage by default and record storage protection posture as audit evidence.
- [ ] Do not silently fall back to plaintext; any explicit dev-only degraded posture must be recorded and surfaced.
- [ ] Expose a local-only readiness signal consumable via the broker local API for supervision and TUI status.
- [ ] Define readiness as more than process liveness:
  - recovery complete
  - append position stable
  - current segment writable
  - verifier material available
  - derived index sufficiently caught up for reads

Parallelization: can proceed in parallel with verifier implementation once the physical segment contract and signer-evidence contract are fixed.

## Segment Sealing + Lifecycle Model

- [ ] Define segment-cutting rules for MVP using explicit size and/or time windows, not per-run ownership of the primary ledger.
- [ ] Introduce `AuditSegmentSeal` as a first-class signed object emitted after a segment is closed.
- [ ] Compute segment roots using an ordered Merkle construction over signed record digests with:
  - domain-separated leaf and internal-node hashing
  - deterministic ordering by append position
  - no ad-hoc unordered-set semantics
- [ ] Compute and record a separate exact `segment_file_hash` over the raw framed segment bytes.
- [ ] Chain segment seals via previous-seal digest so segment history remains explainable across retention, archival, import, and restore.
- [ ] Define segment lifecycle states explicitly:
  - `open`
  - `sealed`
  - `anchored`
  - `imported`
  - `quarantined`
- [ ] Keep segment seals and receipts outside the segment they attest to avoid sealing recursion.
- [ ] Use segment seals as the anchoring target for receipts in `runecontext/changes/CHG-2026-006-84f0-audit-anchoring-v0/`.

Parallelization: can proceed alongside anchoring design and verifier work once the Merkle and seal contract is frozen.

## Audit Receipts + Import/Restore Provenance

- [ ] Update `AuditReceipt` to remove inline `signatures` and target generic signed-object subjects rather than only one event digest.
- [ ] Define and register `audit_receipt_kind` values needed by the foundation now:
  - `anchor`
  - `import`
  - `restore`
  - `reconciliation`
- [ ] Record import/restore as explicit signed evidence and audit events without rewriting imported segment bytes.
- [ ] Ensure imported historical segments remain byte-identical sealed units while the current instance records how those segments entered local history.
- [ ] Define provenance links from import/restore evidence to:
  - imported segment seal digests
  - imported segment roots
  - source backup/export manifest digests
  - operator/authority context where applicable

Parallelization: can proceed in parallel with artifact backup/restore work so long as byte-identity and no-history-rewriting rules stay aligned.

## Redaction Boundaries + Audit Views

- [ ] Define what is always excluded or redacted in the default operational audit view.
- [ ] Ensure secrets never cross trust boundaries by construction, not only by best-effort scrubbing.
- [ ] Prefer typed allowlists and schema field classification over heuristic redaction.
- [ ] Keep enough typed evidence for later trusted verification even when default operator views redact sensitive fields.

Parallelization: can proceed with schema and TUI work once common payload and field-classification rules are stable.

## Verification + Machine-Readable Findings

- [ ] Implement a deterministic verifier that checks:
  - segment framing and exact file-hash integrity
  - ordered Merkle root and seal correctness
  - per-stream chain continuity and sequence monotonicity
  - detached signature validity
  - signer-evidence admissibility
  - event-contract catalog compatibility
  - import/restore provenance consistency
  - receipt validity when receipts are present
- [ ] Distinguish verification dimensions instead of collapsing everything into one result:
  - integrity status
  - anchoring status
  - storage posture status
  - segment lifecycle posture
  - degraded reasons
  - hard failures
- [ ] Define a machine-readable findings model with stable reason codes and severities so policy, TUI, and later formal/proof work can consume verifier output directly.
- [ ] Distinguish:
  - cryptographically valid
  - historically admissible at event time
  - currently degraded or revoked by later trust posture
- [ ] If anchor receipts are present, validate them and surface anchored vs unanchored status.
- [ ] Missing anchors are reported as degraded posture by default; invalid anchors fail closed.
- [ ] Produce and store a machine-readable `AuditVerificationReport` artifact (`audit_verification_report`).
- [ ] Attach verification status/finding summaries to derived run metadata so the TUI and local API can surface clear posture.

Parallelization: can proceed in parallel with writer and anchoring work once signed-object, segment-seal, and findings contracts are defined.

## Acceptance Criteria

- [ ] The authoritative audit ledger is instance-global, append-only, and owned by `auditd` rather than by per-run artifact copies.
- [ ] `AuditEvent`, `AuditReceipt`, and `AuditSegmentSeal` use detached RFC 8785 JCS-signed envelopes with a consistent canonical digest contract.
- [ ] Typed references, scope/correlation fields, and signer-evidence refs are sufficient for later policy, approval, gateway, import/restore, and attestation work without another audit schema rewrite.
- [ ] Segment sealing uses an ordered Merkle root plus exact file hash and survives archival/import without breaking verification.
- [ ] Verification runs locally, fails closed on invalid evidence, and distinguishes hard failure from degraded posture with machine-readable findings.
- [ ] Verification output is storable and reviewable as a first-class artifact for later API/TUI use.
- [ ] Missing anchors are surfaced as degraded posture by default; invalid anchors fail closed.
