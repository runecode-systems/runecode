# Tasks

## Preservation Manifest

- [ ] Define the `AuditEvidenceSnapshot` object family.
- [ ] Include segment, seal, receipt, verification-report, runtime, attestation, policy, approval, and anchor evidence identities.
- [ ] Make the snapshot cheap to generate and suitable for retention, export, and backfill planning.

## Bundle Manifest

- [ ] Define the `AuditEvidenceBundleManifest` object family.
- [ ] Include bundle scope, created-by tool identity, export profile, included object list, root digests, seal references, verifier identity, trust-root digests, disclosure posture, and redaction list when present.
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

## Retention And Future Portability

- [ ] Use preservation snapshots for retention checks and backfill completeness review.
- [ ] Preserve stable instance identity and exportable canonical evidence without relying on machine-local mutable state.
- [ ] Keep the design future-safe for cross-machine workflows without solving full federation here.

## Verification

- [ ] Add bundle completeness tests.
- [ ] Add selective-disclosure profile tests.
- [ ] Add streaming export tests over large evidence sets.
- [ ] Add offline verification tests using exported bundles alone.
- [ ] Add tests proving manifests are not treated as substitutes for underlying evidence.

## Acceptance Criteria

- [ ] RuneCode can export a verifier-friendly evidence bundle.
- [ ] Bundle export is streaming-friendly.
- [ ] Preserved evidence is sufficient for later verification without mutable ambient state.
- [ ] Export profiles and selective disclosure are explicit.
- [ ] External relying parties can verify bundle provenance independently.
