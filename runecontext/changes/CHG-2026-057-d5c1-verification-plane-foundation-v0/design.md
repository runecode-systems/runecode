# Design

## Overview
RuneCode should build `Verification Plane Foundation v0` around inspectable canonical evidence, not around hidden reasoning or mutable operational views.

The audit plane creates, canonicalizes, seals, preserves, and optionally anchors evidence. The verification plane resolves, reconstructs, replays, attests, exports, and independently verifies what that evidence means. This project change defines the shared architecture for both planes and delegates implementation detail to the child feature changes beneath it.

The foundation should optimize for:

- inspectable evidence
- deterministic verification where determinism is realistic
- portable evidence bundles
- strong provenance links between actor, run, policy, runtime, approvals, and outputs
- explicit degraded posture
- independent verification outside RuneCode's UI and database

It should not optimize first for:

- opaque cryptographic claims that hide the underlying evidence
- giant undifferentiated logs that are expensive but not decision-useful
- separate verification architectures for small devices and large deployments

## Core Definitions

### Audit Plane
The audit plane is the part of the system that captures, canonicalizes, seals, preserves, and optionally anchors evidence.

### Verification Plane
The verification plane is the part of the system that resolves, reconstructs, replays, attests, exports, and independently verifies evidence.

### Tamper-Evident Audit Trail
For RuneCode, a tamper-evident audit trail means:

- every material event is recorded in canonical form
- deletion, rewriting, or reordering becomes detectable
- records are linked strongly enough that later edits leave evidence
- sealing and anchoring make silent historical rewriting difficult to hide

It does not mean `tamper-proof`. It means `tampering leaves proof`.

### Signed, Auditable
For RuneCode, signed and auditable means more than signing a blob. It means RuneCode can bind:

- the actor
- the run
- the runtime identity
- the policy set
- the input artifacts
- the approvals and overrides
- the output artifacts
- the relevant timestamps

into signed or hash-committed evidence that an independent party can verify.

### Canonical Evidence
Canonical evidence is any immutable authoritative object that forms part of the source of truth. Examples include canonical audit event payloads, sealed audit segments, signed segment seals, signed approvals, signed policy decisions, runtime evidence snapshots, attestation verification records, external anchor evidence, and final verification reports.

### Derived Surface
Derived surfaces are rebuildable views and indexes that exist to make verification practical. Examples include lookup indexes, search views, watch streams, dashboards, SIEM exports, and cache entries. They are useful, but they must never become the sole history model.

### Evidence Bundle
An evidence bundle is a portable package of canonical evidence and a manifest that lets someone verify a run, artifact, or decision outside the originating machine.

## Non-Negotiables
- Preserve the trust boundary described in `docs/trust-boundaries.md`.
- Do not add runner-side access to trusted state or trusted evidence internals.
- Keep cross-boundary contracts schema-driven and fail closed.
- Do not leak secrets, tokens, or sensitive local paths into logs, evidence, fixtures, exports, or errors.
- Keep the source of truth in the trusted domain.
- Keep the same overall trust model and verification semantics on small devices and scaled deployments.
- Do not create a second project-truth surface or a second authorization engine.
- Do not treat UI views, mutable databases, or search indexes as authoritative evidence.
- Do not silently accept degraded assurance.
- Record denials, failures, deferrals, and overrides, not only successful actions.
- Prefer the smallest correct change, but do not under-scope the evidence needed for future verification.

## The Most Useful Auditable Facts
RuneCode should prioritize evidence for:

- who initiated a run or action
- what identity approved it
- what policy and capability envelope applied
- what data and artifacts crossed trust boundaries
- what secrets, providers, endpoints, or network targets were touched
- what runtime actually executed the work
- what artifacts came out
- what was denied or deferred
- what was degraded or overridden
- what final claims can be independently verified later

RuneCode should not optimize first for:

- exact token-by-token model internals
- raw transcript capture without strong provenance and access controls
- giant undifferentiated logs with no semantic structure

