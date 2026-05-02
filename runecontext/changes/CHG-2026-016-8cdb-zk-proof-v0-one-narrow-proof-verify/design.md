# Design

## Overview
Select and deliver one narrow zero-knowledge proof workflow with deterministic trusted verification, audit-owned authoritative persistence, and a scheme-agnostic proof contract that later proof families can reuse.

## Key Decisions
- ZK is used for integrity attestations of deterministic computations/records, not for proving arbitrary reasoning.
- Proof generation is an explicit workflow step; verification is always deterministic.
- The first ZK proof ships only if a concrete proving system can be selected with acceptable performance; otherwise release is deferred rather than weakening the contract.
- The first proof statement is audit-bound and narrowly scoped: prove knowledge of one private normalized witness bound to one audited commitment that is included in one verified `AuditSegmentSeal`, rather than proving broad policy-program execution or arbitrary event-sequence properties in `v0`.
- The proof contract is scheme-agnostic. Proof objects, verification records, and broker/audit linkage must identify `scheme_id`, `curve_id`, `circuit_id`, verifier-key identity, normalization-profile identity, and public-input identity explicitly rather than assuming one proving system forever.
- For `v0`, prefer a Go-native implementation with `gnark` and a fixed-circuit `Groth16` verifier because RuneCode currently needs one narrow statement with small proofs and cheap local verification. This is a performance choice for the first proof lane, not a commitment that all later proof families must use the same proving system.
- `Groth16` setup posture for `v0` is accepted only under strict controls: one frozen reviewed circuit, one explicit setup-provenance lineage, one immutable verifier-key digest, no runtime setup on user machines, no ambient key download, and fail-closed verification on any setup-identity mismatch.
- The authoritative verification boundary remains trusted Go code. The runner does not generate, verify, or interpret proof trust outcomes across the trust boundary.
- The first proof's authoritative persistence is audit-owned sidecar evidence. Artifact-store copies may exist for review or export, but they must not replace the authoritative audit-sidecar truth model already established for receipts, seals, and verification artifacts.
- `AuditProofBinding`-style sidecars are part of the intended `v0` foundation, not just a later convenience. The first proof lane should establish the additive proof-binding substrate now so later proof families and later remote proof backfill do not require a second semantics rewrite.
- Proof verification is supplemental assurance evidence in `v0`; it must not create a second authorization path or redefine allow/deny/approval semantics outside the shared policy engine.
- Proof verification must not become an ambient hot-path cost in read models, watch paths, or ordinary `audit_finalize_verify` execution. Read models should consume persisted verification results, and verification results should be cached by immutable proof identity.
- Performance posture follows the same RuneCode rule used for attestation and other trusted verification work: one architecture everywhere, no smaller-device exception architecture, and no optimization that weakens trust semantics.
- If a proof statement depends on project context, it should bind the validated project-substrate snapshot identity rather than ambient repository assumptions.
- If a proof statement depends on runtime execution identity, it should bind the attested runtime identity seam established by `CHG-2026-030-98b8-isolate-attestation-v0`, using signed runtime-image descriptor identity, persisted reviewed launch evidence, and attestation evidence or verification where relevant rather than ambient platform-specific runtime assumptions.
- If a proof statement depends on external audit anchoring, it should bind the canonical `AuditSegmentSeal` subject plus authoritative anchor receipt identity, canonical target descriptor identity, and typed sidecar proof references rather than raw transport URLs, flattened target-local summaries, or exported-copy artifacts.

## First Proof Foundation

### Recommended `v0` Statement Family
- The recommended first proof family is one audited-commitment statement rather than a general proof of event-sequence correctness or policy-program execution.
- The exact recommended `v0` statement family is `audit.isolate_session_bound.attested_runtime_membership.v0`.
- Human meaning of the first proof statement:
  - a verified audited `isolate_session_bound` event exists inside one verified `AuditSegmentSeal`
  - that event binds to one public attested runtime identity seam
  - the proof does not reveal the full private session payload needed to make the claim true
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
  - `session_binding_digest`
  - `binding_commitment`
