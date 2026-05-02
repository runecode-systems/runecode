# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`

## Verification Notes
- Confirm this change remains additive to `CHG-2026-016-8cdb-zk-proof-v0-one-narrow-proof-verify` rather than expanding the local `v0` implementation scope.
- Confirm the change clearly defines the non-optional local evidence-retention rule for systems that do not enable the remote or public proof lane yet.
- Confirm the change clearly defines export-bundle authenticity, ingest, backfill, and disagreement posture.
- Confirm the change now defines concrete protocol surfaces for the export bundle, manifest, coverage ranges, authenticity envelope, and remote proof write-back artifacts.
- Confirm the change explicitly defines coverage-range semantics, complete versus partial export meaning, and anti-rollback posture for bundles from the same exporter.
- Confirm the change explicitly requires digest-addressed immutable runtime and attestation evidence in export bundles rather than only ambient live-store references.
- Confirm the change explicitly defines disclosure or redaction profile semantics for remote and public lanes.
- Confirm the change clearly defines cross-machine merge identity rules.
- Confirm the change clearly defines the future public-assurance lane as derivative and non-authoritative for local trust semantics.
- Confirm the change explicitly defines remote proof write-back validation and non-overwrite posture for authoritative local records.
- Confirm the change fully captures the additive dual-commitment architecture option, its benefits and risks, and the decision rule for revisiting it later.

## Close Gate
Use the repository's standard verification flow before closing this change.
