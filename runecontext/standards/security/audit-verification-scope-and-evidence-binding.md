---
schema_version: 1
id: security/audit-verification-scope-and-evidence-binding
title: Audit Verification Scope And Evidence Binding
status: active
suggested_context_bundles:
    - go-control-plane
    - protocol-foundation
---

# Audit Verification Scope And Evidence Binding

When trusted services verify locally persisted audit evidence:

- Bind signer-evidence references to the actual detached-envelope signer; referenced signer evidence must not point at a different signer identity
- Evaluate import and restore provenance against the verified segment entry, not against unrelated entries in the same receipt payload
- Treat unrelated historical receipts as non-authoritative context and do not let them invalidate the currently verified segment
- Scope derived verification surfaces and summaries to the segment identified by the verification report, not whichever segment happens to be latest on disk
- Recompute and compare persisted frame record digests before building operational audit views; mismatches must fail closed
- Authenticate segment seal envelopes before accepting seal rotation or advancing durable ledger state
- Keep distinct durable input classes for external-anchor evidence records versus external-anchor sidecar objects; verification must not infer one class from the filenames or directory scan results of the other
- When a verification scope references external-anchor sidecars by digest, resolve those digests only from the authoritative sidecar store and fail closed on missing, unreadable, malformed, or digest-mismatched sidecar content
- When rebuilding or loading verification foundations, preserve the evidence/sidecar split exactly as persisted; do not merge, backfill, or heuristically recover one input class from the other
- Preserve fail-closed behavior when verification scope, signer binding, receipt provenance, or persisted digest evidence is ambiguous