- The prover witness should stay bounded and typed. Recommended minimum witness inputs are:
  - one private normalized remainder of the `IsolateSessionBoundPayload`
  - one Merkle authentication path from the audited record digest to the segment root
  - the minimum linking material required to prove the witness commits to the audited statement under the declared normalization profile

### Why This First Statement
- This statement is preferred for `v0` because it already sits on one of RuneCode's strongest existing typed assurance seams: audited isolate session binding plus attested runtime identity.
- It reuses the stable typed payload family `IsolateSessionBoundPayload` and its existing public digest bindings such as `runtime_image_descriptor_digest`, `attestation_evidence_digest`, and `session_binding_digest`.
- It avoids creating a second policy semantics implementation and avoids depending on broader or less settled proof families before the first narrow proof is evaluated.

### Trusted Statement Compilation
- Trusted Go code should first verify the authoritative audit objects and then compile one small typed proof-input contract for proving and verification.
- The proof circuit must not parse full signed envelopes, arbitrary protocol JSON, or RFC 8785 JCS objects directly.
- The proof-input contract should be versioned by `normalization_profile_id` so proof meaning does not drift silently when trusted code evolves.
- Trusted Go code should compile the proof input from a verified `AuditEventPayload` with `audit_event_type = isolate_session_bound` and `event_payload_schema_id = runecode.protocol.v0.IsolateSessionBoundPayload`.
- The first proof family should require `attestation_evidence_digest` to be present. Events without attested posture are not eligible for this initial statement family.
- If a proof statement later depends on verified-mode RuneContext project context, it must bind the validated project-substrate snapshot digest already produced by the verified project-substrate flow rather than inventing a second project-context identity.
- If a proof statement later depends on attested runtime evidence, it must bind the attestation evidence or attestation verification identity already established by the trusted runtime-evidence path rather than flattening runtime identity into launch-only assumptions.

### Normalization Profiles
- The first proof family should use a two-layer normalization model.
- Logical normalization profile:
  - scheme-agnostic
  - defines eligible source object families, required field presence, public/private field split, enum coding, field ordering, and missing-field rules
  - recommended first logical profile id: `runecode.zk.normalize.audit.isolate_session_bound.attested_runtime.v0`
- Scheme-adapter profile:
  - proving-system-specific
  - defines how normalized slots are packed into field elements and how the proof-friendly commitment is computed for the chosen proving backend
  - recommended first adapter profile id: `runecode.zk.adapter.gnark.groth16.isolate_session_bound_attested_runtime.v0`
- This split is the main mechanism that keeps the proof contract agnostic while allowing RuneCode to switch from `Groth16` to `PLONK` or another proving family later without rewriting the whole broker/audit contract.

### Recommended First Logical Profile Details
- Eligible source object:
  - verified `AuditEventPayload`
  - `audit_event_type = isolate_session_bound`
  - `event_payload_schema_id = runecode.protocol.v0.IsolateSessionBoundPayload`
  - event must already pass signed-envelope verification, event-contract validation, signer-evidence validation, payload-hash validation, and seal inclusion verification
- Recommended public logical fields:
  - `runtime_image_descriptor_digest`
  - `attestation_evidence_digest`
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

### Proof-Binding Sidecars
- `AuditProofBinding`-style sidecars are part of the intended `v0` implementation foundation and should be produced by trusted Go after normal authoritative audit verification.
- Recommended first sidecar family purpose:
  - bind one proof family to one authoritative source record and seal identity
  - preserve one scheme-agnostic normalized statement projection
  - allow later local or remote proof systems to consume proof-ready bindings without reparsing arbitrary historical objects differently
- `v0` should not defer this substrate, because later operator-private backfill and later proving-system agility both depend on preserving proof-ready derived bindings from the start.
- Recommended first proof-binding sidecar fields include at least:
  - `statement_family`
  - `statement_version`
  - `normalization_profile_id`
  - source `audit_record_digest`
  - source `audit_segment_seal_digest`
  - `protocol_bundle_manifest_hash`
  - `binding_commitment`
  - projected public bindings such as `runtime_image_descriptor_digest`, `attestation_evidence_digest`, and `session_binding_digest`
