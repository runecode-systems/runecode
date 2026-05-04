# Tasks

## Preservation Manifest

- [ ] Define the `AuditEvidenceSnapshot` object family.
- [ ] Include segment, seal, receipt, verification-report, runtime, attestation, policy, approval, and anchor evidence identities.
- [ ] Make the snapshot cheap to generate and suitable for retention, export, and backfill planning.
- [ ] Include verifier-record, event-contract-catalog, signer-evidence, storage-posture, typed-request, action-request, and control-plane provenance digests where those identities are required for later offline verification.
- [ ] Preserve explicit identity seams for project or repository identity, repo-scoped product-instance identity, persistent ledger identity, and project-substrate snapshot identity when needed for verification continuity.

## Bundle Manifest

- [ ] Define the `AuditEvidenceBundleManifest` object family.
- [ ] Include bundle scope, created-by tool identity, export profile, included object list, root digests, seal references, verifier identity, trust-root digests, disclosure posture, and redaction list when present.
- [ ] Distinguish directly included canonical objects from transitive digest-reference dependencies in manifest semantics.
- [ ] Sign the manifest when the bundle is intended for external sharing.

## Bundle Export

- [ ] Support run-scoped, artifact-scoped, and incident-scoped bundle shapes.
- [ ] Support auditor-minimal, operator-private, and external relying-party export use cases.
- [ ] Keep export streaming-friendly and avoid full in-memory assembly for large bundles.

## Selective Disclosure

- [ ] Define explicit export profiles.
- [ ] Default to digest and typed-metadata export when raw payloads are unnecessary.
- [ ] Record selective-disclosure declarations and redactions in the bundle manifest.

## Offline Verification

- [ ] Support independent verification of exported bundles without RuneCode's UI or internal database.
- [ ] Preserve verifier identity and trust-root identity in included verification artifacts.
- [ ] Ensure missing-evidence or degraded-posture findings remain visible when verifying a bundle offline.
- [ ] Recompute verification conclusions from exported canonical evidence when the bundle includes the required verification inputs, rather than treating included verification reports as the only verification basis.
- [ ] Fail closed or degrade explicitly when a bundle omits evidence required for verification recomputation.

## Retention And Future Portability

- [ ] Use preservation snapshots for retention checks and backfill completeness review.
- [ ] Preserve stable instance identity and exportable canonical evidence without relying on machine-local mutable state.
- [ ] Preserve persistent ledger identity as required continuity state for export, restore, and later federation-safe workflows.
- [ ] Keep the design future-safe for cross-machine workflows without solving full federation here.
- [ ] Preserve enough evidence identity for import, restore, and later merge-oriented workflows without creating a second truth surface.
- [ ] Do not overload snapshots or bundle manifests into replication-checkpoint or federation-authority roles.

## Verification

- [ ] Add bundle completeness tests.
- [ ] Add selective-disclosure profile tests.
- [ ] Add streaming export tests over large evidence sets.
- [ ] Add offline verification tests using exported bundles alone.
- [ ] Add tests proving manifests are not treated as substitutes for underlying evidence.
- [ ] Add deterministic artifact-scope and incident-scope bundle selection tests.
- [ ] Add offline recomputation tests that verify bundles can be re-verified from exported canonical evidence when inputs are complete.

## Acceptance Criteria

- [ ] RuneCode can export a verifier-friendly evidence bundle.
- [ ] Bundle export is streaming-friendly.
- [ ] Preserved evidence is sufficient for later verification without mutable ambient state.
- [ ] Export profiles and selective disclosure are explicit.
- [ ] External relying parties can verify bundle provenance independently.
- [ ] Artifact-scoped and incident-scoped bundles can be resolved and exported deterministically from canonical evidence.
- [ ] Offline verification can distinguish bundle-integrity success from evidence-sufficiency for recomputed verification.
