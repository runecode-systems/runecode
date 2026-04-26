# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `go test ./internal/protocolschema ./internal/policyengine ./internal/artifacts ./internal/brokerapi`
- `cd runner && npm run boundary-check`
- `cd runner && npm test`
- `just test`

## Verification Notes
- Confirm the migrated change preserves the legacy task breakdown and acceptance criteria in `tasks.md`.
- Confirm canonical references remain on RuneContext project, spec, and change paths, with no active workflow depending on legacy planning paths.
- Confirm the migrated text assumes RuneContext is canonical, RuneCode owns the user-facing UX, and verified-mode project state remains the expected operating posture.
- Confirm the change still matches its `v0.1.0-alpha.8` roadmap bucket and title after migration.
- Confirm dependency-fetch operations remain aligned with the shared gateway operation taxonomy rather than dependency-local outbound verbs.
- Confirm dependency-fetch audit fields remain aligned with the shared gateway audit evidence model.
- Confirm moderate-profile approval semantics require approval for dependency scope enablement or expansion rather than for every ordinary `fetch_dependency` cache fill.
- Confirm the design binds `payload_hash` to a canonical typed dependency-fetch request object rather than raw lockfile bytes or tool-private cache state.
- Confirm the design treats lockfile-bound batch requests and resolved dependency units as separate but linked cache layers, with resolved units as canonical CAS storage.
- Confirm dependency cache artifact classes distinguish dependency metadata/manifests from payload units and do not rely on generic `spec_text` classification.
- Confirm offline cached dependency use is modeled as broker-mediated internal artifact handoff rather than third-party egress.
- Confirm runner surfaces remain non-authoritative and never receive or persist registry credentials.
- Confirm the first end-to-end slice is explicitly public-registry-first and that this choice is justified as a delivery-slice optimization rather than a weakening of the long-term foundation.
- Confirm private-registry support remains additive through broker-owned lease-based auth plumbing and that the design avoids public-only shortcuts that would force a later cache redesign.
- Confirm the design bakes in stream-to-CAS persistence, bounded concurrency, and miss coalescing so the foundation scales from constrained local hardware to later larger deployments.

## Close Gate
Use the repository's standard verification flow before closing this change.
