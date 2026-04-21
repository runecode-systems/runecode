# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm bootstrap and supervision behavior do not create a second public authority surface beside the broker.
- Confirm sessions and linked runs can outlive a TUI client.
- Confirm attach, detach, and reconnect semantics depend on broker-owned truth rather than client-local lifecycle guesses.
- Confirm start, attach, reconnect, and status flows do not silently initialize, upgrade, or rewrite repository project substrate.
- Confirm blocked repository substrate posture routes clients to diagnostics/remediation flows rather than implicit bootstrap repair.
- Confirm readiness/version remain summary surfaces and project-substrate posture remains broker-owned on a dedicated typed surface.
- Confirm the change preserves topology-neutral contracts for later non-Linux service implementations.
- Confirm the roadmap and change text both place this feature in `v0.1.0-alpha.7`.

## Close Gate
Use the repository's standard verification flow before closing this change.
