## Summary
RuneCode can add a future operator-private remote proof lane and later public-assurance publication lane that consume exported canonical evidence and proof-binding sidecars without replacing the authoritative local trust model.

## Problem
`CHG-2026-016-8cdb-zk-proof-v0-one-narrow-proof-verify` now intentionally focuses on one local `v0` proof family. The larger additive remote and public proof story still needs canonical planning so RuneCode preserves the right data now and can expand later without redesigning local trust semantics.

## Proposed Change
- Define the additive operator-private remote proof lane as a follow-on change over the same canonical audit and proof-binding substrate established by `CHG-2026-016-8cdb-zk-proof-v0-one-narrow-proof-verify`.
- Define how canonical evidence, proof-binding sidecars, manifests, and authenticity material are persisted locally even on machines that do not have the remote or public lane enabled yet.
- Define the export-bundle, ingest, backfill, merge, and publication posture for the future lane so preserved local evidence can be replayed later without ambient source-machine context.
- Define the concrete export-bundle protocol, manifest, coverage-range semantics, anti-rollback rules, disclosure profile, and remote proof write-back validation strongly enough that the follow-on lane is implementable rather than only directional.
- Keep the remote lane additive and asynchronous rather than a replacement for local correctness.
- Keep future public-assurance publication on the same binding substrate rather than introducing a second public-only trust model.
- Capture the possible additive dual-commitment architecture switch as a deliberate future design option if direct authoritative-Merkle in-circuit membership proves too expensive.

## Why Now
RuneCode now has enough clarity on the local `v0` proof core to split the future remote and public story into its own canonical change.

Capturing that lane now prevents accidental loss of local evidence-retention requirements while also keeping `CHG-2026-016-8cdb-zk-proof-v0-one-narrow-proof-verify` focused on a shippable local first step.

## Assumptions
- `CHG-2026-016-8cdb-zk-proof-v0-one-narrow-proof-verify` remains the only change intended to implement the first local proof family end-to-end.
- The future remote lane must consume the same canonical proof-binding identities and logical statement families as the local lane.
- RuneCode's authoritative audit and policy boundaries remain local and trusted even when later additive remote proof services exist.

## Out of Scope
- Implementing the future remote or public lane in this planning step.
- Changing the local `v0` proof family scope or weakening its trust semantics.
- Replacing RuneCode's authoritative audit Merkle construction as part of this planning step.

## Impact
Provides a canonical home for the additive remote and public proof roadmap, including local persistence guarantees, export and ingest design, backfill posture, public publication posture, and the dual-commitment architecture alternative.
