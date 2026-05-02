# Design

## Overview
Select and deliver one narrow local zero-knowledge proof workflow with deterministic trusted verification, audit-owned authoritative persistence, explicit proof-binding sidecars, and a scheme-agnostic proof contract that later proof families can reuse.

## Key Decisions
- ZK is used for integrity attestations of deterministic computations and records, not for proving arbitrary reasoning.
- Proof generation is an explicit workflow step; verification is always deterministic.
- The first ZK proof ships only if a concrete proving system can be selected with acceptable performance; otherwise release is deferred rather than weakening the contract.
- The first proof statement is audit-bound and narrowly scoped: prove knowledge of one normalized witness bound to one audited commitment that is included in one verified `AuditSegmentSeal`, rather than proving broad policy-program execution or arbitrary event-sequence properties in `v0`.
- The proof contract is scheme-agnostic. Proof objects, verification records, and broker or audit linkage must identify `scheme_id`, `curve_id`, `circuit_id`, verifier-key identity, normalization-profile identity, and public-input identity explicitly rather than assuming one proving system forever.
- For `v0`, prefer a Go-native implementation with `gnark` and a fixed-circuit `Groth16` verifier because RuneCode currently needs one narrow statement with bounded proof size and cheap local verification. This is a performance choice for the first proof lane, not a commitment that all later proof families must use the same proving system.
- The authoritative verification boundary remains trusted Go code. The runner does not generate, verify, or interpret proof trust outcomes across the trust boundary.
- The first proof's authoritative persistence is audit-owned sidecar evidence. Artifact-store copies may exist for review or export, but they must not replace the authoritative audit-sidecar truth model already established for receipts, seals, and verification artifacts.
- `AuditProofBinding`-style sidecars are part of the intended `v0` foundation, not just a later convenience. The first proof lane should establish the additive proof-binding substrate now so later proof families can reuse it without a second semantics rewrite.
- Proof verification is supplemental assurance evidence in `v0`; it must not create a second authorization path or redefine allow, deny, or approval semantics outside the shared policy engine.
- Proof verification must not become an ambient hot-path cost in read models, watch paths, or ordinary `audit_finalize_verify` execution. Read models should consume persisted verification results, and verification results should be cached by immutable proof identity.
- Performance posture follows the same RuneCode rule used for attestation and other trusted verification work: one architecture everywhere, no smaller-device exception architecture, and no optimization that weakens trust semantics.
- If a proof statement depends on project context, it should bind the validated project-substrate snapshot identity rather than ambient repository assumptions.
- If a proof statement depends on runtime execution identity, it should bind the attested runtime identity seam established by `CHG-2026-030-98b8-isolate-attestation-v0`, using signed runtime-image descriptor identity, persisted reviewed launch evidence, and attestation evidence or verification where relevant rather than ambient platform-specific runtime assumptions.
- If a proof statement depends on external audit anchoring, it should bind the canonical `AuditSegmentSeal` subject plus authoritative anchor receipt identity, canonical target descriptor identity, and typed sidecar proof references rather than raw transport URLs, flattened target-local summaries, or exported-copy artifacts.
- Future additive remote or public proof-lane work is intentionally split into `CHG-2026-055-b7e4-additive-remote-public-proof-lane`. This change keeps only the local `v0` proof core and the local persistence foundation needed so that follow-on lane can exist later without changing local trust semantics.

## Security Model

### What The First Proof Provides
- Privacy of the normalized private session-field projection selected by the logical normalization profile.
- Knowledge of a witness that is bound to the audited `session_binding_digest` under the declared proof-binding rules.
- Membership of the audited record inside one verified `AuditSegmentSeal` by reproducing the authoritative RuneCode Merkle construction inside the proof system.

### What The First Proof Does Not Replace
- The authoritative audit ledger.
- `AuditSegmentSeal` verification.
- Audit receipt, signer-evidence, attestation, or verified project-substrate verification.
- The shared policy engine.

### Trust Model
- The prover is trusted Go code with access to authoritative audit evidence and normalized private inputs.
- Local verification is trusted Go code and remains authoritative for any `v0` proof family RuneCode supports everywhere.
- Future export-only or external verifiers may validate proof objects and public inputs, but that does not replace RuneCode's local authoritative audit verification path.
- Proof soundness for `Groth16` depends on setup integrity. A compromised setup can forge the knowledge claim for that circuit family, but it does not let an attacker rewrite the independent audit chain or invent new authoritative seal digests.

