# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `go test ./internal/protocolschema`
- `just test`

## Verification Notes
- Confirm the change defines the signed-object contract as semantic payload plus detached attestation over RFC 8785 JCS bytes and does not rely on ambiguous omission of `signatures` fields.
- Confirm `ApprovalRequest` and `ApprovalDecision` are planned as signed objects and reserve an assurance-evidence hook for verified user assertions.
- Confirm authority scopes and verifier identities remain topology-neutral and can collapse physically in MVP without changing logical meaning.
- Confirm the trusted-state integrity scope explicitly covers canonical RuneCode trusted state and excludes runner-internal non-canonical state unless exported into canonical protocol objects.
- Confirm related changes `CHG-2026-006`, `CHG-2026-007`, `CHG-2026-008`, `CHG-2026-009`, `CHG-2026-014`, and `CHG-2026-033` stay aligned on signed approvals, assurance semantics, pending-approval behavior, and per-session isolate identity keys.
- Confirm canonical references remain on RuneContext project, spec, and change paths, with no active workflow depending on legacy planning paths.
- Confirm the change still matches its v0.1.0-alpha.2 roadmap bucket and title after refinement.

## Close Gate
Use the repository's standard verification flow before closing this change.
