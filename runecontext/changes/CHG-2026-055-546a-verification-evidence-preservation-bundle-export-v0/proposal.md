## Summary
Deliver the preservation and export half of RuneCode's verification plane: evidence-preservation snapshots, verifier-friendly evidence-bundle manifests, streaming bundle export, selective-disclosure profiles, and offline verification support.

This feature ensures RuneCode preserves the right canonical evidence now instead of discovering later that critical evidence was lost, unexportable, or tied only to mutable local state.

## Problem
Canonical evidence is only useful if it can survive beyond one machine, one local state root, or one internal database view.

RuneCode needs an explicit preservation and export lane because:

- future verification should not depend on ambient mutable state
- retention and backfill planning require a durable list of required evidence identities
- external relying parties need portable provenance without access to internal RuneCode databases
- incident-response and auditor workflows need scoped export rather than ad hoc scraping
- privacy-aware disclosure requires explicit export profiles rather than one all-or-nothing bundle shape

Without this feature, the project risks preserving some evidence strongly but losing the ability to package, stream, retain, restore, or independently verify it later.

## Proposed Change
- Add `AuditEvidenceSnapshot` as a preservation manifest for the minimum set of canonical evidence identities that must be preserved or exported for future verification and backfill.
- Add `AuditEvidenceBundleManifest` as the portable description of an evidence bundle.
- Support streaming-friendly bundle export so large evidence sets do not require full in-memory assembly.
- Support explicit export profiles and selective-disclosure declarations.
- Keep portable bundles independently verifiable outside RuneCode's UI and database.
- Preserve enough identity material for export, restore, retention checks, and future cross-machine work.
- Keep bundle manifests signed when bundles are intended for external sharing.

## Why Now
This feature is part of the foundation because preservation mistakes are expensive to fix later.

If RuneCode does not record preservation requirements up front, later work can easily discover that:

- runtime evidence was retained only by convenience keys instead of digest identity
- anchor sidecars or verifier identity were omitted from export
- bundles cannot be verified offline without access to mutable local state
- privacy-sensitive payloads were overshared because export profiles were never defined

This lane also provides the portable evidence substrate needed by users, auditors, and external relying parties, not just internal operator tooling.

## Assumptions
- Canonical evidence remains the source of truth; snapshots and manifests are preservation and export helpers, not substitutes for the evidence itself.
- Bundle verification should work outside the originating machine.
- Streaming export is required because evidence sets can grow beyond comfortable in-memory assembly.
- Selective disclosure is a policy concern and must be explicit in the bundle metadata.
- Future cross-machine workflows should reuse exportable canonical evidence rather than machine-local mutable state.

## Out of Scope
- Treating bundle manifests as the only retained evidence.
- Requiring bundle verification to trust RuneCode's UI or internal database.
- Defaulting to raw prompt, code, provider payload, or secret export when digests and typed metadata are sufficient.
- Solving full cross-machine federation in this foundation lane.

## Impact
This feature gives RuneCode a durable preservation and export contract.

If completed, RuneCode will be able to:

- enumerate the evidence identities that must survive retention, export, and restore
- produce verifier-friendly bundles for runs, artifacts, incidents, operators, auditors, and external relying parties
- stream large bundle exports safely
- declare what was disclosed or redacted in a portable way
- verify exported bundles independently without mutable ambient state
