# Design

## Overview
Define a future additive proof lane that can ingest exported canonical RuneCode evidence, reconstruct proof-ready bindings, generate or aggregate additional proofs on stronger infrastructure, and optionally publish public-assurance artifacts without replacing RuneCode's authoritative local trust model.

## Key Decisions
- The local proof core remains authoritative for any proof family RuneCode supports everywhere.
- The future remote lane is additive and asynchronous. It must never become RuneCode's only correctness path.
- The future public-assurance lane, if enabled later, must reuse the same proof bindings, statement families, and logical normalization profiles rather than inventing a second public-only trust model.
- Machines that do not enable the remote or public lane must still retain the canonical evidence and proof-binding substrate needed so those lanes can be enabled later without losing historical backfill coverage.
- Remote and public lanes may use broader proof portfolios, recursion, or proof aggregation later, but they must continue to consume the same canonical local evidence model.

## Authority Boundaries

### Local Authority
- Authoritative local audit evidence, runtime evidence, attestation evidence, verified project-substrate bindings, proof-binding sidecars, and local proof-verification records are produced and verified inside trusted RuneCode services.
- Local verification remains authoritative for any proof family implemented in local RuneCode.

### Remote Authority
- A future operator-private remote proof service may ingest exported canonical evidence and proof-binding sidecars.
- Remote proofs are additive derived evidence only.
- Remote services must not replace the authoritative local audit ledger, the authoritative local runtime or attestation evidence, the authoritative verified project-substrate identities, or authoritative local verification results.
- Remote services must not invent a second approval, policy, or project-truth model.

### Public Assurance Posture
- A future public-assurance lane may publish proof objects and public evidence projections for external consumers.
- Public publication remains derivative and must not become RuneCode's internal authorization surface.

## Shared Binding Rules
- Local, remote, and public proof lanes must share the same:
  - `statement_family`
  - `statement_version`
  - logical `normalization_profile_id`
  - source authoritative digests
  - typed assurance bindings
- Different lanes may use different proving backends only if they preserve the same logical meaning and proof-binding identities.

## Local Persistence Requirements Before Remote Enablement

### Non-Optional Local Retention Rule
- Every RuneCode machine must preserve enough canonical proof-relevant source evidence and proof-binding information locally even when the remote or public proof lane is disabled or not configured.
- The purpose is to guarantee that a later remote or public lane can backfill the full retained history from that machine without asking the machine to recreate missing ambient context.
- The absence of remote-lane configuration must never be treated as permission to omit proof-relevant retention.

### Minimum Evidence Classes
- The local substrate retained for future backfill should include at least:
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

### Persistence Expectations
- The local retention design should prefer keeping authoritative evidence once rather than duplicating it solely for the future remote lane.
- Evidence may live in trusted local persistence until later export rather than in pre-staged remote-ready copies, as long as later export can still reconstruct a self-contained bundle without ambient context.
- The future lane should assume that some deployments will enable it months or years after the local evidence was first recorded.

## Export-Bundle Model

### Export Principle
- The future remote lane should ingest self-contained exported canonical evidence bundles rather than ambient paths into live local stores.

### Bundle Contents
- Export bundles intended for proof backfill should carry at least:
  - canonical audit evidence
  - canonical runtime and attestation evidence needed by supported proof families
  - validated project-substrate bindings when required by supported proof families
  - proof-binding sidecars
  - manifest and authenticity material needed to verify the bundle itself
  - bundle-level provenance identifying the exporting RuneCode instance, export time, and covered evidence ranges
- Runtime and attestation evidence in the bundle must be immutable evidence objects or canonical envelopes addressed by digest, not only live-store references such as `run_id` lookups.

### Bundle Protocol Surfaces
- The remote lane must define checked-in protocol schemas for at least:
  - `ProofBackfillExportBundle`
  - `ProofBackfillBundleManifest`
  - `ProofBackfillCoverageRange`
  - `ProofBackfillAuthenticityEnvelope`
  - any remote proof write-back artifact or receipt family used to return additive proof results to a RuneCode node
