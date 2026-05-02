# Tasks

## Pick the First Proof Statement

- [ ] Select one MVP proof type and freeze it as one audit-bound statement family rather than a broad proof lane.
- [ ] Recommended exact `v0` statement family: `audit.isolate_session_bound.attested_runtime_membership.v0`.
- [ ] Recommended `v0` statement meaning: prove that one verified audited `isolate_session_bound` event exists inside one verified `AuditSegmentSeal`, and that the event binds to one public attested runtime identity seam without revealing the full normalized private session payload.
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
  - `applied_hardening_posture_digest`
  - `session_binding_digest`
  - `binding_commitment`
- [ ] Cryptographically bind every statement-critical public field either directly in-circuit or through a canonical `public_inputs_digest` that is itself circuit-public and recomputed from the full typed public-input object during trusted verification.
- [ ] Keep witness inputs bounded and typed, including the normalized private remainder of the `IsolateSessionBoundPayload` plus one Merkle authentication path.
- [ ] Add an explicit feasibility gate:
  - the statement must have bounded inputs and fully deterministic verification
  - if proof generation or verification performance is not acceptable, defer release rather than weakening the proof contract
- [ ] Add a trusted statement-compilation step in Go that derives one small typed proof-input contract from already-verified trusted objects.
- [ ] Restrict the first proof family to verified `AuditEventPayload` records where:
  - `audit_event_type = isolate_session_bound`
  - `event_payload_schema_id = runecode.protocol.v0.IsolateSessionBoundPayload`
  - `attestation_evidence_digest` is present
- [ ] Record explicitly that end-to-end production proofs for real events depend on `CHG-2026-030-98b8-isolate-attestation-v0` producing eligible attested events in the authoritative audit ledger.
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
  - `applied_hardening_posture_digest`
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
- [ ] State explicitly that the logical profile's public and private split is a proof-disclosure rule, not a rewrite of the authoritative source schema's `x-data-class` semantics.
- [ ] Normalize variable-length identifiers and enum-like values into stable proof-friendly representations rather than open-ended raw strings where practical.
- [ ] Define `binding_commitment` as a proof-time derived ZK-friendly Poseidon-family commitment over the normalized private field set, not as an existing source audit field.
- [ ] Require trusted Go statement compilation to verify the off-circuit relationship between the normalized private field set and the source `session_binding_digest` before the proof-binding sidecar is emitted.
- [ ] Require that off-circuit session-binding verification to validate every normalized private field selected by the profile against immutable runtime or session evidence, not just compare the source digest string.

Parallelization: can be done in parallel with audit or artifact specs; keep the chosen statement aligned with the canonical audit root and verification artifacts.

## Merkle Membership Foundation

- [ ] Implement a trusted helper that derives a Merkle authentication path for one audited record inside one authoritative ordered audit segment.
- [ ] Implement a trusted helper that verifies the derived authentication path against the authoritative `AuditSegmentSeal` root outside the circuit.
- [ ] Keep the path format versioned and stable because it becomes part of the proof witness contract and proof-binding sidecar contract.
- [ ] Store the Merkle authentication path in the `AuditProofBinding` sidecar rather than requiring re-reading the full segment during proof generation.
- [ ] Reproduce the exact RuneCode Merkle construction in the circuit and fixtures, including:
  - leaf domain separation `runecode.audit.merkle.leaf.v1:`
  - node domain separation `runecode.audit.merkle.node.v1:`
  - ordered left and right sibling semantics
  - odd-leaf duplication when no right sibling exists
- [ ] Set an explicit maximum Merkle depth bound of `12` for the first proof family and fail closed if a witness exceeds that bound.
- [ ] Add deterministic fixtures that prove the circuit path logic matches the authoritative Go Merkle implementation before any verifier key is accepted.

## Choose Proving System + Libraries

- [ ] Choose a pragmatic proving approach for MVP.
- [ ] Keep the proof contract scheme-agnostic even if `v0` uses one concrete implementation.
- [ ] For `v0`, prefer `gnark` in trusted Go with a fixed-circuit `Groth16` verifier if the performance targets are met.
- [ ] Treat `Groth16` as a `v0` performance choice, not a forever-global proving-system commitment.
- [ ] Freeze one reviewed first circuit before setup material is generated.
- [ ] Close the circuit-shaping gaps before serious benchmarking, especially cryptographically binding every statement-critical public field, likely through a circuit-public `public_inputs_digest` recomputed by trusted verification.
- [ ] Track the frozen circuit with both:
  - `circuit_id` for the reviewed circuit family and version
  - `constraint_system_digest` for the compiled constraint-system artifact