- The proof-binding sidecar is not the proof itself. It is additive derived evidence that stabilizes proof-input meaning across time, proving systems, and later backfill services.

### Proof-Binding Production Rules
- Trusted Go code should emit an `AuditProofBinding` sidecar only after the source audit event, source segment seal, and all required trusted verification preconditions for the selected proof family have succeeded.
- The sidecar should be immutable once persisted.
- The sidecar should be keyed by its own digest and should also retain stable references to the source `audit_record_digest` and `audit_segment_seal_digest`.
- If the selected proof family requires verified-mode project context, the sidecar should carry the validated project-substrate snapshot digest used at proof-binding time.
- If the selected proof family requires attested runtime context, the sidecar should carry the attestation evidence or attestation verification identity required by that proof family.
- Sidecar generation must fail closed on ambiguous source identity, ambiguous normalization-profile selection, or incomplete required source evidence.
- Sidecar generation must not infer missing proof inputs from filenames, directory scans, client-local caches, or other non-canonical ambient context.

## Proof Object And Verification Model

### Scheme-Agnostic Proof Contract
- The proof object family should remain scheme-agnostic even if `v0` only enables one concrete `gnark` plus `Groth16` implementation.
- Recommended required proof-object fields are:
  - `statement_family`
  - `statement_version`
  - `scheme_id`
  - `curve_id`
  - `circuit_id`
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
  - verifier-key digest
  - public-inputs digest
  - verification timestamp
  - verification result
  - stable reason codes
  - cache provenance

### Verification Placement
- Trusted Go verification remains authoritative.
- Broker-owned explicit proof commands or APIs should trigger proof generation and proof verification using the same reviewed local control-plane pattern already used for audit verification surfaces.
- Proof verification should record machine-readable results in trusted persistence and audit the verification outcome.
- `audit_finalize_verify` may later surface proof-reference posture or cached proof-verification posture, but it should not become the only entrypoint for running ZK verification in `v0`.

### Persistence Model
- For the first proof family, authoritative proof persistence should live alongside other audit sidecar evidence owned by `auditd`.
- `AuditProofBinding` sidecars should live in the same audit-owned authoritative sidecar model as the first proof family rather than in a second proof-only authority store.
- Artifact-store copies may exist for export or review, but those copies remain derivatives.
- Audit records should reference proof and verification digests, status, and reason codes rather than embedding large proof bytes in ordinary events.

### Local And Remote Lane Authority Boundaries
- Local lane authority:
  - authoritative local audit evidence and proof-binding sidecars are produced and verified inside trusted RuneCode services
  - local proof verification is authoritative for any `v0` proof family RuneCode supports everywhere
- Remote lane authority:
  - any future operator-private remote proof service consumes exported canonical evidence and proof-binding sidecars
  - remote proofs are additive derived evidence and must not replace the authoritative audit ledger, authoritative runtime evidence, authoritative verified project-substrate bindings, or authoritative local verification results
  - remote services must not invent a second approval, policy, or project-truth model
- Shared binding rule:
  - both local and remote lanes must consume the same `statement_family`, `normalization_profile_id`, source digests, and typed assurance bindings

## Canonical Evidence Preservation And Future Proof Backfill

### Preservation Requirement
- If RuneCode later wants a broader proof portfolio or a stronger external verifiability story, every RuneCode instance must preserve enough canonical source evidence now so later proofs can be generated from history without depending on ambient local process state.
- Preserving only final digests is insufficient where later proof witnesses need more than one already-compressed summary field.
- Evidence preservation must be efficient and resilient, but it must bias toward retaining canonical proof-relevant source evidence strongly enough that future witness reconstruction remains possible.
- The preservation rule applies regardless of whether a deployment ever turns on the future remote proof lane. Evidence capture must be complete enough now that the remote lane can be enabled later and backfill the entire RuneCode-managed project history from that point onward.