## Why This Fits RuneCode
RuneCode is primarily a security-first automation platform focused on isolation, provenance, and explicit trust boundaries. Its verification plane should therefore be strongest where its real risk lives:

- authority
- policy
- runtime identity
- trust-boundary crossing
- artifact lineage
- approval and override chains
- anti-tamper history

Operators and auditors rarely ask only whether a claim is true. They ask what happened, in what order, under what constraints, using which runtime, based on which inputs, with what approvals, and whether the evidence can be inspected independently. Inspectable signed evidence is therefore a better foundation than opaque proofs.

## Audiences And Required Views

### Individual Users And Operators
These users need confidence and explainability. They want to know what RuneCode touched, what files or artifacts were read and written, whether the network was used, whether secrets were accessed, what policy constrained the run, whether the output really came from a scoped audited run, and how to share provenance without oversharing private content.

The most useful views for them are:

- a readable run timeline
- a touched-artifacts summary
- a capability-use summary
- a degraded-posture summary
- a portable provenance bundle for outputs

### Reviewers And Approvers
These users need decision-quality evidence. They want to know what exact diff, artifact set, or action they are approving, what policy would allow or deny it, what risk or degradation applies, whether their approval scope is narrow and explicit, and what final action consumed the approval.

The most useful views for them are:

- approval-basis evidence
- scope digests
- approval consumption links
- final mutation receipts

### Companies, Security Teams, And Compliance Owners
These users care about governance and risk at scale. They want to know whether least-privilege policy was actually enforced, whether production-affecting actions required approval, which runs touched sensitive repos, data classes, or secrets, how to reconstruct incidents later, how to export evidence into SIEM or GRC systems, how long evidence is retained, and who can read or export it.

The most useful features for them are:

- centralized search over canonical evidence identifiers and derived metadata
- explicit override and exception records
- exportable evidence bundles
- clear degraded posture reporting
- retention and access-control evidence

### Security And Regulatory Auditors
These users care about independent verification. They want traceability from artifact to run to policy to runtime to approval, evidence that required controls actually ran, evidence that denied or deferred paths are visible, proof that records were not silently rewritten later, and verifier identity plus trust-root evidence.

### Incident Response And Forensics Teams
These users need fast reconstruction under pressure. They want to know what happened first, what crossed boundaries, what secrets or providers were involved, whether an action was authorized, denied, or break-glass, whether evidence is intact and complete, and whether any degradation or missing evidence exists.

The most useful features for them are:

- record inclusion lookup
- scope-filtered evidence bundles
- clear causal links
- explicit missing-evidence findings

### External Relying Parties
These are parties who receive outputs from RuneCode but do not run RuneCode themselves. They need portable provenance, independent verification without access to RuneCode's internal database, and minimal-disclosure export profiles.

### Platform, SRE, And Fleet Operators
These users care about system health under real load. They need to know whether sealing is keeping up, whether anchoring is delayed or unavailable, whether verification caches are healthy, whether a node is silently running in reduced assurance, and whether backpressure is causing evidence gaps or deferred work.

### Privacy, Legal, And Data Governance Teams
These users care about what left the boundary and what remains discloseable. They need data-class movement, provider and endpoint visibility, secret lease posture, retention policy evidence, and privacy-aware export profiles.

### Managed-Service And Multi-Tenant Operators
These users need tenant-isolation evidence, per-tenant exportability, proof that one tenant's state did not bleed into another, and stable instance identity across restarts and migrations.

## Auditor Questions The Foundation Must Make Routine
- Show how this artifact traces back to a specific run, actor, policy set, and runtime.
- Show that this production-affecting action required approval.
- Show every run that accessed a particular secret, provider, model, or network destination.
- Show what happened when anchoring was unavailable.
- Show how audit records detect rewriting or backdating.
- Show which verifier version and trust roots were used for a verification report.
- Show who exported an evidence bundle.
- Show whether any required evidence was missing or only degraded.

## Method Evaluation And Recommendation

### Signed Commitments, Hash Chains, And Merkle Commitments
Role: primary integrity substrate.

