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
- Proof verification is supplemental assurance evidence in `v0`; it must not create a second authorization path or redefine allow/deny/approval semantics outside the shared policy engine.
- Proof verification must not become an ambient hot-path cost in read models, watch paths, or ordinary `audit_finalize_verify` execution. Read models should consume persisted verification results, and verification results should be cached by immutable proof identity.
- Performance posture follows the same RuneCode rule used for attestation and other trusted verification work: one architecture everywhere, no smaller-device exception architecture, and no optimization that weakens trust semantics.
- If a proof statement depends on project context, it should bind the validated project-substrate snapshot identity rather than ambient repository assumptions.
- If a proof statement depends on runtime execution identity, it should bind the attested runtime identity seam established by `CHG-2026-030-98b8-isolate-attestation-v0`, using signed runtime-image descriptor identity, persisted reviewed launch evidence, and attestation evidence or verification where relevant rather than ambient platform-specific runtime assumptions.
- If a proof statement depends on external audit anchoring, it should bind the canonical `AuditSegmentSeal` subject plus authoritative anchor receipt identity, canonical target descriptor identity, and typed sidecar proof references rather than raw transport URLs, flattened target-local summaries, or exported-copy artifacts.

## First Proof Foundation

### Recommended `v0` Statement Family
- The recommended first proof family is one audited-commitment statement rather than a general proof of event-sequence correctness or policy-program execution.
- Public inputs should stay bounded and typed. Recommended minimum public inputs are:
  - `statement_family`
  - `statement_version`
  - `audit_segment_seal_digest`
  - `merkle_root`
  - `audit_record_digest`
  - `protocol_bundle_manifest_hash`
  - `public_witness_commitment`
- The prover witness should stay bounded and typed. Recommended minimum witness inputs are:
  - one private normalized witness payload
  - one Merkle authentication path from the audited record digest to the segment root
  - the minimum linking material required to prove the witness commits to the audited statement under the declared normalization profile

### Trusted Statement Compilation
- Trusted Go code should first verify the authoritative audit objects and then compile one small typed proof-input contract for proving and verification.
- The proof circuit must not parse full signed envelopes, arbitrary protocol JSON, or RFC 8785 JCS objects directly.
- The proof-input contract should be versioned by `normalization_profile_id` so proof meaning does not drift silently when trusted code evolves.
- If a proof statement later depends on verified-mode RuneContext project context, it must bind the validated project-substrate snapshot digest already produced by the verified project-substrate flow rather than inventing a second project-context identity.
- If a proof statement later depends on attested runtime evidence, it must bind the attestation evidence or attestation verification identity already established by the trusted runtime-evidence path rather than flattening runtime identity into launch-only assumptions.

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
- Artifact-store copies may exist for export or review, but those copies remain derivatives.
- Audit records should reference proof and verification digests, status, and reason codes rather than embedding large proof bytes in ordinary events.

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