## First Proof Foundation

### Recommended `v0` Statement Family
- The recommended first proof family is one audited-commitment statement rather than a general proof of event-sequence correctness or policy-program execution.
- The exact recommended `v0` statement family is `audit.isolate_session_bound.attested_runtime_membership.v0`.
- Human meaning of the first proof statement:
  - a verified audited `isolate_session_bound` event exists inside one verified `AuditSegmentSeal`
  - that event binds to one public attested runtime identity seam
  - the proof does not reveal the full normalized private session payload needed to make the claim true

### Hard Prerequisites
- The first proof family requires `attestation_evidence_digest` to be present.
- Events without attested posture are not eligible for this initial statement family.
- This means end-to-end production proof generation for real events depends on `CHG-2026-030-98b8-isolate-attestation-v0` producing attested `isolate_session_bound` events in the authoritative audit ledger.

### Public Inputs
- Public inputs should stay bounded and typed. Recommended minimum public inputs are:
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
  - `applied_hardening_posture_digest`
  - `session_binding_digest`
  - `binding_commitment`

### Witness Inputs
- The prover witness should stay bounded and typed. Recommended minimum witness inputs are:
  - one private normalized remainder of the `IsolateSessionBoundPayload`
  - one Merkle authentication path from the audited record digest to the segment root
  - the minimum linking material required to prove the witness commits to the audited statement under the declared normalization profile

### Why This First Statement
- This statement is preferred for `v0` because it already sits on one of RuneCode's strongest existing typed assurance seams: audited isolate session binding plus attested runtime identity.
- It reuses the stable typed payload family `IsolateSessionBoundPayload` and its existing public digest bindings such as `runtime_image_descriptor_digest`, `attestation_evidence_digest`, `applied_hardening_posture_digest`, and `session_binding_digest`.
- It avoids creating a second policy semantics implementation and avoids depending on broader or less settled proof families before the first narrow proof is evaluated.

## Trusted Statement Compilation
- Trusted Go code should first verify the authoritative audit objects and then compile one small typed proof-input contract for proving and verification.
- The proof circuit must not parse full signed envelopes, arbitrary protocol JSON, or RFC 8785 JCS objects directly.
- The proof-input contract should be versioned by `normalization_profile_id` so proof meaning does not drift silently when trusted code evolves.
- Trusted Go code should compile the proof input from a verified `AuditEventPayload` with `audit_event_type = isolate_session_bound` and `event_payload_schema_id = runecode.protocol.v0.IsolateSessionBoundPayload`.
- If a proof statement later depends on verified-mode RuneContext project context, it must bind the validated project-substrate snapshot digest already produced by the verified project-substrate flow rather than inventing a second project-context identity.
- If a proof statement later depends on attested runtime evidence, it must bind the attestation evidence or attestation verification identity already established by the trusted runtime-evidence path rather than flattening runtime identity into launch-only assumptions.

## Normalization Profiles

### Two-Layer Model
- The first proof family should use a two-layer normalization model.
- Logical normalization profile:
  - scheme-agnostic
  - defines eligible source object families, required field presence, proof disclosure split, enum coding, field ordering, and missing-field rules
  - recommended first logical profile id: `runecode.zk.normalize.audit.isolate_session_bound.attested_runtime.v0`
- Scheme-adapter profile:
  - proving-system-specific
  - defines how normalized slots are packed into field elements and how the proof-friendly commitment is computed for the chosen proving backend
  - recommended first adapter profile id: `runecode.zk.adapter.gnark.groth16.isolate_session_bound_attested_runtime.v0`
- This split is the main mechanism that keeps the proof contract agnostic while allowing RuneCode to switch from `Groth16` to `PLONK` or another proving family later without rewriting the whole broker or audit contract.

### Disclosure Split Meaning
- The logical profile's public and private split defines what the proof object discloses to a verifier.
- It does not rewrite the source protocol schema's `x-data-class` meaning for authoritative source records.
- A field may be publicly visible in the authoritative audit event family while still being omitted from an exported proof's public inputs if RuneCode chooses not to disclose it through the proof contract.