### What Every RuneCode Machine Should Preserve
- Every RuneCode instance should preserve or export enough canonical proof-relevant evidence to support both the local high-performance proof core and a later additive remote proof lane.
- Required evidence classes include at least:
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
- This preservation requirement applies equally when RuneCode runs concurrently across more than one machine on the same project. The shared assurance story depends on canonical evidence capture remaining complete enough that later proof generation can explain cross-machine activity using the same typed bindings.
- The preservation/export design should assume that a later operator-private remote proof service may need to ingest historical evidence in bulk and then backfill proofs for all preserved history, not only for newly generated events.

### Export And Backfill Expectations
- The future remote lane should ingest self-contained exported canonical evidence bundles rather than ambient paths into live local stores.
- Export bundles intended for later proof backfill should carry:
  - canonical audit evidence
  - canonical runtime and attestation evidence needed by supported proof families
  - validated project-substrate bindings when required by supported proof families
  - proof-binding sidecars
  - manifest and authenticity material needed to verify the bundle itself
- A future remote proof service should be able to rebuild its proof-work queue entirely from those exported bundles plus configured proof-family support, without asking the originating machine for additional ambient local context.
- Backfilled proofs should be written back as additive derived evidence using the same proof contract and the same proof-binding identities as locally generated proofs.

### Dual-Lane ZK Roadmap
- RuneCode should optimize first for a small stable high-performance local proof core and only then add a broader faster-evolving proof portfolio as an additive lane over the same canonical evidence substrate.
- Local high-performance proof core:
  - available everywhere RuneCode runs
  - small number of narrow stable proof families
  - cheap trusted local verification
  - no remote dependency for correctness
- Broader proof portfolio lane:
  - optional and asynchronous
  - may run on stronger operator-private remote infrastructure
  - consumes exported canonical evidence and proof-binding sidecars
  - may backfill history and extend external verifiability over past project activity
  - must not become the local product's only correctness path or a second source of trust semantics
- Implementation rule for `v0` planning:
  - the local proof core is the only lane this change intends to implement end-to-end
  - the broader remote lane is captured as a required future-compatible foundation and evidence-preservation target, not as implementation work for this change

### Future Remote Proof Service Posture
- The recommended future posture is to start with an operator-private remote proof service rather than an externally consumable public-assurance publication layer.
- That future service should be able to:
  - ingest exported canonical evidence bundles
  - verify imported evidence against the same authoritative typed bindings
  - consume proof-binding sidecars or derive equivalent proof-ready inputs from preserved evidence
  - generate additional proofs over historical records
  - publish those proofs back as additive evidence without changing the local trust model
- Later externally consumable publication should reuse the same proof bindings rather than creating a second public-assurance-only binding model.
- The future remote lane may use a broader or faster-evolving proof portfolio than the local lane, but it must still consume canonical evidence and proof-binding sidecars produced under the same reviewed semantics.
- The future remote lane may use a different proving backend than the local lane, but it must still honor the same logical normalization profiles and statement-family bindings.

### What This Does Not Change
- The existence of a future remote proof lane does not justify a separate local architecture for constrained devices.
- Every RuneCode node should still capture the same canonical evidence and run the same reviewed local high-performance proof core.
- The remote lane remains additive and optional. If it is unavailable, RuneCode's core security and assurance architecture must still stand on its own.

## Performance And Scaling Posture

### One Architecture Everywhere
- RuneCode must keep one reviewed proof-verification architecture across constrained and scaled environments.
- Different deployments may vary in cache population, local storage sizing, or background scheduling, but they must not vary in trust roots, verification semantics, or fallback behavior.

### Required Performance Shape
- Proof generation is explicit and off the interactive hot path.
- Proof verification must be cheap enough for routine local use and cached by immutable identity.
- Read models and watch surfaces must consume persisted proof-verification results and must not rerun expensive proof verification during refresh.
- External network dependence, remote prover dependence, or device-class-specific alternate proof architectures are out of scope for `v0`.

### Recommended Verification-Result Cache Identity
- Proof verification results should be cached by immutable identity including at least:
  - proof digest
  - `statement_family`
  - `statement_version`
  - `scheme_id`
  - `curve_id`
  - `circuit_id`
  - `verifier_key_digest`
  - `public_inputs_digest`
  - `normalization_profile_id`
  - verifier implementation version

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
