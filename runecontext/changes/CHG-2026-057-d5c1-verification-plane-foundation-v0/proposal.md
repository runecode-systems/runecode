## Summary
Build RuneCode's verification plane and audit plane around inspectable, signed, replayable, attested evidence rather than opaque proof systems, mutable database state, or exhaustive low-level logs that do not answer operator and auditor questions.

This project change is the umbrella for the verification-plane foundation. It freezes the shared architecture, terminology, non-negotiables, object model, verifier posture, and sequencing rules that the implementation features beneath it must inherit.

## Problem
RuneCode already has important audit, policy, protocol, and runtime-assurance building blocks, but it does not yet have one canonical project change that states what the verification plane is actually for and what evidence model it should optimize around.

Without that foundation, future work could drift in damaging ways:

- treating mutable indexes, UI views, or convenience databases as the authoritative history
- optimizing first for opaque proof machinery instead of inspectable evidence and independent verification
- capturing giant transcripts or undifferentiated logs without strong provenance or access controls
- splitting into separate small-device and scaled-deployment verification architectures
- hiding degraded assurance instead of making it explicit, signed, and reviewable
- losing critical evidence because export, retention, and backfill needs were not designed up front
- expanding coverage unevenly across approvals, provider egress, runtime identity, anchoring, and verifier identity

RuneCode's most important verification question is not whether an internal model process can be proven in the abstract. It is whether a third party can inspect evidence and answer:

"Why should I trust that this artifact, decision, or action came from the run it claims to have come from, under the policy, runtime, and approval conditions it claims to have used?"

That requires a project-level foundation that keeps the source of truth in trusted canonical evidence, keeps verification deterministic where possible, and makes degraded posture, denials, deferrals, and overrides first-class outcomes.

## Proposed Change
- Define the verification plane foundation as one project-level lane covering both the audit plane and the verification plane.
- Freeze the core architecture around:
  - content-addressed artifacts and evidence objects
  - append-only, hash-linked audit history
  - signed receipts for material decisions and outputs
  - deterministic replay for deterministic subsystems
  - runtime identity evidence and attestation where execution identity matters
  - external anchoring as an anti-rewrite and anti-backdating strengthening layer
  - portable evidence bundles with independent verification
  - selective disclosure and privacy-aware export profiles
  - explicit degraded-assurance posture rather than silent fallback
- Keep one architecture and one trust model across constrained local devices and larger deployments.
- Separate canonical evidence from derived operational surfaces, and keep derived surfaces rebuildable rather than authoritative.
- Freeze explicit identity separation among project or repository identity, repo-scoped product-instance identity, persistent ledger identity, and project-substrate snapshot identity.
- Require persistent ledger identity as a first-class foundation seam for export, restore, reconcile, and future federation continuity.
- Define the recommended evidence object families, receipt kinds, verifier findings, and coverage gaps that must be closed for `v0`.
- Require truthful completeness semantics for portable evidence: directly included canonical objects versus transitive digest-reference dependencies must be distinguishable.
- Harden prepared-record evidence seams now so downstream publication durability barriers and crash reconcile can bind exact action intent to prior evidence checkpoints without implementing federation behavior in this lane.
- Make this parent project responsible for sequencing and integration posture while child feature changes carry the implementation detail.
- Track three child workstreams under this umbrella:
  - `CHG-2026-056-8c75-audit-evidence-index-record-inclusion-v0`
  - `CHG-2026-055-546a-verification-evidence-preservation-bundle-export-v0`
  - `CHG-2026-058-04e9-verification-coverage-expansion-v0`
- Keep proof-specific generation, verification, and protocol surfaces out of the foundation lane.
- Do not overload bundle manifests or preservation snapshots into federation authority primitives.

## Why Now
RuneCode is moving toward a more complete end-to-end product surface where users, reviewers, operators, companies, and auditors will all need stronger answers about provenance, authority, runtime identity, approvals, provider usage, anti-tamper history, and exportable verification.

The project already has enough audited and trusted control-plane machinery that the next risk is not missing raw data. The next risk is missing the right architecture for preserving, linking, exporting, and independently verifying that data.

Freezing the foundation now prevents later work from treating proof systems, UI projections, or mutable indexes as the primary history model and gives the repository one durable planning surface before more feature-specific verification work lands.

## Assumptions
- RuneCode remains a security-first automation platform where the main verification risks live in authority, policy, runtime identity, trust-boundary crossing, artifact lineage, approval scope, and anti-tamper history.
- The trusted domain remains the only authoritative owner of canonical evidence.
- The runner remains untrusted and must not gain direct access to trusted evidence internals or second-path authority.
- Cross-boundary contracts remain schema-driven and fail closed.
- Hardware-backed attestation will not exist everywhere, so the system must emit an explicit weaker measured-launch posture when stronger attestation is unavailable rather than pretending equal assurance.
- Deterministic replay is practical and desirable for deterministic subsystems such as policy, schema validation, approval preconditions, artifact-flow checks, and verifier report generation from committed evidence.
- Exact token-by-token replay of LLM behavior is not a foundation promise.
- Portable evidence bundles should be verifiable without RuneCode's UI or internal database.

## Out of Scope
- Implementing product code directly in this umbrella change.
- Treating zero-knowledge proofs as the foundation for `v0`.
- Adding proof-generation CLI surfaces, proof-verification CLI surfaces, proof-family-specific broker APIs, proof-specific protocol objects, or proof-specific setup-material plumbing as part of the mainline foundation.
- Creating a second authorization engine, a second project-truth surface, or a second deployment-specific trust model.
- Treating mutable search indexes, watch views, or dashboards as authoritative evidence.
- Promising exact replay for non-deterministic LLM output or ambient external service behavior.
- Defaulting to raw prompt, raw provider payload, or raw secret retention when digest-addressed artifacts and controlled references are sufficient.

## Impact
This project change gives RuneCode one canonical foundation for verification-plane work.

If the project follows this change, RuneCode will have a reviewable plan for:

- tracing a material artifact or mutation back to a specific run, actor, policy set, runtime identity, and approval chain
- detecting rewriting, deletion, or reordering of material audit history through canonicalized, sealed, hash-linked evidence
- exporting portable evidence bundles that can be verified independently
- resolving where a record lives and which seal commits to it without treating a mutable index as authoritative
- recording degraded posture, denials, deferrals, and overrides explicitly rather than implicitly
- preserving runtime and attestation evidence by digest identity
- preserving enough evidence for retention, backfill, and future cross-machine workflows
- keeping the same verification semantics on constrained local systems and larger deployments

The strongest immediate value is a trustworthy evidence system that users, companies, and auditors can inspect and verify. This project change exists to keep that as the architectural center of gravity.
