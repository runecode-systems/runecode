# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `go test ./internal/protocolschema`
- `cd runner && node --test scripts/protocol-fixtures.test.js`
- `just test`
- `just ci`

## Verification Notes
- Confirm the change explicitly freezes the shared workflow-kernel semantics rather than relying on feature-local approximations.
- Confirm the change explicitly owns the protocol and trusted-runtime refinements needed to implement the model instead of deferring them to a separate follow-on change.
- Confirm the design distinguishes approval decision acceptance from approval consumption, even if some broker operations apply both in one atomic transaction.
- Confirm stage sign-off now depends on a canonical `StageSummary` contract and hash, leaves `RunStageSummary` as a derived read model, and treats `summary_revision` as monotonic metadata rather than the trust root by itself.
- Confirm the effective-policy-context hashing contract behind `manifest_hash` is explicit enough that policy, approval, audit, and the formal model cannot drift.
- Confirm runner-supplied gate evidence is advisory and canonical gate-evidence linkage is broker-owned.
- Confirm gate-override and canonical gate-evidence contracts bind exact gate, attempt, failed-result, and policy-context identities through schema rules or explicit trusted validation.
- Confirm public run lifecycle stays on the shared broker vocabulary and partial blocking remains coordination and detail state.
- Confirm the `v0` audit-transition-obligation matrix is explicit and bounded rather than attempting the full audit ledger in the first formal model.
- Confirm protocol bundle manifest and any touched schema families or fixtures are updated consistently.
- Confirm TLC is the authoritative CI engine for `v0`, with spec structure kept finite-state and extensible for later engines.
- Confirm the planned invariants and traceability map back to concrete schemas, modules, and standards.
- Confirm the change still matches its `v0.1.0-alpha.5` roadmap bucket and title after refinement.

## Close Gate
Use the repository's standard verification flow before closing this change.
