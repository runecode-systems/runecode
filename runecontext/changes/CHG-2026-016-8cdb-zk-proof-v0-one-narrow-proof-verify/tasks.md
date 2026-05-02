# Tasks

## Pick the First Proof Statement

- [ ] Select one MVP proof type and freeze it as one audit-bound statement family rather than a broad proof lane.
- [ ] Recommended exact `v0` statement family: `audit.isolate_session_bound.attested_runtime_membership.v0`.
- [ ] Recommended `v0` statement meaning: prove that one verified audited `isolate_session_bound` event exists inside one verified `AuditSegmentSeal`, and that the event binds to one public attested runtime identity seam without revealing the full private session payload.
- [ ] Keep the first proof audit-bound rather than proving policy-program execution directly.
- [ ] Keep public inputs bounded and typed, including at least:
  - `statement_family`
  - `statement_version`
  - `normalization_profile_id`
  - `scheme_adapter_id`
  - `audit_segment_seal_digest`
  - `merkle_root`
  - `audit_record_digest`
  - `protocol_bundle_manifest_hash`
  - `runtime_image_descriptor_digest`
  - `attestation_evidence_digest`
  - `session_binding_digest`
  - `binding_commitment`
- [ ] Keep witness inputs bounded and typed, including the normalized private remainder of the `IsolateSessionBoundPayload` plus one Merkle authentication path.
- [ ] Add an explicit feasibility gate:
  - the statement must have bounded inputs and fully deterministic verification
  - if proof generation or verification performance is not acceptable, defer release rather than weakening the proof contract
- [ ] Add a trusted statement-compilation step in Go that derives one small typed proof-input contract from already-verified trusted objects.
- [ ] Restrict the first proof family to verified `AuditEventPayload` records where:
  - `audit_event_type = isolate_session_bound`
  - `event_payload_schema_id = runecode.protocol.v0.IsolateSessionBoundPayload`
  - `attestation_evidence_digest` is present
- [ ] Keep the proof circuit from directly parsing full signed envelopes, arbitrary protocol JSON, or ambient repository state.
- [ ] When the chosen statement depends on project context, bind it to validated project-substrate snapshot identity rather than ambient repo state.
- [ ] When the chosen statement depends on runtime execution identity, bind it to the attested runtime identity seam from `CHG-2026-030-98b8-isolate-attestation-v0`, using signed runtime-image descriptor identity, persisted reviewed launch evidence, and attestation evidence or verification where relevant rather than ambient platform-specific runtime state.
- [ ] When the chosen statement depends on external audit anchoring, bind it to canonical `AuditSegmentSeal` identity, authoritative anchor receipt identity, canonical target descriptor identity where relevant, and typed sidecar proof references rather than flattened summaries, raw transport details, or exported-copy artifacts.
- [ ] If proof families later expand deeper into RuneCode, reuse verified-mode RuneContext audit, project-substrate, attestation, and related typed assurance identities rather than introducing summary-only or ambient trust surfaces.

## Normalization Profiles

- [ ] Define a scheme-agnostic logical normalization profile for the first proof family.
- [ ] Recommended first logical profile id: `runecode.zk.normalize.audit.isolate_session_bound.attested_runtime.v0`.
- [ ] Define a scheme-adapter profile for the first proving backend without letting adapter details leak into the durable proof contract.
- [ ] Recommended first adapter profile id: `runecode.zk.adapter.gnark.groth16.isolate_session_bound_attested_runtime.v0`.
- [ ] Define the public logical field set for the first profile:
  - `runtime_image_descriptor_digest`
  - `attestation_evidence_digest`
  - `session_binding_digest`
  - `protocol_bundle_manifest_hash`
- [ ] Define the private logical field set for the first profile:
  - `run_id`
  - `isolate_id`
  - `session_id`
  - `backend_kind`
  - `isolation_assurance_level`
  - `provisioning_posture`
  - `launch_context_digest`
  - `handshake_transcript_hash`
- [ ] Normalize variable-length identifiers and enum-like values into stable proof-friendly representations rather than open-ended raw strings where practical.

Parallelization: can be done in parallel with audit/artifact specs; keep the chosen statement aligned with the canonical audit root/verification artifacts.

## Choose Proving System + Libraries

- [ ] Choose a pragmatic proving approach for MVP.
- [ ] Keep the proof contract scheme-agnostic even if `v0` uses one concrete implementation.
- [ ] For `v0`, prefer `gnark` in trusted Go with a fixed-circuit `Groth16` verifier if the performance targets are met.
- [ ] Treat `Groth16` as a `v0` performance choice, not a forever-global proving-system commitment.
- [ ] Accept circuit-specific setup for `v0` only under strict controls:
  - one frozen reviewed circuit
  - one explicit setup-provenance lineage
  - one immutable verifier-key digest
  - no runtime setup on user machines
  - no ambient key download