Recommendation: required as the foundation.

Rationale:

- fast
- easy to inspect
- naturally supports append-only audit history
- supports inclusion proofs and later anchoring

### Signed Receipts And Signed Decision Records
Role: primary authority and control evidence.

Recommendation: required as the foundation.

Rationale:

- they explain what the system is claiming
- they bind actor, policy, scope, and outputs
- they are directly useful to users and auditors

### Deterministic Replay Or Re-Execution
Role: verification for deterministic subsystems.

Recommendation: required where determinism is realistic.

Rationale:

- very auditor-friendly
- cheap to explain and verify
- strong for policy, schema, validation, and gate outcomes

Important limit: do not make exact LLM replay a foundation promise.

### Runtime Attestation And Measured Execution Evidence
Role: execution identity proof.

Recommendation: required where execution identity matters.

Rationale:

- users and auditors care about what actually ran, not only what was intended to run
- it fits execution claims much better than opaque proof systems

Important limit: when hardware-backed attestation is unavailable, emit an explicit weaker measurement record rather than pretending equal assurance.

### Transparency Logging And External Anchoring
Role: anti-rewrite and anti-backdating strengthening layer.

Recommendation: required as a complement, not as the sole verification model.

### Formal Verification
Role: design-time and implementation-time assurance for the verification machinery itself.

Recommendation: strongly recommended for critical invariants such as canonicalization, log consistency, rebuild correctness, and fail-closed behavior.

### Zero-Knowledge Proofs
Role: future optional privacy-preserving layer.

Recommendation: explicitly not the foundation for `v0`.

Rationale:

- higher complexity
- weaker human inspectability
- less aligned with RuneCode's main auditability mission

## Foundation Architecture

### Layer Model
The verification-plane foundation should have these layers:

1. canonical evidence objects
2. audit segmenting and sealing
3. derived evidence index
4. deterministic verifiers and replay helpers
5. runtime identity and attestation bindings
6. anchor evidence and anti-tamper strengthening
7. evidence snapshot and bundle export
8. derived operator, company, and auditor views

### Canonical Evidence Layer
This is the source of truth. It contains canonical audit events, signed envelopes and receipts, sealed segments, immutable sidecars, runtime evidence, attestation evidence, approval and policy evidence, and verification reports.

Rules:

- canonical evidence must be immutable once persisted
- canonical evidence must be content-addressed or strongly tied to a canonical digest identity
- canonical evidence must be canonicalized deterministically before hashing and signing

### Segment And Seal Layer
Audit records should be grouped into segments that are sealed periodically or by policy.

Rules:

- segment seals are signed
- segment seals commit to the exact raw segment file hash and ordered Merkle root
- segment seals link to previous seal digests
- segment seals are authoritative anti-rewrite checkpoints

### Derived Evidence Index Layer
The system needs a generic evidence index for practical lookup, but it is not authoritative. It exists to make verification fast without rescanning the full ledger for routine lookups.

This layer is implemented by `CHG-2026-056-8c75-audit-evidence-index-record-inclusion-v0`.

### Deterministic Verification Layer
This layer should verify:

- schema validity
- digest validity
- signature validity
- segment Merkle and file-hash validity
- seal chain validity
- replayable policy decisions
- approval scope and consumption rules
- attestation verification when relevant
- anchor evidence validity

### Runtime Identity Layer
This layer binds execution claims to runtime evidence. It must preserve image identity, toolchain identity, launch receipt, hardening posture, attestation evidence, and attestation verification results.

### Evidence Bundle Layer
This layer turns local evidence into portable evidence. It should support run-scoped bundles, artifact-scoped bundles, incident-scoped bundles, auditor-minimal bundles, operator-private bundles, and external relying-party bundles.

Independent verification in this layer means more than checking archive integrity or replaying an included report payload. The foundation should support recomputing verification conclusions from exported canonical evidence alone when the bundle carries the required verification inputs, and should fail closed or degrade explicitly when the bundle omits evidence needed for that recomputation.

