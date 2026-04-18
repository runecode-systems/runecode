# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `go test ./internal/protocolschema`
- `just test`

## Verification Notes
- Confirm the migrated change preserves the legacy task breakdown and acceptance criteria in `tasks.md`.
- Confirm canonical references remain on RuneContext project, spec, and change paths, with no active workflow depending on legacy planning paths.
- Confirm the migrated text assumes RuneContext is canonical, RuneCode owns the user-facing UX, and verified-mode project state remains the expected operating posture.
- Confirm the change still matches its v0.2 roadmap bucket and title after migration.
- Confirm the auth lane now explicitly inherits the canonical `SecretLease` handoff model rather than introducing a second token-delivery path.
- Confirm auth-gateway and model-gateway separation remains explicit and that user-facing auth posture is broker-projected rather than daemon-private.
- Confirm auth destination identity is expressed through the shared logical destination model rather than raw transport-URL policy.
- Confirm auth-derived leases are scoped to destination identity, allowed operation set, and relevant action or policy bindings where supported.
- Confirm guided TUI setup and straightforward CLI setup remain thin clients of the same broker-owned typed auth and account-linking flows.
- Confirm any manual token-entry fallback remains limited to trusted interactive broker-mediated prompts rather than flags or environment variables.

## Close Gate
Use the repository's standard verification flow before closing this change.
