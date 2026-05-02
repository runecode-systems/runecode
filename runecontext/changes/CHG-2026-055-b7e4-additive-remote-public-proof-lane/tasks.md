# Tasks

## Define The Remote Lane Contract

- [ ] Define the operator-private remote proof lane as additive derived evidence over the canonical local audit and proof-binding substrate.
- [ ] Keep the local proof core authoritative and the remote lane non-authoritative for local trust decisions.
- [ ] Define the shared binding rules across local, remote, and future public lanes.

## Define Local Persistence Guarantees

- [ ] Define the non-optional local retention rule for machines that do not enable the remote or public lane yet.
- [ ] Define the minimum proof-relevant evidence classes that must remain locally preserved or exportable for future backfill.
- [ ] Define how retained evidence remains reconstructible without ambient source-machine context months or years later.
- [ ] Define storage-budget, retention, and export expectations strongly enough that constrained devices can preserve future-backfill prerequisites without accidental evidence loss.

## Define Export-Bundle Contract

- [ ] Define the export-bundle format for future proof backfill.
- [ ] Define the required bundle authenticity and manifest material.
- [ ] Define bundle provenance, evidence coverage, and export-identity rules.

## Define Remote Ingest And Backfill

- [ ] Define remote ingest verification rules.
- [ ] Define proof-work queue reconstruction from exported bundles.
- [ ] Define how backfilled proofs are published back as additive derived evidence.
- [ ] Define disagreement posture when local and remote proof results differ.

## Define Cross-Machine Merge Rules

- [ ] Define cross-machine stream identity and merge keys for preserved evidence.
- [ ] Define how the remote lane treats concurrent project history from more than one authoritative RuneCode stream.

## Define Future Public-Assurance Posture

- [ ] Define how public publication reuses the same binding substrate as the local and remote lanes.
- [ ] Define what public artifacts may be published without redefining local trust semantics.
- [ ] Define the information-asymmetry and selective-disclosure use cases for exported proofs.

## Evaluate Recursive Or Aggregate Proofs

- [ ] Define the conditions under which recursive or aggregate proofs become worthwhile for the remote or public lane.
- [ ] Keep recursion and aggregation out of local `v0` until the narrow local proof core is proven first.

## Evaluate Alternative Architecture

- [ ] Capture the additive dual-commitment proof-bridge option in full detail.
- [ ] Compare it against direct authoritative in-circuit membership and feature deferral.
- [ ] Require measured evidence before choosing the dual-commitment option.

## Acceptance Criteria

- [ ] The follow-on change clearly defines the operator-private remote proof lane without weakening or replacing RuneCode's authoritative local trust model.
- [ ] The follow-on change clearly defines what machines must retain locally even before any remote or public lane is enabled.
- [ ] The follow-on change clearly defines export-bundle, ingest, backfill, and disagreement posture strongly enough that another developer can implement it without additional product clarification.
- [ ] The follow-on change clearly defines the future public-assurance posture without creating a second public-only trust model.
- [ ] The follow-on change fully captures the additive dual-commitment architecture option and its decision rule.
