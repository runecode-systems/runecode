# Verification

## Planned Checks
- `just test`
- `go test ./internal/protocolschema`
- `cd runner && node --test scripts/protocol-fixtures.test.js`
- `cd runner && npm run boundary-check`

## Verification Notes
- Confirm the change distinguishes the audit plane from the verification plane and gives each a stable meaning.
- Confirm the design keeps canonical evidence authoritative and keeps derived indexes, search surfaces, dashboards, and watch views rebuildable.
- Confirm the foundation optimizes for inspectable evidence, deterministic verification, portable bundles, strong provenance, and explicit degraded posture.
- Confirm zero-knowledge proofs are described as an optional future privacy layer rather than the `v0` foundation.
- Confirm the change captures the full audience set: users, approvers, companies, auditors, incident response, external relying parties, fleet operators, privacy teams, and managed-service operators.
- Confirm the design includes signed receipts, append-only seal chains, runtime evidence, external anchoring, and verifier identity.
- Confirm the object-model split assigns index and record inclusion to `CHG-2026-056-8c75-audit-evidence-index-record-inclusion-v0`, preservation and bundle export to `CHG-2026-055-546a-verification-evidence-preservation-bundle-export-v0`, and coverage expansion to `CHG-2026-058-04e9-verification-coverage-expansion-v0`.
- Confirm the change makes degraded posture, denials, deferrals, overrides, and missing-evidence findings explicit rather than optional.
- Confirm the design preserves one architecture across constrained and scaled deployments and does not introduce a second truth surface or second authorization engine.
- Confirm proof-specific CLI, protocol, API, and dependency work is explicitly excluded from the foundation.
- Confirm the design requires canonical receipt families for material authority, approval, boundary, publication, override, and summary evidence rather than relying on derived summaries alone.
- Confirm the design requires preservation of enough evidence identity for cross-machine export, restore, and future federation-safe workflows without relying on machine-local mutable state as sole history.
- Confirm offline verification is described as recomputable from exported canonical evidence when required inputs are present, not only as archive integrity checking.
- Confirm Phase 6 explicitly includes invariant/model-check hardening and append-friendly performance work without changing trust semantics.
- Confirm the foundation explicitly separates project or repository identity, repo-scoped product-instance identity, persistent ledger identity, and project-substrate snapshot identity.
- Confirm persistent ledger identity is treated as a required seam for continuity across export, restore, migration, and reconcile workflows.
- Confirm bundle and snapshot semantics distinguish directly included canonical objects from transitive digest-reference dependencies.
- Confirm prepared-record evidence seams are hardened for downstream publication durability barriers and crash reconcile without implementing federation execution here.
- Confirm bundle manifests, preservation snapshots, and storage namespace layout are not treated as federation authority primitives.

## Close Gate
Use the repository's standard verification flow before closing this change.