- `ProofBackfillBundleManifest` should include at least:
  - manifest schema and version
  - exporter identity
  - export timestamp
  - supported statement families and normalization profiles covered by the bundle
  - redaction or disclosure profile identity
  - canonical digests for every included evidence object or archive member
  - declared coverage ranges by authoritative stream
  - predecessor or checkpoint material needed for anti-rollback evaluation
  - authenticity-envelope digest or equivalent signature binding

### Coverage-Range Semantics
- Coverage must be declared explicitly rather than inferred from filenames or directory layout.
- Each bundle should declare one or more authoritative coverage windows keyed by typed stream identity such as `emitter_stream_id` plus seal or segment progression.
- Coverage windows must state whether they are complete or partial for the declared stream range so a remote service can distinguish intentionally partial export from accidental truncation.
- Remote ingest and backfill rules should use those declared ranges when deciding whether historical proof gaps are expected, missing, or suspicious.

### Bundle Authenticity
- Export bundles must be verifiable without trusting the transport path used to deliver them.
- Bundle authenticity should be pinned to canonical digests, bundle manifests, and locally trusted verification evidence rather than to filenames or directory layout.
- The authenticity model must define how the exporter signs or otherwise authenticates the bundle manifest and how the remote service validates that authenticity against reviewed trust roots.
- Authenticity validation must also cover any bundle-level archive container so the manifest cannot be replayed over different payload bytes.

### Anti-Rollback Posture
- Remote ingest must track exporter checkpoints or predecessor relationships strongly enough to detect silent rollback, stream truncation, or replay of older bundle manifests as if they were current.
- A newly ingested bundle must not silently reduce the highest fully covered range previously accepted for the same exporter and authoritative stream unless the bundle is explicitly marked as partial or diagnostic and the remote service preserves that distinction.
- Remote services must treat manifest regression, conflicting digests for the same typed identity, or inconsistent predecessor links as hard ingest failures or explicit disagreement evidence.

## Remote Ingest And Backfill Model

### Ingest Rules
- A future remote proof service should be able to rebuild its proof-work queue entirely from exported bundles plus configured proof-family support.
- Remote ingest must not require live reach-back into the originating machine's local storage.
- Remote ingest should verify the bundle's authenticity and canonical source evidence before any proof work starts.
- Remote ingest must validate the declared coverage ranges, predecessor or anti-rollback material, disclosure profile, and statement-family support before accepting the bundle into proof-work scheduling.

### Backfill Rules
- The remote service may backfill proofs for all retained history, not just newly exported events.
- Backfilled proofs should be written back as additive derived evidence using the same proof contract and the same proof-binding identities as locally generated proofs.
- If the remote service later uses recursive or aggregate proofs, those aggregate artifacts must still refer back to the same canonical statement families and source identities.
- Remote proof write-back must be validated by the receiving RuneCode node before persistence.
- Write-back validation must check at least:
  - remote bundle or provenance references
  - proof-binding digest and statement-family identity
  - artifact-declared setup and verifier identity against the locally trusted acceptance posture for that remote result family
  - source-ref integrity and non-overwrite rules for authoritative local records
- Remote write-back must remain additive derived evidence and must not overwrite local authoritative proof-verification records.

### Disagreement Posture
- If local and remote proof results disagree, local authoritative verification posture remains the source of truth for RuneCode's internal assurance model.
- Remote disagreement should be surfaced as additive diagnostic evidence, not as an override of local trust outcomes.

## Cross-Machine Evidence Model

### Identity Rules
- Concurrent RuneCode execution across more than one machine on the same project produces distinct authoritative evidence streams.
- Cross-machine merge identity should be based on authoritative stream and segment identities such as `emitter_stream_id`, segment identity, and seal identity, not on project-level deduplication.

### Merge Posture
- A future remote proof service should treat each machine's evidence as an independent authoritative stream and merge them by typed identity rather than attempting ambient deduplication.
- Shared project history in the future lane is the union of preserved authoritative streams, not a rewritten consolidated ledger.

## Future Public-Assurance Lane

