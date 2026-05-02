# Tasks

## Phase 0: Freeze Scope And Terminology

- [ ] Freeze the meaning of audit plane, verification plane, tamper-evident audit trail, canonical evidence, derived surface, and evidence bundle.
- [ ] Freeze the rule that the chain of authority and side effects is the primary verification target.
- [ ] Freeze the initial object-family, receipt-kind, and reason-code expansion shape for `v0`.
- [ ] Keep proof-specific work explicitly out of the verification-plane foundation lane.

## Child Workstream Tracking

- [ ] Track `CHG-2026-056-8c75-audit-evidence-index-record-inclusion-v0` to completion.
- [ ] Track `CHG-2026-055-546a-verification-evidence-preservation-bundle-export-v0` to completion.
- [ ] Track `CHG-2026-058-04e9-verification-coverage-expansion-v0` to completion.

## Cross-Feature Coordination

- [ ] Keep canonical evidence under trusted control and keep derived operational surfaces rebuildable.
- [ ] Keep one verification architecture across constrained and scaled deployments.
- [ ] Keep degraded posture explicit rather than silently falling back to weaker assurance.
- [ ] Ensure denials, failures, deferrals, and overrides remain first-class evidence outcomes.
- [ ] Ensure portable evidence bundles remain independently verifiable outside RuneCode's UI and database.
- [ ] Ensure runtime evidence, attestation evidence, approval scope, and policy posture remain part of the same provenance chain rather than separate auxiliary views.

## Phase Sequencing

- [ ] Phase 1: deliver the generic audit-evidence index with deterministic rebuild and fail-closed mismatch handling.
- [ ] Phase 2: deliver `AuditRecordInclusion` with trusted local resolution from record digest to sealing checkpoint.
- [ ] Phase 3: deliver evidence-preservation snapshots, verifier-friendly bundle manifests, export profiles, and streaming export.
- [ ] Phase 4: expand coverage for control-plane provenance, approval basis, provider and egress provenance, meta-audit, degraded-posture summaries, and negative-capability receipts.
- [ ] Phase 5: strengthen verification reports with verifier identity, trust-root identity, missing-evidence findings, and clearer anchoring posture.
- [ ] Phase 6: harden invariants and performance without changing trust semantics.

## Acceptance Criteria

- [ ] RuneCode can trace a material artifact or mutation back to a specific run, actor, policy set, runtime identity, and approval chain.
- [ ] RuneCode can detect rewriting or reordering of material audit history.
- [ ] RuneCode can export portable evidence bundles and verify them independently.
- [ ] RuneCode can resolve where a record lives and which seal commits to it.
- [ ] RuneCode can report degraded posture explicitly.
- [ ] RuneCode records denials, deferrals, and overrides.
- [ ] RuneCode preserves immutable runtime and attestation evidence by digest identity.
- [ ] RuneCode preserves enough evidence for future backfill and cross-machine export.
- [ ] RuneCode does not require a different architecture for small devices.
- [ ] RuneCode does not weaken trust boundaries or introduce a second truth surface.