This layer is implemented by `CHG-2026-055-546a-verification-evidence-preservation-bundle-export-v0`.

### Derived Views Layer
This layer powers UI timelines, audit search, watch views, SIEM exports, and compliance dashboards. These views must always point back to canonical evidence identities.

## Signed Versus Committed Versus Derived

### Must Be Signed
- segment seals
- approval decisions
- high-risk policy decisions or receipts
- capability grant and deny receipts where the action matters outside local ephemeral state
- boundary-crossing authorization receipts
- artifact publication receipts
- runtime launch receipts when they represent a trust claim
- attestation verification records when they represent a trust claim
- final verification reports
- evidence-bundle manifests when exported for external use

### Must Be Hash-Committed Or Content-Addressed
- raw artifacts
- canonical event payloads
- raw provider input and output artifacts when retained
- runtime evidence snapshots
- anchor sidecars
- imported evidence payloads

### Must Be Hash-Linked
- ordered audit records inside streams or segments
- segment seal chain
- bundle manifest references

### Must Remain Derived And Rebuildable
- lookup indexes
- search materializations
- cache entries
- alert summaries

These derived surfaces may use different cache sizes, queue depth, and storage backends across constrained and scaled environments, but they must preserve the same trust roots, verification semantics, evidence objects, and failure posture everywhere.

## Canonical Evidence Object Families
The foundation should reuse existing object families on `main` where they already fit and add new families only where the current model has a clear gap.

Core families:

- audit events
- audit segment seals
- audit receipts
- runtime evidence snapshots
- external anchor evidence
- verification reports

Key additions owned by child workstreams:

- `AuditEvidenceIndex` and `AuditRecordInclusion` in `CHG-2026-056-8c75-audit-evidence-index-record-inclusion-v0`
- `AuditEvidenceSnapshot` and `AuditEvidenceBundleManifest` in `CHG-2026-055-546a-verification-evidence-preservation-bundle-export-v0`
- receipt-kind expansion, verification-report strengthening, degraded-posture coverage, meta-audit coverage, and missing-evidence reason codes in `CHG-2026-058-04e9-verification-coverage-expansion-v0`

Recommended initial receipt kinds are:

- `run_start`
- `run_finalize`
- `policy_decision_allow`
- `policy_decision_deny`
- `capability_grant`
- `capability_deny`
- `approval_resolution`
- `approval_consumption`
- `boundary_crossing_authorized`
- `boundary_crossing_denied`
- `secret_lease_issued`
- `secret_lease_revoked`
- `provider_invocation_authorized`
- `provider_invocation_denied`
- `artifact_published`
- `override_or_break_glass`
- `degraded_posture_summary`
- `network_usage_summary`
- `secret_usage_summary`
- `verification_finalize`

These receipt families should land as canonical protocol and trusted-code work rather than as UI-only summaries. Approval resolution, approval consumption, publication, boundary, override, and summary evidence should remain first-class canonical evidence whenever they express authority, side effects, or explicit absence claims.

## Privacy And Selective Disclosure
RuneCode is not primarily a privacy product, but the verification plane still must avoid oversharing.

The recommended model is:

- store canonical evidence strongly
- make payload visibility policy-aware
- export digests and typed metadata by default where raw payloads are not necessary
- support explicit export profiles

Recommended export profiles are:

- `operator_private_full`
- `company_internal_audit`
- `external_relying_party_minimal`
- `incident_response_scope`

The default bundle should reveal enough to verify provenance without automatically disclosing prompts, code, or secrets that are not necessary for the verifier's purpose.

## Deterministic Replay Guidance
Deterministic replay should be first-class for the parts of RuneCode that can actually be deterministic. Replayable foundations include:

- policy evaluation
- approval precondition evaluation
- schema validation
- artifact-flow checks
- anchor subject selection and request validation
- verification report generation from committed evidence

RuneCode should not promise exact replay for:

- token-by-token LLM output generation
- ambient external service behavior