### Publication Posture
- The future public-assurance lane should start after the operator-private remote lane, not before it.
- Public publication should reuse the same proof bindings and canonical source identities already established for local and remote proof work.
- The public lane may publish selected proof objects, aggregate proofs, public-input projections, or authenticity manifests, but it must not redefine the local trust model.

### Information-Asymmetry Use Case
- The public-assurance lane is the clearest consumer of proof-disclosure rules where a verifier sees public inputs and proof objects without receiving the full authoritative private source payload.
- This lane is a major reason the logical normalization profile's proof-disclosure split must remain separate from the source schema's `x-data-class` semantics.
- The remote and public lanes must define explicit disclosure or redaction profiles so bundle producers and consumers agree on which evidence classes may be omitted, summarized, or projected for a given export or publication mode.
- Proof-required evidence classes must not be removed by a disclosure profile if their absence would make later local or remote validation ambiguous.

## Current Design Gap
- This change is not ready for implementation until the concrete bundle protocol, manifest schema, coverage semantics, anti-rollback rules, disclosure profiles, and remote write-back validation rules above are specified in enough detail to produce authoritative protocol schemas and deterministic verification logic.

## Recursive Proofs And Aggregation

### Potential Later Role
- Recursive proofs are not part of the local `v0` lane.
- A future remote or public lane may use recursive proofs or proof aggregation to compress many historical proofs into one small external artifact.
- If RuneCode adds recursion later, it should do so on the remote or public lane first, where stronger proving infrastructure and asynchronous workflows are expected.

### Why Not In `v0`
- Recursion-first proof architecture would be a major proving-system redesign relative to the intended local `gnark` plus `Groth16` starting point.
- That complexity is not justified before RuneCode validates one narrow local proof family.

## Alternative Architecture: Additive Dual-Commitment Proof Bridge

### Problem This Option Tries To Solve
- Direct in-circuit membership against RuneCode's authoritative SHA-256 Merkle tree may prove too expensive if the path cost materially misses the local `v0` performance gates.

### The Alternative
- Keep the authoritative `AuditSegmentSeal` SHA-256 Merkle root exactly as-is.
- Add an additive proof-friendly segment-binding sidecar at seal time that binds:
  - the authoritative seal identity
  - the authoritative SHA-256 Merkle root
  - a second proof-friendly root over the same ordered records
- Future circuits could then prove membership against the proof-friendly root while trusted verification checks the sidecar's binding to the authoritative seal and authoritative root.

### Benefits
- Dramatically lower circuit cost for membership proofs.
- Preserves the authoritative audit ledger and authoritative seal format.
- Creates a reusable bridge that may support broader proof portfolios later.

### Costs And Risks
- Changes the exact proof semantics relative to direct authoritative in-circuit membership.
- Moves more correctness weight into the additive proof-bridge sidecar and its trusted derivation process.
- Requires a careful new threat-model writeup to ensure RuneCode does not accidentally weaken the assurance claim.
- Introduces more additive cryptographic state at seal time, which increases implementation and review scope.

### Decision Rule
- RuneCode should not adopt this architecture before first attempting direct authoritative-Merkle membership for the local `v0` proof family.
- If the direct design misses the documented gates badly, RuneCode should stop and perform an explicit architecture review comparing:
  - direct authoritative in-circuit membership
  - additive dual-commitment proof bridge
  - feature deferral
- RuneCode should switch only if the measured benefit is clearly worth the added semantics and review complexity.

## Future Proof Portfolio Growth
- The future remote lane may support a broader or faster-evolving proof portfolio than the local lane.
- That broader portfolio must still consume canonical evidence and proof-binding sidecars produced under the same reviewed semantics.
- Future proof families that depend on project context, runtime posture, attestation, or anchoring should continue to reuse the same typed assurance identities already established elsewhere in RuneCode.

## What This Does Not Change
- The remote or public lane does not justify a separate local architecture for constrained devices.
- Every RuneCode node should still capture the same canonical evidence and run the same reviewed local trust semantics.
- If the future lane is unavailable, RuneCode's core security and assurance architecture must still stand on its own.