### Recommended First Logical Profile Details
- Eligible source object:
  - verified `AuditEventPayload`
  - `audit_event_type = isolate_session_bound`
  - `event_payload_schema_id = runecode.protocol.v0.IsolateSessionBoundPayload`
  - event must already pass signed-envelope verification, event-contract validation, signer-evidence validation, payload-hash validation, and seal inclusion verification
- Recommended public logical fields:
  - `runtime_image_descriptor_digest`
  - `attestation_evidence_digest`
  - `applied_hardening_posture_digest`
  - `session_binding_digest`
  - `protocol_bundle_manifest_hash`
- Recommended private logical fields:
  - `run_id`
  - `isolate_id`
  - `session_id`
  - `backend_kind`
  - `isolation_assurance_level`
  - `provisioning_posture`
  - `launch_context_digest`
  - `handshake_transcript_hash`
- Variable-length identifiers should not flow into the proving backend as open-ended raw strings when a stable digest or fixed-width normalized encoding can be used instead.
- Enum-like values should be normalized to registry-owned codes rather than raw strings whenever the proof family depends on them.

### Binding Commitment Semantics
- `session_binding_digest` remains the existing audited public binding to the source event family.
- `binding_commitment` is a new proof-time derived commitment, not a source audit field.
- For `v0`, `binding_commitment` should be computed by the trusted statement-compilation step using a ZK-friendly Poseidon-family commitment over the normalized private field set selected by the logical profile and encoded by the selected scheme-adapter profile.
- Trusted Go code must verify the off-circuit relationship between the normalized private field set and the source `session_binding_digest` before emitting the `AuditProofBinding` sidecar.

## Merkle Membership Posture

### Authoritative Merkle Construction
- The first proof family should reproduce RuneCode's current ordered SHA-256 Merkle construction exactly.
- The circuit and witness path format must preserve:
  - leaf domain separation `runecode.audit.merkle.leaf.v1:`
  - node domain separation `runecode.audit.merkle.node.v1:`
  - ordered left and right sibling semantics
  - odd-leaf duplication when a node has no right sibling

### Depth Bound
- `v0` should set an explicit maximum Merkle path depth of `12`.
- Witness inputs that exceed that bound should fail closed.
- If the authoritative audit segment policy later exceeds that depth bound for the first proof family, RuneCode should defer shipping that proof family rather than silently weakening the statement.

## Proof-Binding Sidecars

### Role
- `AuditProofBinding`-style sidecars are part of the intended `v0` implementation foundation and should be produced by trusted Go after normal authoritative audit verification.
- Recommended first sidecar family purpose:
  - bind one proof family to one authoritative source record and seal identity
  - preserve one scheme-agnostic normalized statement projection
  - preserve the exact Merkle authentication path needed for the first proof family
  - allow later proof systems to consume proof-ready bindings without reparsing arbitrary historical objects differently
- The proof-binding sidecar is not the proof itself. It is additive derived evidence that stabilizes proof-input meaning across time and proving systems.

### Recommended First Sidecar Fields
- Recommended first proof-binding sidecar fields include at least:
  - `statement_family`
  - `statement_version`
  - `normalization_profile_id`
  - source `audit_record_digest`
  - source `audit_segment_seal_digest`
  - source `merkle_root`
  - `protocol_bundle_manifest_hash`
  - `binding_commitment`
  - projected public bindings such as `runtime_image_descriptor_digest`, `attestation_evidence_digest`, `applied_hardening_posture_digest`, and `session_binding_digest`
  - ordered `merkle_authentication_path`
  - `merkle_path_depth`
  - leaf position or equivalent branch-direction data required to replay the authoritative tree exactly

