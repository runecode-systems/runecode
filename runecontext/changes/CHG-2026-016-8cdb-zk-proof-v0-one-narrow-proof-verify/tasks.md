# Tasks

## Pick the First Proof Statement

- [ ] Select one MVP proof type (keep narrow and deterministic), e.g.:
  - Audit integrity: prove knowledge of a private audit event sequence whose hash chain yields a published audit root.
  - Policy decision integrity: prove that a policy program evaluated `{manifest_hash, request_hash}` to decision `D`.
- [ ] Add an explicit feasibility gate:
  - the statement must have bounded inputs and fully deterministic verification
  - if proof generation or verification performance is not acceptable, defer release rather than weakening the proof contract
- [ ] When the chosen statement depends on project context, bind it to validated project-substrate snapshot identity rather than ambient repo state.
- [ ] When the chosen statement depends on runtime execution identity, bind it to the attested runtime identity seam from `CHG-2026-030-98b8-isolate-attestation-v0`, using signed runtime-image descriptor identity, persisted reviewed launch evidence, and attestation evidence or verification where relevant rather than ambient platform-specific runtime state.
- [ ] When the chosen statement depends on external audit anchoring, bind it to canonical `AuditSegmentSeal` identity, authoritative anchor receipt identity, canonical target descriptor identity where relevant, and typed sidecar proof references rather than flattened summaries, raw transport details, or exported-copy artifacts.

Parallelization: can be done in parallel with audit/artifact specs; keep the chosen statement aligned with the canonical audit root/verification artifacts.

## Choose Proving System + Libraries

- [ ] Choose a pragmatic proving approach for MVP.
- [ ] Ensure verification is fast and deterministic.
- [ ] Define MVP performance targets before implementation:
  - verification must be fast enough for routine use (target: sub-second; ideally sub-100ms)
  - proof artifacts must have bounded size
  - proof generation must not dominate the run (or must be explicitly opt-in)

Parallelization: can be evaluated in parallel with other later hardening work; treat library selection as security-sensitive.

## Proof Artifact Format + Storage

- [ ] Define a proof artifact type with:
  - statement id/version
  - public inputs
  - proof bytes
  - verifier result
- [ ] Store proofs in the artifact store and record verification in the audit chain.

Parallelization: can be implemented in parallel with artifact store and audit log work; it depends on stable proof artifact schemas.

## CLI Integration

- [ ] Add commands to:
  - generate proof for a run or audit root
  - verify a proof artifact

Parallelization: can be implemented in parallel with TUI/CLI work.

## Acceptance Criteria

- [ ] At least one proof type can be generated and verified end-to-end.
- [ ] Proof verification is deterministic, recorded in the audit log, and failure is non-destructive (it flags the run).
- [ ] If performance targets cannot be met with a concrete proving system, this capability is deferred to a later release rather than weakening core deliverables.
