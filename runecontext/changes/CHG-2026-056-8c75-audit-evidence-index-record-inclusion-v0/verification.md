# Verification

## Planned Checks
- `just test`
- `go test ./internal/protocolschema`

## Verification Notes
- Confirm the index is described as derived and rebuildable rather than authoritative.
- Confirm the minimum lookup surface includes record digest to segment or frame mapping, segment to seal mapping, and seal-chain-index lookup.
- Confirm `AuditRecordInclusion` includes segment, seal, and enough inclusion material to be independently checked.
- Confirm mismatch between the index and canonical evidence triggers refresh or fail-closed behavior.
- Confirm the design keeps owner-only permissions and trusted local storage for sensitive derived evidence data.
- Confirm the feature does not introduce proof-specific CLI, API, or protocol surfaces.
- Confirm performance expectations require index-backed interactive lookup and canonical-evidence-only rebuild.
- Confirm tests include multi-segment previous-seal linkage, real computed Merkle roots, mismatch handling, and permission checks.

## Close Gate
Use the repository's standard verification flow before closing this change.
