# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `go test ./internal/protocolschema`
- `go test ./internal/policyengine`
- `go test ./internal/brokerapi`
- `go test ./cmd/runecode-tui`
- `just lint`
- `just test`
- `just ci`

## Verification Notes
- Confirm git remote mutation remains on the shared gateway contracts and does not introduce a git-only policy or approval authority.
- Confirm `git_remote` policy identity is logical and exact-match rather than raw URL or path-prefix policy.
- Confirm repo and ref allowlists, destructive-op posture, and repository-specific commit policy are authoritative only through signed artifacts or manifests rather than mutable settings.
- Confirm the typed patch artifact binds base identity and expected result tree hash, and confirm typed git request families carry the canonical request hashes used by policy, approval, audit, and runtime verification.
- Confirm push, tag, and pull-request remote mutation require exact final approval through `git_remote_ops`, and confirm stage sign-off alone cannot authorize remote mutation.
- Confirm standalone commit remains a typed substep rather than a separate first-class action in this change.
- Confirm force push, ref deletion, and non-fast-forward remote mutation remain denied by default in `v1`.
- Confirm long-lived provider auth material remains isolated to `secretsd`, with git leases bound to repository identity, allowed operation set, and action or policy hashes.
- Confirm audit records include matched allowlist entry identity, destination identity, referenced patch artifact digests, expected result tree hash, bytes, timing, outcome, and the bound action and policy hashes.
- Confirm provider auth bootstrap remains broker, auth-gateway, and `secretsd` owned, with no environment-variable or CLI-arg secret injection.
- Confirm TUI guided setup uses the normal Bubble Tea and Lip Gloss shell architecture plus broker APIs rather than direct daemon-private state, and confirm the CLI is a thin adapter over the same typed flows.
- Confirm local convenience state remains non-authoritative and that authoritative policy updates still flow through reviewed artifact or manifest paths.
- Confirm canonical references remain on RuneContext project, spec, change, and decision paths, with no active workflow depending on legacy planning paths.
- Confirm the change still matches its `v0.1.0-alpha.5` roadmap bucket and title after refinement.

## Close Gate
Use the repository's standard verification flow before closing this change, with `just ci` as the final gate.