- [ ] Prefer reuse of a well-audited Phase 1 Powers-of-Tau lineage for the selected curve rather than inventing a fresh RuneCode-specific Phase 1 ceremony.
- [ ] Run and document an explicit Phase 2 ceremony for the frozen first circuit, including a stable transcript digest.
- [ ] Define `setup_provenance_digest` as a canonical SHA-256 digest over the setup-lineage object, including at least Phase 1 lineage identity and digest, Phase 2 transcript digest, frozen circuit source digest, `constraint_system_digest`, and selected `gnark` module version identity.
- [ ] Deliver verifier-key material only through reviewed trusted assets.
- [ ] Prohibit runtime setup on user machines and prohibit ambient key download.
- [ ] Keep `NewTrustedLocalGroth16BackendV0()` disabled for authoritative broker or API use until reviewed trusted setup assets and the remaining authoritative correctness gaps are closed.
- [ ] Add a separate evaluation-only Groth16 constructor and setup loader that reuse the same frozen circuit code but derive posture from explicitly non-authoritative benchmark assets.
- [ ] Load evaluation-only setup material from checked-in or embedded benchmark fixtures with pinned digests for the constraint system, proving key, verifying key, and setup metadata so measurements are reproducible.
- [ ] Document the currently unfinished gap explicitly: the authoritative proof backend stays hard-disabled until reviewed trusted setup assets are delivered, because runtime deterministic setup generation is prohibited.
- [ ] Fail closed on any `verifier_key_digest`, `constraint_system_digest`, or `setup_provenance_digest` mismatch.
- [ ] Derive trusted verifier posture only from reviewed local verifier assets and compare artifact-declared setup and scheme identity against that local posture before proof verification.
- [ ] Split prover and verifier construction so ordinary verification never loads or constructs proving-key material and never reruns setup.
- [ ] Pin dependency versions, wrap external library usage behind local trusted interfaces, and treat version drift as security-sensitive.
- [ ] Add an internal package boundary such as `internal/zkproof/` so no `gnark` types escape the trusted local proof implementation surface.
- [ ] Verify Go toolchain compatibility, target-platform build compatibility, and binary-size impact before committing to the library introduction.
- [ ] Add CI coverage that derives and checks the expected `constraint_system_digest` for the frozen circuit so incompatible library drift is caught before release.
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
- [ ] Add benchmark fixtures for the frozen measurement circuit, including precomputed setup material and pinned metadata digests for the evaluation-only path.
- [ ] Add `go test -bench` coverage for proof generation wall time, warm verify, cold verify including key load and public-input normalization, invalid-proof rejection, cache-hit lookup, proof size, public-input envelope size, and peak memory.
- [ ] Add a broker-adjacent end-to-end benchmark harness that exercises compile, proof binding, artifact construction, verification, and temporary persistence without using the authoritative `zk-proof-generate` or `zk-proof-verify` commands.
- [ ] Fix cache timing before trusting cache-hit benchmarks so lookup happens before cryptographic verification rather than only before duplicate persistence.
- [ ] Enforce the documented proof-generation and proof-verification performance gates through required CI or scheduled jobs before re-enabling the backend.
- [ ] Default local proof-generation concurrency to one trusted worker until measured evidence demonstrates a safe higher bound on the target deployment class.
- [ ] Keep proof verification cached by immutable identity and out of watch or read-model refresh hot paths.

Parallelization: can be evaluated in parallel with other later hardening work; treat library selection as security-sensitive.

## Proof Artifact Format + Storage

- [ ] Define a scheme-agnostic proof object type with at least:
  - statement family and version
  - scheme id
  - curve id
  - circuit id
  - `constraint_system_digest`
  - verifier-key digest
  - setup-provenance digest
  - normalization-profile id
  - public inputs
  - public-inputs digest
  - proof bytes
  - source refs
