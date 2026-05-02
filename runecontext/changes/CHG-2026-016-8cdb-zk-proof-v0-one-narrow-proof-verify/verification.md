# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm the migrated change preserves the legacy task breakdown and acceptance criteria in `tasks.md`.
- Confirm canonical references remain on RuneContext project, spec, and change paths, with no active workflow depending on legacy planning paths.
- Confirm the migrated text assumes RuneContext is canonical, RuneCode owns the user-facing UX, and verified-mode project state remains the expected operating posture.
- Confirm the change still matches its `v0.1.0-alpha.10` roadmap bucket and title after migration.
- Confirm the first proof direction is explicitly audit-bound and does not attempt to prove broad policy-program execution in `v0`.
- Confirm the exact first proof family is documented as `audit.isolate_session_bound.attested_runtime_membership.v0` or an equivalent typed name with the same scoped meaning.
- Confirm the first proof is tied to attested `isolate_session_bound` audit evidence rather than an ambient runtime or repository assumption.
- Confirm the proof contract is explicitly scheme-agnostic even though the recommended `v0` implementation path is `gnark` plus `Groth16`.
- Confirm the change now defines both a logical normalization profile and a proving-system adapter profile for the first proof family.
- Confirm `AuditProofBinding`-style sidecars or equivalent additive proof-ready derived evidence are explicitly part of the intended `v0` foundation rather than a later convenience.
- Confirm authoritative persistence for the first proof follows the audit-sidecar truth model and does not promote artifact-store copies into the primary audit authority.
- Confirm proof verification remains supplemental integrity evidence and does not create a second authorization semantics path outside the shared trusted policy engine.
- Confirm project-context-sensitive proof families are documented to reuse the validated project-substrate snapshot digest from verified-mode RuneContext rather than ambient repository assumptions.
- Confirm any project-context-sensitive proof statement binds validated project-substrate snapshot identity rather than ambient repository assumptions.
- Confirm any runtime-sensitive proof statement binds the attested runtime identity seam from `CHG-2026-030-98b8-isolate-attestation-v0` rather than only pre-attestation launch assumptions or ambient platform-specific state.
- Confirm any audit-anchoring-sensitive proof statement binds canonical `AuditSegmentSeal` identity, authoritative anchor receipt identity, canonical target descriptor identity where relevant, and typed sidecar proof references rather than flattened summaries or exported-copy artifacts.
- Confirm the change explicitly requires preserving enough canonical proof-relevant source evidence and proof-ready binding information for later operator-private remote proof backfill.
- Confirm the change explicitly distinguishes the always-available local proof core from a later additive operator-private remote proof lane rather than turning remote proving into a required correctness dependency.
- Confirm the change documents the local-lane and future-remote-lane authority boundaries clearly enough that a follow-on implementer does not need new product clarification to understand what is authoritative locally, what is additive remotely, and what bindings must stay shared.
- Confirm the change documents export-bundle expectations for future proof backfill strongly enough that a remote lane can ingest canonical evidence without ambient source-machine context.
- Confirm the change explicitly requires a post-implementation evaluation and user check-in before expanding further proof families or follow-on remote-lane preparation.
- Confirm the documented performance gates preserve one architecture across constrained and scaled environments and use caching/scheduling rather than separate trust models for low-power systems.

## Close Gate
Use the repository's standard verification flow before closing this change.
