# ZK Proof v0 (One Narrow Proof + Verify)

User-visible outcome: RuneCode can generate and verify at least one narrowly scoped zero-knowledge proof that attests to deterministic integrity claims without revealing sensitive contents.

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-08-1039-zk-proof-v0/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

## Task 2: Pick the First Proof Statement

Select one MVP proof type (keep narrow and deterministic), e.g.:
- Audit integrity: prove knowledge of a private audit event sequence whose hash chain yields a published audit root.
- Policy decision integrity: prove that a policy program evaluated `{manifest_hash, request_hash}` to decision `D`.
- Add an explicit feasibility gate:
  - the statement must have bounded inputs and fully deterministic verification
  - if proof generation or verification performance is not acceptable, ship this as post-MVP (interfaces/fixtures only)

## Task 3: Choose Proving System + Libraries

- Choose a pragmatic proving approach for MVP.
- Ensure verification is fast and deterministic.
- Define MVP performance targets before implementation:
  - verification must be fast enough for routine use (target: sub-second; ideally sub-100ms)
  - proof artifacts must have bounded size
  - proof generation must not dominate the run (or must be explicitly opt-in)

## Task 4: Proof Artifact Format + Storage

- Define a proof artifact type with:
  - statement id/version
  - public inputs
  - proof bytes
  - verifier result
- Store proofs in the artifact store and record verification in the audit chain.

## Task 5: CLI Integration

- Add commands to:
  - generate proof for a run or audit root
  - verify a proof artifact

## Acceptance Criteria

- At least one proof type can be generated and verified end-to-end.
- Proof verification is deterministic, recorded in the audit log, and failure is non-destructive (it flags the run).
- If performance targets cannot be met with a concrete proving system, this capability is deferred post-MVP rather than weakening core MVP deliverables.
