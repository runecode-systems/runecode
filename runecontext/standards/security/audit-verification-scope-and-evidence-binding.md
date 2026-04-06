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
- Preserve fail-closed behavior when verification scope, signer binding, receipt provenance, or persisted digest evidence is ambiguous