- [ ] Pin dependency versions, wrap external library usage behind local trusted interfaces, and treat version drift as security-sensitive.
- [ ] Ensure verification is fast and deterministic.
- [ ] Define MVP performance targets before implementation:
  - verification must be fast enough for routine use (target: sub-second; ideally sub-100ms)
  - proof artifacts must have bounded size
  - proof generation must not dominate the run (or must be explicitly opt-in)
- [ ] Adopt concrete initial verification targets:
  - warm verify `<= 100ms` on required Linux CI fixture
  - cold verify `<= 250ms` on required Linux CI fixture
  - invalid-proof reject `<= 50ms` on required Linux CI fixture
  - cache-hit verification status lookup `<= 10ms`
- [ ] Adopt low-power ARM64 scheduled targets using the same implementation path:
  - warm verify `<= 300ms`
  - cold verify `<= 750ms`
  - cache hit `<= 25ms`
- [ ] Adopt explicit proof-generation targets for the canonical `v0` fixture:
  - proof generation `<= 10s` on extended Linux CI
  - proof generation `<= 30s` on scheduled low-power ARM64
- [ ] Keep proof verification cached by immutable identity and out of watch or read-model refresh hot paths.

Parallelization: can be evaluated in parallel with other later hardening work; treat library selection as security-sensitive.

## Proof Artifact Format + Storage

- [ ] Define a scheme-agnostic proof object type with at least:
  - statement family/version
  - scheme id
  - curve id
  - circuit id
  - verifier-key digest
  - setup-provenance digest
  - normalization-profile id
  - public inputs
  - public-inputs digest
  - proof bytes
  - source refs
- [ ] Define a separate proof-verification record type with verifier implementation identity, proof identity, verification outcome, and stable reason codes.
- [ ] Define an `AuditProofBinding`-style sidecar family as part of the intended `v0` foundation, as additive canonical derived evidence and not as the proof itself.
- [ ] Recommended first proof-binding sidecar fields:
  - `statement_family`
  - `statement_version`
  - `normalization_profile_id`
  - source `audit_record_digest`
  - source `audit_segment_seal_digest`
  - `protocol_bundle_manifest_hash`
  - `binding_commitment`
  - projected public bindings such as `runtime_image_descriptor_digest`, `attestation_evidence_digest`, and `session_binding_digest`
- [ ] Emit the proof-binding sidecar only after the source audit event, source segment seal, and all required proof-family preconditions verify successfully.
- [ ] Keep the proof-binding sidecar immutable once persisted.
- [ ] Key the proof-binding sidecar by its own digest while retaining stable references to the source `audit_record_digest` and `audit_segment_seal_digest`.
- [ ] If a proof family requires verified project context or attested runtime context, include the required typed assurance identities directly in the proof-binding sidecar rather than reconstructing them later from ambient context.
- [ ] Keep the first proof family's authoritative persistence as audit-owned sidecar evidence.
- [ ] Keep artifact-store copies optional review/export products rather than the primary trust source.
- [ ] If a proof-export artifact data class is introduced, use a proof-specific class rather than overloading existing audit-report classes.
- [ ] Record proof-generation and proof-verification outcomes in the audit chain.

## Canonical Evidence Preservation

- [ ] Define the minimum canonical source-evidence set that every RuneCode machine must preserve or export so later proof backfill remains possible.
- [ ] Preserve enough canonical source evidence and proof-ready binding information that a later operator-private remote proof service can reconstruct witnesses from archived authoritative evidence rather than from ambient local process state.
- [ ] Treat this preservation requirement as mandatory even before any remote proof lane exists, so the future lane can be enabled later and backfill preserved history.
- [ ] Preserve or export at least:
  - raw sealed audit segments
  - signed `AuditSegmentSeal` envelopes
  - signed `AuditReceipt` sidecars
  - audit verification reports
  - signer evidence and verifier records needed for historical verification
  - immutable runtime evidence
  - attestation evidence and attestation verification records
  - validated RuneContext project-substrate snapshot digests and related proof-relevant bindings when project context matters
  - policy decisions
  - action request identities
  - approval identities
  - protocol bundle manifest hashes
  - proof-binding sidecars or equivalent proof-ready normalized bindings for proof-relevant records
