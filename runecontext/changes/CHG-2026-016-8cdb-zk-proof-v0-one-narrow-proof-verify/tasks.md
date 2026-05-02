# Tasks

## Pick the First Proof Statement

- [ ] Select one MVP proof type and freeze it as one audit-bound statement family rather than a broad proof lane.
- [ ] Recommended `v0` statement family: prove knowledge of one private normalized witness bound to one audited commitment whose audited record digest is included in one verified `AuditSegmentSeal`.
- [ ] Keep the first proof audit-bound rather than proving policy-program execution directly.
- [ ] Keep public inputs bounded and typed, including at least:
  - `statement_family`
  - `statement_version`
  - `audit_segment_seal_digest`
  - `merkle_root`
  - `audit_record_digest`
  - `protocol_bundle_manifest_hash`
  - `public_witness_commitment`
- [ ] Keep witness inputs bounded and typed, including one normalized private witness and one Merkle authentication path.
- [ ] Add an explicit feasibility gate:
  - the statement must have bounded inputs and fully deterministic verification
  - if proof generation or verification performance is not acceptable, defer release rather than weakening the proof contract
- [ ] Add a trusted statement-compilation step in Go that derives one small typed proof-input contract from already-verified trusted objects.
- [ ] Keep the proof circuit from directly parsing full signed envelopes, arbitrary protocol JSON, or ambient repository state.
- [ ] When the chosen statement depends on project context, bind it to validated project-substrate snapshot identity rather than ambient repo state.
- [ ] When the chosen statement depends on runtime execution identity, bind it to the attested runtime identity seam from `CHG-2026-030-98b8-isolate-attestation-v0`, using signed runtime-image descriptor identity, persisted reviewed launch evidence, and attestation evidence or verification where relevant rather than ambient platform-specific runtime state.
- [ ] When the chosen statement depends on external audit anchoring, bind it to canonical `AuditSegmentSeal` identity, authoritative anchor receipt identity, canonical target descriptor identity where relevant, and typed sidecar proof references rather than flattened summaries, raw transport details, or exported-copy artifacts.
- [ ] If proof families later expand deeper into RuneCode, reuse verified-mode RuneContext audit, project-substrate, attestation, and related typed assurance identities rather than introducing summary-only or ambient trust surfaces.

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
- [ ] Keep the first proof family's authoritative persistence as audit-owned sidecar evidence.
- [ ] Keep artifact-store copies optional review/export products rather than the primary trust source.
- [ ] If a proof-export artifact data class is introduced, use a proof-specific class rather than overloading existing audit-report classes.
- [ ] Record proof-generation and proof-verification outcomes in the audit chain.

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
- [ ] The proof contract remains scheme-agnostic even though `v0` uses one concrete proving system.
- [ ] Authoritative proof persistence for the first proof follows the audit-sidecar truth model, with artifact-store copies remaining optional derivatives.
- [ ] Verified-mode RuneContext bindings are reused whenever project-context-sensitive, attestation-sensitive, or later assurance-sensitive proof families expand deeper into RuneCode.
- [ ] The same proof-verification architecture and trust semantics run on constrained and scaled deployments, with performance differences handled by caching and scheduling rather than by separate architectures.
- [ ] If performance targets cannot be met with a concrete proving system, this capability is deferred to a later release rather than weakening core deliverables.
