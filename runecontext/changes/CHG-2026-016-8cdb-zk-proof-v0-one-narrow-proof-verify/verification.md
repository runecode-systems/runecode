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
- Confirm the change explicitly records that production proof eligibility depends on `CHG-2026-030-98b8-isolate-attestation-v0` producing attested events.
- Confirm the proof contract is explicitly scheme-agnostic even though the recommended `v0` implementation path is `gnark` plus `Groth16`.
- Confirm the change now defines both a logical normalization profile and a proving-system adapter profile for the first proof family.
- Confirm the logical profile's public or private split is described as a proof-disclosure rule rather than as a rewrite of the source schema's `x-data-class` semantics.
- Confirm `binding_commitment` is explicitly defined as a proof-time derived ZK-friendly commitment and not as a source audit field.
- Confirm the change now explicitly requires every statement-critical public field to be cryptographically bound either directly in-circuit or through a verifier-recomputed canonical `public_inputs_digest`.
- Confirm `AuditProofBinding`-style sidecars or equivalent additive proof-ready derived evidence are explicitly part of the intended `v0` foundation rather than a later convenience.
- Confirm authoritative persistence for the first proof follows the audit-sidecar truth model and does not promote artifact-store copies into the primary audit authority.
- Confirm proof verification remains supplemental integrity evidence and does not create a second authorization semantics path outside the shared trusted policy engine.
- Confirm the proof reproduces RuneCode's authoritative Merkle construction exactly, including the leaf or node domain separators and odd-leaf duplication rule.
- Confirm the proof-binding sidecar captures the Merkle authentication path needed for the first proof family.
- Confirm the change defines circuit freeze, `constraint_system_digest`, `setup_provenance_digest`, verifier-key pinning, and fail-closed setup-identity mismatch handling clearly enough for implementation.
- Confirm the change explicitly requires trusted verifier posture to be derived only from reviewed local verifier assets rather than from proof-artifact declarations.
- Confirm the change explicitly records the unfinished `v0` gap that the trusted local proof backend remains hard-disabled until reviewed setup assets are delivered through trusted assets.
- Confirm the change now distinguishes the authoritative/default trusted backend from a separate evaluation-only Groth16 path for benchmark and feasibility work.
- Confirm the evaluation-only path is documented as non-authoritative, benchmark-only, and not reachable through the normal authoritative broker proof-generate or proof-verify surfaces.
- Confirm the change explicitly says serious benchmarking should happen only after the circuit shape binds every statement-critical public field, likely through a circuit-public `public_inputs_digest`.
- Confirm the change requires pinned non-authoritative benchmark setup assets for the evaluation-only path rather than runtime trusted setup generation.
- Confirm the change requires proof-related protocol schemas and registries to be added to the authoritative protocol manifest discipline.
- Confirm the change requires proof-library isolation behind trusted local interfaces and compatibility checks for `gnark` introduction.
- Confirm project-context-sensitive proof families are documented to reuse the validated project-substrate snapshot digest from verified-mode RuneContext rather than ambient repository assumptions.
- Confirm any runtime-sensitive proof statement binds the attested runtime identity seam from `CHG-2026-030-98b8-isolate-attestation-v0` rather than only pre-attestation launch assumptions or ambient platform-specific state.
- Confirm any audit-anchoring-sensitive proof statement binds canonical `AuditSegmentSeal` identity, authoritative anchor receipt identity, canonical target descriptor identity where relevant, and typed sidecar proof references rather than flattened summaries or exported-copy artifacts.
- Confirm the change explicitly requires preserving enough canonical proof-relevant source evidence and proof-ready binding information locally for later backfill prerequisites, even when no remote or public proof lane is enabled.
- Confirm the change explicitly requires immutable preserved runtime, attestation, and attestation-verification evidence rather than treating live `run_id` store lookup as the only historical evidence model.
- Confirm the change explicitly requires verification to validate the referenced `AuditProofBinding` and authoritative source evidence before persisting a `verified` proof result.
- Confirm the change explicitly requires the verification cache to avoid duplicate cryptographic verification rather than only duplicate persistence.
- Confirm the change explicitly requires proof-generation and proof-verification audit recording failures to fail closed or produce an explicit degraded record.
- Confirm the change explicitly requires trusted Go validation to mirror protocol-schema bounds for proof artifacts and public-input envelopes.
- Confirm the change explicitly requires benchmark coverage for crypto-core timings and a broker-adjacent end-to-end harness, using `go test -bench` by default and keeping any optional benchmark command non-authoritative.
- Confirm the change explicitly separates evaluation-only performance measurement from authoritative broker enablement, and that passing benchmarks alone is not sufficient to re-enable the authoritative backend.
- Confirm the detailed additive remote or public proof-lane design has been moved to `CHG-2026-055-b7e4-additive-remote-public-proof-lane` rather than left ambiguously inside this local `v0` implementation change.
- Confirm the documented performance gates preserve one architecture across constrained and scaled environments and use caching, queueing, and scheduling rather than separate trust models for low-power systems.
- Confirm the documented performance gates are described as required CI or scheduled enforcement prerequisites for re-enabling the backend rather than as optional benchmark guidance.
- Confirm the change now documents a phased implementation order: authoritative correctness and circuit shaping first, evaluation-only backend and benchmarks second, authoritative reviewed setup assets and backend enablement later, and CHG-055 alternative review only if measured results miss the gates badly.
- Confirm the change explicitly requires a post-implementation evaluation and user check-in before considering broader proof-lane expansion or the additive dual-commitment alternative.

## Close Gate
Use the repository's standard verification flow before closing this change.
