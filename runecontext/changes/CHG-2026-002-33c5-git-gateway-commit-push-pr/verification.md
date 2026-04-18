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
- Confirm full typed git request objects are the only authority inputs for policy, approval, audit, and execution.
- Confirm `GitRemoteMutationSummary` (or equivalent summary models) is derived-only UX/read-model data and is not consumed as authority by any git-lane path.
- Confirm migration away from summary-authority is complete for the git lane (no policy/approval/execution path depends on summary objects as source-of-truth).
- Confirm push, tag, and pull-request remote mutation require exact final approval through `git_remote_ops`, and confirm stage sign-off alone cannot authorize remote mutation.
- Confirm standalone commit remains a typed substep rather than a separate first-class action in this change.
- Confirm force push, ref deletion, and non-fast-forward remote mutation remain denied by default in `v1`.
- Confirm long-lived provider auth material remains isolated to `secretsd`, with git leases bound to repository identity, allowed operation set, and action or policy hashes.
- Confirm audit records include matched allowlist entry identity, destination identity, referenced patch artifact digests, expected result tree hash, bytes, timing, outcome, and the bound action and policy hashes.
- Confirm provider auth bootstrap remains broker, auth-gateway, and `secretsd` owned, with no environment-variable or CLI-arg secret injection.
- Confirm broker contract for git mutation is typed request-union `prepare/get/execute`, with CLI and TUI operating as thin interaction layers over the same APIs.
- Confirm trusted orchestration is implemented in Go, with native git handling repo-local mutation and ref push.
- Confirm provider-specific API calls are mediated through Go provider adapters, with GitHub as first in-scope adapter and provider-neutral request contracts unchanged.
- Confirm TUI guided setup uses the normal Bubble Tea and Lip Gloss shell architecture plus broker APIs rather than direct daemon-private state, and confirm the CLI is a thin adapter over the same typed flows.
- Confirm local convenience state remains non-authoritative and that authoritative policy updates still flow through reviewed artifact or manifest paths.
- Confirm canonical references remain on RuneContext project, spec, change, and decision paths, with no active workflow depending on legacy planning paths.
- Confirm the change still matches its `v0.1.0-alpha.5` roadmap bucket and title after refinement.

## Close Gate
Use the repository's standard verification flow before closing this change, with `just ci` as the final gate.