For non-deterministic or externally dependent behavior, RuneCode should still commit the input artifact digest, request artifact digest, provider, model, endpoint identity, output artifact digest, and policy or authorization receipts.

## Runtime Attestation Guidance
RuneCode should bind execution identity to the verification plane without creating separate architectures for different deployment classes.

Recommended approach:

- use one execution-evidence envelope everywhere
- attach stronger hardware-backed attestation where available
- emit an explicit weaker measured-launch record where it is not available
- keep the same logical fields and verification semantics across both cases

Minimum public execution identity fields are:

- runtime image descriptor digest
- toolchain identity or digest
- hardening posture digest
- session binding digest
- attestation evidence digest where present
- attestation verification record digest where present

Recommended minimum preserved execution evidence fields additionally include:

- isolate identity
- backend kind
- launch context digest
- handshake transcript digest or equivalent session-establishment binding

## Meta-Audit, Completeness, And Negative Evidence
RuneCode should audit the audit plane itself where security and compliance care. That includes evidence export, import, restore, retention changes, trust-root updates, verifier configuration changes, and sensitive evidence views where appropriate.

The verification plane must also support claims about what did not happen. For categories where absence matters, RuneCode should emit explicit final summary receipts so verifiers can determine whether required evidence is missing rather than only validating evidence that happens to be present.

Important examples include:

- if a run claims `attested`, attestation evidence and verification records must exist
- if an artifact crossed a boundary, an authorization receipt must exist
- if a production-affecting mutation happened, approval or explicit policy-exception evidence must exist
- if a run claims no network egress, a final network summary receipt should support that claim

Completeness checks should also cover the case where a production-affecting mutation, publication, or boundary-crossing side effect occurred without the required approval or explicit policy-exception evidence.

## Gaps The Foundation Must Close
- control-plane provenance gap
- approval-basis evidence gap
- provider and egress provenance gap
- degraded-posture gap
- meta-audit gap
- completeness and omission-detection gap
- cross-machine federation gap
- verification-of-verification gap
- negative capability evidence gap

## The Most Important Architectural Distinction
Separate the canonical evidence plane from derived operational surfaces.

Canonical evidence should be:

- immutable
- content-addressed or signed
- independently verifiable
- stored under trusted control

Derived surfaces should be:

- rebuildable
- optional for correctness
- optimized for query and UX

This distinction is critical for correctness, portability, and performance.

## Performance And Scaling Requirements
RuneCode must keep one reviewed verification architecture across constrained and scaled environments.

Rules:

- same trust roots everywhere
- same verification semantics everywhere
- same evidence objects everywhere
- same failure posture everywhere
- different cache sizes, queue depth, storage backends, and retention windows are acceptable only when they do not change trust semantics

Performance guidance:

- keep the hot path semantic, not exhaustive
- hash incrementally
- sign material receipts and segment seals, not every tiny internal event
- batch sealing and external anchoring
- keep deep verification on demand
- keep quick local consistency checks cheap
- cache verification results by immutable identity
- make read models consume persisted verification results instead of re-running deep checks on every refresh
- make bundle export streaming-friendly
- make index rebuild possible from canonical evidence

Recommended additional foundation guidance:

- avoid derived-index implementations that rewrite a monolithic state file on every append or seal at large scale
- prefer append-friendly or sharded trusted derived storage so hot-path work remains close to constant time without changing trust semantics
- keep record-inclusion material compact where possible so exported evidence remains practical on constrained devices and scaled systems alike

Concrete foundation targets:

- routine evidence append should stay near constant time with respect to historical ledger size
- record inclusion lookup should be index-backed and fast enough for interactive use
- verification report retrieval should not require rescanning the full ledger in the common case
- full ledger rebuild and full verification must remain possible from canonical evidence alone
- bundle export should stream large evidence sets without requiring full in-memory assembly

## Protocol And Schema Work
The verification foundation should remain schema-driven and consistent with `runecontext/specs/protocol-schema-bundle-v0.md`.

Recommended new or extended protocol objects are:

