---
schema_version: 1
id: security/audit-evidence-bundles-and-offline-verification
title: Audit Evidence Bundles And Offline Verification
status: active
suggested_context_bundles:
    - go-control-plane
    - protocol-foundation
---

# Audit Evidence Bundles And Offline Verification

When trusted RuneCode services preserve, package, export, or verify portable audit evidence bundles:

- Treat canonical audit evidence objects and signed receipts as the source of truth; preservation snapshots, bundle manifests, and exported archives are portability helpers and must not become a second authority surface
- Keep `AuditEvidenceSnapshot` focused on preservation scope and canonical evidence identity, while `AuditEvidenceBundleManifest` describes one concrete portable bundle export; do not collapse those roles into one overloaded object
- Preserve explicit identity seams across repository or project identity, repo-scoped product-instance identity, persistent ledger identity, and project-substrate snapshot identity whenever later verification continuity depends on them
- Keep bundle completeness truthful: distinguish canonical objects directly included in the bundle from transitive digest-referenced dependencies that were intentionally not embedded
- Make selective disclosure explicit in bundle metadata; do not imply that omitted or redacted material was included when only its digest identity or summary metadata is present
- Sign manifests intended for external sharing so offline consumers can verify the manifest identity and signer binding without ambient local state
- Keep offline verification independent of RuneCode's UI, live broker state, mutable local databases, or storage-path conventions; verification should run from the exported bundle plus explicit trust roots and verifier inputs
- Stream large bundle export through typed export events rather than requiring one in-memory archive assembly path for the authoritative implementation
- Fail closed when canonical evidence, manifest references, archive contents, or signed-manifest identity disagree; do not silently continue with a partially verified portable evidence set
- Treat bundle export receipts, disclosure declarations, and similar meta-audit evidence as part of the verification surface for later review; portable export is itself a sensitive evidence action, not an invisible implementation detail
- Keep bundles and offline verification topology-neutral: local-device and larger-deployment workflows should vary by scale, retention, or target count, not by a different evidence-trust model