- [ ] Keep authoritative evidence preservation resilient enough that canonical proof-relevant source evidence has little to no chance of being lost through ordinary restart, recovery, retention, backup, or multi-machine project operation.
- [ ] Preserve canonical source evidence rather than relying on final digests alone where later proof witnesses may need richer historical inputs.
- [ ] Capture the requirement that concurrent RuneCode execution across more than one machine on the same project must still preserve enough shared canonical evidence for later cross-machine historical proof backfill.
- [ ] Define export-bundle expectations for future proof backfill, including canonical evidence, proof-binding sidecars, and bundle authenticity material needed for remote ingest without additional ambient context.

## Evaluation And Check-In Gate

- [ ] Implement and verify the first narrow proof family end-to-end before broadening the ZK roadmap work.
- [ ] Evaluate whether the first proof materially improves RuneCode's assurance story relative to its implementation and runtime cost.
- [ ] Evaluate whether the first proof meets the documented performance gates on required Linux CI and scheduled low-power ARM64.
- [ ] Evaluate whether `gnark` plus `Groth16` remains the right `v0` proving choice after real end-to-end measurement.
- [ ] Check in with the user after the first proof evaluation before expanding into broader proof-binding, evidence-preservation hardening, or future remote-proof-lane preparation tasks.

## Future Remote Proof Lane Preparation

- [ ] Capture the future additive dual-lane roadmap explicitly:
  - local high-performance proof core available everywhere RuneCode runs
  - optional operator-private remote proof service for broader or faster-evolving proof families and history backfill
- [ ] Keep the future remote proof lane additive and asynchronous rather than a required replacement for local correctness.
- [ ] Keep future remote-proof ingest based on exported canonical evidence bundles plus proof-binding sidecars or equivalent proof-ready bindings.
- [ ] Preserve compatibility with a later externally consumable or public-assurance publication lane that reuses the same proof bindings rather than introducing a second public-only binding model.
- [ ] Keep the local lane and remote lane bound to the same statement families, logical normalization profiles, and canonical assurance identities even if they eventually use different proving backends.
- [ ] Record explicitly that this change intends to implement only the local proof core end-to-end; the remote lane is a follow-on change that must reuse the preserved evidence and proof-binding foundation from this change.

Parallelization: can be implemented in parallel with artifact store and audit log work; it depends on stable proof artifact schemas.

## CLI Integration

- [ ] Add commands to:
  - generate proof for a supported audited statement
  - verify a proof artifact
- [ ] Keep broker/API commands explicit and trusted rather than ambient background work.
- [ ] Keep proof verification out of ordinary TUI/watch/read-model refresh paths.

Parallelization: can be implemented in parallel with TUI/CLI work.

## Acceptance Criteria

- [ ] At least one proof type can be generated and verified end-to-end.
- [ ] Proof verification is deterministic, recorded in the audit log, and failure is non-destructive (it flags the run).
- [ ] The first proof statement is audit-bound, uses bounded typed inputs, and does not create a second policy or project-truth surface.
- [ ] The exact `v0` proof family is documented and bound to one attested `isolate_session_bound` audited event shape plus one verified `AuditSegmentSeal` inclusion path.
- [ ] The first proof family has one logical normalization profile and one initial scheme-adapter profile documented explicitly.
- [ ] The proof contract remains scheme-agnostic even though `v0` uses one concrete proving system.
- [ ] Authoritative proof persistence for the first proof follows the audit-sidecar truth model, with artifact-store copies remaining optional derivatives.
- [ ] Proof-binding sidecars or equivalent additive proof-ready derived evidence are part of the planned foundation for later proof backfill and proving-system agility.
- [ ] `AuditProofBinding`-style sidecars are part of the intended `v0` implementation foundation, not merely a later recommendation.
- [ ] Verified-mode RuneContext bindings are reused whenever project-context-sensitive, attestation-sensitive, or later assurance-sensitive proof families expand deeper into RuneCode.
- [ ] The same proof-verification architecture and trust semantics run on constrained and scaled deployments, with performance differences handled by caching and scheduling rather than by separate architectures.
- [ ] The change explicitly captures the requirement to preserve enough canonical proof-relevant source evidence for later operator-private remote proof backfill without making that remote lane a prerequisite for local correctness.
- [ ] The change explicitly captures enough detail about the local/remote dual-lane boundary, bundle ingest expectations, and shared binding rules that another developer can plan the follow-on remote lane without further product clarification.
- [ ] The change explicitly requires an evaluation-and-user-check-in gate after the first proof is implemented and measured.
- [ ] If performance targets cannot be met with a concrete proving system, this capability is deferred to a later release rather than weakening core deliverables.