- `AuditRecordInclusion`
- `AuditEvidenceSnapshot`
- `AuditEvidenceBundleManifest`
- any needed extensions to existing audit receipt and verification report objects

Recommended registry work is:

- extend the audit receipt kind registry for the recommended receipt kinds
- extend the audit verification reason-code registry

Recommended initial audit verification reason-code additions are:

- `external_anchor_valid`
- `external_anchor_deferred_or_unavailable`
- `external_anchor_invalid`
- `missing_required_approval_evidence`
- `missing_runtime_attestation_evidence`
- `negative_capability_summary_missing`
- `verifier_identity_missing_or_unknown`
- `evidence_export_incomplete`

The exact final set can evolve, but the external-anchor reasons should land early because they make anchoring posture much clearer.

## Recommended Local Code Organization
The exact filenames may differ, but the recommended responsibility split is:

### `internal/auditd/`
- evidence index implementation
- record inclusion resolution
- evidence snapshot generation
- bundle export helpers
- verification report strengthening
- canonical sidecar persistence for new evidence classes
- offline verification replay from exported canonical evidence

### `internal/brokerapi/`
- read-only trusted local API for record inclusion
- read-only trusted local API for evidence snapshot
- explicit trusted local API for bundle export
- explicit trusted local API for local bundle verification if exposed

### `protocol/schemas/`
- new schema definitions for cross-boundary or exported verification objects
- registry extensions

### Tests
- audit index rebuild tests
- inclusion lookup tests
- permission and durability tests
- bundle completeness tests
- degraded-posture tests
- fail-closed tests for missing required evidence
- performance regression checks for append, inclusion lookup, export, and verification-report retrieval
- invariant and model-check updates where verification-plane foundation logic changes critical audit or approval bindings

## Experimental Concepts To Generalize
The foundation should bring back these concepts in generalized form:

- a generic audit-evidence index rather than a proof-specific index
- `AuditRecordInclusion` as a first-class audit feature
- a preservation snapshot or export manifest focused on verification evidence rather than proof backfill
- clear external-anchor reason codes
- correctness-oriented test hardening such as real Merkle roots, previous-seal linkage, runtime evidence fixture seeding, and owner-only permission checks

## Explicit Non-Foundation Work
The foundation should not bring back:

- proof-generation CLI surfaces
- proof-verification CLI surfaces
- proof-family-specific broker APIs
- proof-specific protocol objects and registries
- proof-specific project-substrate gating exceptions
- proof-system dependencies or setup-material plumbing

Those remain optional later layers, not foundation work.

## Workstream Split
This project change owns sequencing and integration posture across three related child features:

- `CHG-2026-056-8c75-audit-evidence-index-record-inclusion-v0`
  - generic audit-evidence index
  - deterministic rebuild
  - fail-closed mismatch handling
  - record inclusion lookup and seal discovery
- `CHG-2026-055-546a-verification-evidence-preservation-bundle-export-v0`
  - evidence-preservation snapshots
  - portable evidence-bundle manifests
  - streaming export
  - selective disclosure and export profiles
- `CHG-2026-058-04e9-verification-coverage-expansion-v0`
  - stronger control-plane provenance
  - approval-basis evidence
  - provider and egress provenance
  - degraded-posture summaries
  - meta-audit coverage
  - missing-evidence detection
  - verifier identity and anchor reason-code strengthening

## Recommended Order Of Implementation

### Phase 0: Freeze Scope And Terminology
Deliver:

- this architecture choice
- verification-plane terminology
- initial object model
- initial receipt kinds
- initial reason-code additions

Acceptance criteria:

- the team shares one meaning of audit plane versus verification plane
- the work is clearly separated from any proof-specific feature lane

### Phase 1: Generic Audit-Evidence Index
Deliver:

- generic local evidence index in trusted code
- incremental updates for append and seal operations
- deterministic rebuild from canonical evidence
- fail-closed refresh behavior on mismatch

Acceptance criteria:

- record inclusion lookup no longer requires rescanning the full ledger in the common case
- seal discovery and previous-seal lookup are index-backed
- rebuild from canonical evidence produces the same result as incremental updates