### Production Rules
- Trusted Go code should emit an `AuditProofBinding` sidecar only after the source audit event, source segment seal, and all required trusted verification preconditions for the selected proof family have succeeded.
- The sidecar should be immutable once persisted.
- The sidecar should be keyed by its own digest and should also retain stable references to the source `audit_record_digest` and `audit_segment_seal_digest`.
- If the selected proof family requires verified-mode project context, the sidecar should carry the validated project-substrate snapshot digest used at proof-binding time.
- If the selected proof family requires attested runtime context, the sidecar should carry the attestation evidence or attestation verification identity required by that proof family.
- Sidecar generation must fail closed on ambiguous source identity, ambiguous normalization-profile selection, or incomplete required source evidence.
- Sidecar generation must not infer missing proof inputs from filenames, directory scans, client-local caches, or other non-canonical ambient context.

## Groth16 Setup And Circuit Freeze Posture

### Circuit Freeze
- `v0` requires one reviewed frozen circuit for the first proof family before setup material is generated.
- The frozen circuit identity should be tracked by:
  - `circuit_id` for the reviewed circuit family and version
  - `constraint_system_digest` for the concrete compiled constraint system artifact

### Setup Lineage
- `v0` should reuse a well-audited existing Phase 1 Powers-of-Tau lineage for the selected curve rather than inventing a fresh RuneCode-specific Phase 1 ceremony.
- RuneCode should conduct its own explicit circuit-specific Phase 2 ceremony for the frozen first circuit and record a transcript digest for that ceremony.
- `setup_provenance_digest` should be a SHA-256 digest over a canonical JSON object that includes at least:
  - Phase 1 lineage identity and digest
  - Phase 2 transcript digest
  - frozen circuit source digest
  - `constraint_system_digest`
  - selected `gnark` module version identity

### Verifier-Key Distribution
- The verifier key must be delivered through reviewed trusted assets only.
- `v0` must not rely on runtime setup on user machines or ambient key download.
- Whether the key is compiled into trusted Go assets or loaded from a reviewed local bundle, verification must pin `verifier_key_digest` explicitly and fail closed on mismatch.

### Rotation Posture
- A future circuit fix or setup rotation should introduce a new `circuit_id`, a new `constraint_system_digest`, a new `verifier_key_digest`, and a new `setup_provenance_digest` rather than mutating the old identity.
- Proofs remain verifiable only under the exact setup and circuit identity they declare.
- Any proof whose setup identity no longer matches the trusted local verifier posture must fail closed with a stable setup-identity mismatch reason code.

## Proof Object And Verification Model

### Scheme-Agnostic Proof Contract
- The proof object family should remain scheme-agnostic even if `v0` only enables one concrete `gnark` plus `Groth16` implementation.
- Recommended required proof-object fields are:
  - `statement_family`
  - `statement_version`
  - `scheme_id`
  - `curve_id`
  - `circuit_id`
  - `constraint_system_digest`
  - `verifier_key_digest`
  - `setup_provenance_digest`
  - `normalization_profile_id`
  - `public_inputs`
  - `public_inputs_digest`
  - `proof_bytes`
  - `source_refs`
- Recommended required verification-record fields are:
  - proof digest
  - verifier implementation identity
  - `circuit_id`
  - `constraint_system_digest`
  - `verifier_key_digest`
  - `setup_provenance_digest`
  - `public_inputs_digest`
  - verification timestamp
  - verification result
  - stable reason codes including `setup_identity_mismatch`
  - cache provenance

### Verification Placement
- Trusted Go verification remains authoritative.
- Broker-owned explicit proof commands or APIs should trigger proof generation and proof verification using the same reviewed local control-plane pattern already used for audit verification surfaces.
- Proof verification should record machine-readable results in trusted persistence and audit the verification outcome.
- `audit_finalize_verify` may later surface proof-reference posture or cached proof-verification posture, but it should not become the only entrypoint for running ZK verification in `v0`.

### Verification-Result Caching
- Proof verification results should be cached by immutable identity including at least:
  - proof digest
  - `statement_family`
  - `statement_version`
  - `scheme_id`
  - `curve_id`
  - `circuit_id`
  - `constraint_system_digest`
  - `verifier_key_digest`
  - `setup_provenance_digest`
  - `public_inputs_digest`
  - `normalization_profile_id`
  - verifier implementation version
- The authoritative verification record and any acceleration cache both live in trusted persistence.
- The runner must not have direct read or write access to the verification-result persistence path.
- Cache entries may be evicted because verification is re-derivable from immutable inputs, but the authoritative audit record of a completed verification decision remains trusted persisted evidence.

