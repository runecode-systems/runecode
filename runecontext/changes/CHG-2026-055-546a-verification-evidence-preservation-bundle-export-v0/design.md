# Design

## Overview
This feature turns RuneCode's local canonical evidence into portable verification evidence.

It introduces two new derived-but-essential object families:

- `AuditEvidenceSnapshot`: a preservation manifest that lists the evidence identities RuneCode must retain for future verification and backfill
- `AuditEvidenceBundleManifest`: a verifier-friendly description of an exported bundle

The design must preserve a strict rule: these objects help preserve and export canonical evidence, but they do not replace canonical evidence as the source of truth.

## Goals
- Preserve the minimum evidence identities required for future verification and backfill.
- Export portable evidence bundles for multiple scopes and disclosure profiles.
- Keep bundle export streaming-friendly.
- Keep bundle verification independent of RuneCode's UI and database.
- Preserve privacy by default through explicit selective-disclosure profiles.

## Non-Goals
- Making snapshots or bundle manifests authoritative substitutes for canonical evidence.
- Solving full federation or global transparency-log design in this lane.
- Exporting raw payloads by default when digest-addressed artifacts and typed metadata are sufficient.

## `AuditEvidenceSnapshot`
Purpose: capture the minimum set of canonical evidence identities that must be preserved or exported for future verification and backfill.

Recommended fields:

- segment ids
- segment seal digests
- audit receipt digests
- audit verification report digests
- verifier record digests
- event contract catalog digests
- signer evidence digests where used
- storage posture digests where used
- runtime image descriptor digests
- attestation evidence digests
- attestation verification record digests
- applied hardening posture digests
- session binding digests
- project substrate snapshot digests where relevant
- typed request digests
- action request digests
- policy decision digests
- required approval ids
- approval request digests
- approval decision digests
- external anchor evidence digests
- external anchor sidecar digests
- provider invocation digests once available
- secret lease digests once available

Rules:

- this object is a preservation manifest, not a substitute for the evidence itself
- it should be cheap to generate
- it should support export planning, retention checks, and backfill completeness checks
- it should preserve enough verification-contract, signer, runtime, approval, and control-plane identity to support later offline re-verification from exported canonical evidence

## `AuditEvidenceBundleManifest`
Purpose: describe a portable evidence bundle in a verifier-friendly way.

Recommended fields:

- bundle id and version
- created at
- created by tool identity
- export profile
- scope
- included object list with family, digest, path, and size
- root digests and seal references
- verifier identity and trust-root digests used to build any included verification report
- selective-disclosure declaration
- redaction list if any

Rules:

- sign the manifest when the bundle is intended for external sharing
- keep the format streaming-friendly
- do not require loading the whole bundle into memory to verify it
- preserve enough verification-input identity that external parties can determine whether the bundle is sufficient for recomputed verification, not just payload-integrity inspection

## Bundle Scopes And Profiles

### Supported Scope Shapes
The feature should support at least:

- run-scoped bundles
- artifact-scoped bundles
- incident-scoped bundles
- auditor-minimal bundles
- operator-private bundles
- external relying-party bundles

### Recommended Export Profiles
- `operator_private_full`
- `company_internal_audit`
- `external_relying_party_minimal`
- `incident_response_scope`

The default export posture should reveal enough to verify provenance without automatically disclosing prompts, code, or secrets that are not necessary for the verifier's purpose.

## Streaming Export Rules
- bundle export must be streaming-friendly
- large evidence sets must not require full in-memory assembly
- object references in the manifest must allow incremental verification by path and digest
- export should remain compatible with local retention and later restore workflows

## Offline Verification Rules
- exported evidence bundles must be independently verifiable without trusting RuneCode's UI or internal database
- bundle manifests should preserve verifier identity and trust-root digests when a verification report is included
- bundle verification should make degraded posture and missing-evidence findings visible rather than hiding them behind bundle creation success
- when a bundle includes the required verification inputs, offline verification should be able to recompute verification conclusions from exported canonical evidence rather than only replaying included verification reports
- when a bundle omits required verification inputs, offline verification should fail closed or degrade explicitly with machine-readable findings describing the missing evidence

## Privacy And Selective Disclosure
RuneCode should store canonical evidence strongly but expose it according to policy.

Export defaults should prefer:

- digests
- typed metadata
- controlled artifact references

Raw prompts or provider payloads should not be exported by default when a digest, data classification, and controlled reference are enough.

The manifest should declare:

- which profile was used
- whether selective disclosure was applied
- what redactions, if any, were made

## Cross-Machine And Retention Posture
This feature does not solve full federation, but it must avoid closing the door on it.

Required rules:

- preserve stable instance identity
- preserve exportable canonical evidence
- do not rely on machine-local mutable state as the only history
- support retention checks and export completeness review from preserved evidence identities
- keep exported evidence and manifests future-safe for cross-machine import, restore, and merge-oriented workflows without requiring a second truth surface

## Trusted Surfaces

### `internal/auditd/`
This feature should own:

- evidence snapshot generation
- bundle export helpers
- manifest creation
- any verifier-friendly export packaging logic

### `internal/brokerapi/`
This feature should expose:

- a trusted local API for evidence snapshots
- an explicit trusted local API for bundle export
- an explicit trusted local API for local bundle verification if that surface is exposed

### `protocol/schemas/`
Any exported verification object that crosses a reviewed boundary should be defined canonically in protocol schemas and registries.

## Test Requirements
- bundle completeness tests
- selective-disclosure profile tests
- streaming export tests over large evidence sets
- retention and backfill completeness tests using preservation manifests
- offline verification tests using exported bundles alone
- tests proving manifests are not treated as substitutes for the underlying evidence
- tests proving artifact-scoped and incident-scoped bundle selection resolve deterministically from canonical evidence and rebuildable indexes
- tests proving offline verification can recompute verification conclusions from exported canonical evidence when required inputs are present

## Key Design Rule
The preservation manifest and bundle manifest exist so RuneCode preserves the right evidence now, not to create a lighter-weight second truth surface later.