### Phase 2: AuditRecordInclusion
Deliver:

- trusted API and local helper for record inclusion
- enough output to map a record to its segment and sealing checkpoint
- optional inclusion material if cheap and stable to provide

Acceptance criteria:

- incident-response and verification flows can resolve a record quickly
- inclusion output is independently checkable against canonical evidence

### Phase 3: Verification Evidence Snapshot And Bundle Manifest
Deliver:

- preservation manifest listing required evidence identities
- bundle manifest format
- export profiles
- streaming bundle export

Acceptance criteria:

- RuneCode can export a verifier-friendly evidence bundle
- preserved evidence is sufficient for later verification without mutable ambient state

### Phase 4: Coverage Expansion
Deliver:

- stronger control-plane provenance
- approval-basis evidence
- provider and egress provenance
- meta-audit coverage
- negative capability summary receipts

Acceptance criteria:

- the most important auditor questions can be answered from canonical evidence
- degraded posture is explicit, not hidden

### Phase 5: Verifier Strengthening
Deliver:

- verification reports that include verifier and trust-root identity
- missing-evidence findings
- improved anchoring posture reporting
- offline bundle verification path

Acceptance criteria:

- verification can be repeated independently on exported evidence
- missing evidence and degraded posture produce explicit findings

### Phase 6: Formal And Performance Hardening
Deliver:

- stronger invariant tests and model checks where appropriate
- performance gates for index rebuild, inclusion lookup, bundle export, and verification report generation

Acceptance criteria:

- the same architecture behaves acceptably on constrained and scaled systems
- optimizations do not change trust semantics

## Foundation Acceptance Criteria
The verification-plane foundation is successful when all of these are true:

- RuneCode can trace a material artifact or mutation back to a specific run, actor, policy set, runtime identity, and approval chain.
- RuneCode can detect rewriting or reordering of material audit history.
- RuneCode can export portable evidence bundles and verify them independently.
- RuneCode can resolve where a record lives and which seal commits to it.
- RuneCode can report degraded posture explicitly.
- RuneCode records denials, deferrals, and overrides.
- RuneCode preserves immutable runtime and attestation evidence by digest identity.
- RuneCode preserves enough evidence for future backfill and cross-machine export.
- RuneCode does not require a different architecture for small devices.
- RuneCode does not weaken trust boundaries or introduce a second truth surface.

## Useful Verification Queries
The foundation should make queries like these practical:

- Given an artifact digest, show the originating run, runtime, policy decision, approval chain, and final verification report.
- Given a record digest, show the segment, seal, and inclusion details.
- Show all runs that used a given provider profile, model, or network target.
- Show all runs that issued a given secret lease or accessed a given data class.
- Show all actions executed under degraded posture.
- Show all runs whose anchoring is valid, deferred, or invalid.
- Export all evidence relevant to one incident scope.
- Verify a bundle offline and show verifier identity and findings.

## Practical Design Rules
- Prefer signed receipts for material decisions instead of trying to sign everything.
- Prefer canonical digests for large payloads and store raw payloads as artifacts only when needed.
- Keep indexes rebuildable.
- Keep export bundles streamable.
- Keep privacy profiles explicit.
- Bind trust claims to runtime evidence and approvals, not only to final outputs.
- Do not depend on filesystem modification times for authoritative ordering decisions.
- Do not rely on ambient repository state during verification when a committed digest can be used instead.
- Do not treat convenience lookup keys like `run_id` as sufficient historical evidence.

## Final Recommendation
RuneCode should build `Verification Plane Foundation v0` around:

- signed immutable evidence objects
- hash-linked append-only history
- deterministic verification for deterministic subsystems
- execution identity evidence and attestation
- external anchoring as a strengthening layer
- preservation and export of canonical evidence
- privacy-aware, independently verifiable evidence bundles

The strongest immediate value is not a proof system. The strongest immediate value is a trustworthy evidence system that users, companies, and auditors can inspect, verify, and rely on.