## Local Persistence Foundation

### Required Local Retention Even Without Remote Lane Enablement
- Every RuneCode machine must preserve enough canonical proof-relevant source evidence and proof-binding information locally so that future proof backfill remains possible even if no remote or public proof lane is configured on that machine today.
- Disabling or not configuring a future remote or public proof feature must not allow RuneCode to skip retention of proof-relevant authoritative evidence.
- Preserving only final digests is insufficient where later proof witnesses need more than one already-compressed summary field.

### Minimum Evidence Classes
- The local foundation must preserve or be able to export at least:
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
- This preservation requirement applies equally when RuneCode runs concurrently across more than one machine on the same project.

### Future-Lane Handoff
- The detailed additive remote and public proof-lane architecture, export-bundle format, cross-machine merge rules, remote ingest, and public publication posture are captured in `CHG-2026-055-b7e4-additive-remote-public-proof-lane`.
- This local `v0` change only commits to preserving the necessary canonical evidence and proof-binding substrate so that follow-on work can be added without changing local trust semantics.

## Performance And Scaling Posture

### One Architecture Everywhere
- RuneCode must keep one reviewed proof-verification architecture across constrained and scaled environments.
- Different deployments may vary in cache population, local storage sizing, proof-generation queue depth, or background scheduling, but they must not vary in trust roots, verification semantics, or fallback behavior.

### Required Performance Shape
- Proof generation is explicit and off the interactive hot path.
- Proof generation should default to trusted local worker concurrency `1` until measured evidence demonstrates a higher safe bound on the target deployment class.
- Proof verification must be cheap enough for routine local use and cached by immutable identity.
- Read models and watch surfaces must consume persisted proof-verification results and must not rerun expensive proof verification during refresh.
- External network dependence, remote prover dependence, or device-class-specific alternate proof architectures are out of scope for `v0`.

### Recommended Initial Performance Gates
- Required Linux PR lane should include:
  - warm single-proof verify on deterministic fixture with target `<= 100ms`
  - cold single-proof verify including verifier-key load and public-input normalization with target `<= 250ms`
  - invalid-proof rejection with target `<= 50ms`
  - cache-hit proof status lookup with target `<= 10ms`
  - bounded proof size with initial target `<= 16KB`
  - bounded public-input envelope with initial target `<= 4KB`
  - warm verification peak working-set target `<= 64MB`
  - cold verification peak working-set target `<= 96MB`
- Extended Linux lane should include:
  - explicit proof generation wall-time target `<= 10s` for the canonical `v0` fixture
  - proof generation peak RSS target `<= 1GB`
  - serial verification throughput checks for `10` and `100` proof fixtures
  - concurrent verification throughput checks with bounded worker concurrency
  - end-to-end `generate -> persist -> verify -> audit record` integration checks
- Scheduled or release low-power ARM64 checks should run the same implementation path and target at least:
  - warm verify `<= 300ms`
  - cold verify `<= 750ms`
  - cache hit `<= 25ms`
  - proof generation `<= 30s`
  - proof-generation peak RSS `<= 1GB`
- If these gates cannot be met, RuneCode should defer the feature rather than weaken correctness or create a second architecture for smaller devices.

## Alternative Architecture Not Chosen For `v0`
- A possible future architecture switch is to preserve the authoritative SHA-256 audit Merkle root exactly as-is while adding an additive proof-friendly segment-binding sidecar that binds the authoritative seal and authoritative root to a second proof-friendly root over the same ordered records.
- That option is not chosen for `v0`.
- `v0` should first attempt direct in-circuit membership against the authoritative RuneCode Merkle construction.
- The detailed pros, cons, and decision posture for the additive dual-commitment architecture are tracked in `CHG-2026-055-b7e4-additive-remote-public-proof-lane` so RuneCode can revisit it later if the direct approach misses performance badly.

## Main Workstreams
- Pick the First Proof Statement
- Choose Proving System + Libraries
- Proof Artifact Format + Storage
- CLI Integration

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
- If later proof families expand deeper into RuneCode, they should preferentially reuse verified-mode RuneContext audit, project-substrate, attestation, and related typed assurance bindings rather than introducing second summary-only trust surfaces.