- [ ] Define a separate proof-verification record type with verifier implementation identity, proof identity, verification outcome, and stable reason codes.
- [ ] Require proof verification to check `setup_provenance_digest` against the trusted verifier posture and reject with a stable setup-identity mismatch code if it differs.
- [ ] Require proof verification to resolve the referenced `AuditProofBinding`, validate it by digest, and compare its canonical fields and projected bindings against the proof artifact's public inputs before persisting a verification result.
- [ ] Require proof verification to validate the authoritative audit, runtime, attestation, and project-context source evidence referenced by the binding sidecar rather than verifying proof bytes in isolation.
- [ ] Define an `AuditProofBinding`-style sidecar family as part of the intended `v0` foundation, as additive canonical derived evidence and not as the proof itself.
- [ ] Recommended first proof-binding sidecar fields:
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
- [ ] Emit the proof-binding sidecar only after the source audit event, source segment seal, and all required proof-family preconditions verify successfully.
- [ ] Keep the proof-binding sidecar immutable once persisted.
- [ ] Key the proof-binding sidecar by its own digest while retaining stable references to the source `audit_record_digest` and `audit_segment_seal_digest`.
- [ ] If a proof family requires verified project context or attested runtime context, include the required typed assurance identities directly in the proof-binding sidecar rather than reconstructing them later from ambient context.
- [ ] Keep the first proof family's authoritative persistence as audit-owned sidecar evidence.
- [ ] Keep artifact-store copies optional review or export products rather than the primary trust source.
- [ ] If a proof-export artifact data class is introduced, use a proof-specific class rather than overloading existing audit-report classes.
- [ ] Record proof-generation and proof-verification outcomes in the audit chain.
- [ ] Fail the operation or persist an explicit degraded result if the proof-generation or proof-verification audit append fails; do not silently return success.
- [ ] Make proof generation idempotent at the proof-binding sidecar layer for the same source record, statement family, and adapter identity.
- [ ] Perform verification-cache lookup before expensive cryptographic verification using immutable artifact identity plus trusted verifier posture.
- [ ] Mirror protocol-schema bounds in trusted Go validation for proof bytes, base64 encoding, public-input envelope size, required `v0` public-input fields, and closed registry values.
- [ ] Keep authoritative broker proof-generate and proof-verify surfaces disabled until binding-sidecar verification, authoritative source-evidence verification, fail-closed audit recording, trusted validation tightening, reviewed trusted setup assets, and performance gates are all complete.

## Protocol Schema And Registry Discipline

- [ ] Define canonical protocol schemas for:
  - `AuditProofBinding`
  - the scheme-agnostic proof object family
  - the proof-verification record family
- [ ] Register those schemas in `protocol/schemas/manifest.json`.
- [ ] Add schema files under `protocol/schemas/objects/`.
- [ ] Add registries for at least:
  - `statement_family`
  - `normalization_profile_id`
  - `scheme_adapter_id`
  - `circuit_id`
- [ ] Keep schema, registry, and bundle-manifest identity aligned with the authoritative `protocol_bundle_manifest_hash` model.

## Local Evidence Preservation Foundation

- [ ] Define the minimum canonical source-evidence set that every RuneCode machine must preserve locally or be able to export later so proof backfill remains possible.
- [ ] Preserve enough canonical source evidence and proof-ready binding information that later proof work can reconstruct witnesses from archived authoritative evidence rather than from ambient local process state.
- [ ] Treat this preservation requirement as mandatory even before any remote or public proof lane exists, so future proof backfill prerequisites are never lost on systems that have not enabled follow-on proof features.
- [ ] Preserve or retain exportable access to at least:
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
- [ ] Preserve immutable runtime evidence, attestation evidence, and attestation verification records in a digest-addressed or equivalently immutable form rather than relying only on live store lookup by `run_id`.
- [ ] Capture the requirement that concurrent RuneCode execution across more than one machine on the same project must still preserve enough shared canonical evidence for later cross-machine historical proof work.
- [ ] Keep the detailed remote-ingest, export-bundle, and public-assurance design work in `CHG-2026-055-b7e4-additive-remote-public-proof-lane` rather than expanding this `v0` local implementation scope.

## Evaluation And Check-In Gate

- [ ] Implement and verify the first narrow proof family end-to-end before broadening the ZK roadmap work.
- [ ] Evaluate whether the first proof materially improves RuneCode's assurance story relative to its implementation and runtime cost.
- [ ] Evaluate performance first through the separate evaluation-only Groth16 path rather than by turning on the authoritative broker generate or verify surfaces.
- [ ] Evaluate whether the first proof meets the documented performance gates on required Linux CI and scheduled low-power ARM64.
- [ ] Evaluate whether `gnark` plus `Groth16` remains the right `v0` proving choice after real end-to-end measurement.
- [ ] Keep benchmark entrypoints non-authoritative: use `go test -bench` by default, and if a dedicated benchmark command is added, keep it separate from authoritative proof generation and verification semantics.
- [ ] If the direct authoritative-Merkle-membership design misses the gates badly, stop and perform a separate explicit architecture review before considering the additive dual-commitment proof-bridge option captured in `CHG-2026-055-b7e4-additive-remote-public-proof-lane`.
- [ ] Check in with the user after the first proof evaluation before expanding into broader follow-on proof-lane work.

## CLI Integration

- [ ] Add commands to:
  - generate proof for a supported audited statement
  - verify a proof artifact
- [ ] Keep broker or API commands explicit and trusted rather than ambient background work.
- [ ] Keep proof verification out of ordinary TUI, watch, or read-model refresh paths.
- [ ] Do not route evaluation-only benchmark work through the authoritative proof-generate or proof-verify commands.
- [ ] Optional: add a dedicated non-authoritative benchmark command such as `zk-proof-benchmark` if RuneCode wants CLI or local-RPC overhead included in measurement.

Parallelization: can be implemented in parallel with TUI or CLI work.

## Acceptance Criteria

- [ ] At least one proof type can be generated and verified end-to-end.
- [ ] Proof verification is deterministic, recorded in the audit log, and failure is non-destructive.
- [ ] The first proof statement is audit-bound, uses bounded typed inputs, and does not create a second policy or project-truth surface.
- [ ] The exact `v0` proof family is documented and bound to one attested `isolate_session_bound` audited event shape plus one verified `AuditSegmentSeal` inclusion path.
- [ ] The first proof family has one logical normalization profile and one initial scheme-adapter profile documented explicitly.
- [ ] The proof contract remains scheme-agnostic even though `v0` uses one concrete proving system.
- [ ] Authoritative proof persistence for the first proof follows the audit-sidecar truth model, with artifact-store copies remaining optional derivatives.
- [ ] `AuditProofBinding`-style sidecars are part of the intended `v0` implementation foundation, not merely a later recommendation.
- [ ] The proof-binding sidecar captures the exact Merkle authentication path needed for the first proof family so proof generation does not depend on re-reading the full segment opportunistically.
- [ ] The circuit and fixtures reproduce RuneCode's authoritative Merkle construction exactly, including the domain separators and odd-leaf duplication rule.
- [ ] The proof design defines `binding_commitment` explicitly as a proof-time derived ZK-friendly commitment and does not require adding it to the source audit payload schema.
- [ ] Every statement-critical public field is cryptographically bound either directly in-circuit or through a verified canonical `public_inputs_digest`.
- [ ] Trusted verifier posture comes only from reviewed local assets, and setup-identity mismatch checks do not compare artifact claims to themselves.
- [ ] The change distinguishes the authoritative trusted backend from a separate evaluation-only Groth16 path rather than treating all `Groth16` work as either fully enabled or fully disabled.
- [ ] The authoritative/default backend and authoritative broker proof-generate or proof-verify surfaces remain disabled until reviewed trusted setup assets and the remaining correctness and trust prerequisites are complete.
- [ ] The evaluation-only path uses pinned non-authoritative benchmark setup assets, is reachable only through benchmark entrypoints, and does not persist authoritative verified proof results.
- [ ] Proof verification validates the referenced `AuditProofBinding` and authoritative source evidence before persisting a `verified` result.
- [ ] Verification caching avoids duplicate cryptographic work instead of only avoiding duplicate persistence.
- [ ] Verification success is not returned when the authoritative proof audit event could not be recorded.
- [ ] Serious benchmarking happens only after the circuit shape reflects full statement-critical public-input binding, so the performance data represents the intended `v0` statement.
- [ ] The proof-verification architecture and trust semantics run on constrained and scaled deployments, with performance differences handled by caching, queueing, and scheduling rather than by separate architectures.
- [ ] The change explicitly requires preserving enough canonical proof-relevant source evidence locally that future proof backfill prerequisites are not lost even when no remote or public proof lane is enabled on that machine.
- [ ] The change explicitly requires immutable preserved runtime and attestation evidence rather than ambient live-store lookup as the only historical evidence model.
- [ ] The change explicitly requires an evaluation-and-user-check-in gate after the first proof is implemented and measured.
- [ ] If performance targets cannot be met with a concrete proving system, this capability is deferred to a later release rather than weakening core deliverables.
