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
- Confirm one local RuneCode product instance is selected per authoritative repository root and that repo-scoped product identity does not collapse into socket path, runtime directory, or other host-local runtime details.
- Confirm pidfiles, lockfiles, runtime directories, socket files, and other bootstrap-local artifacts remain advisory mechanics only; broker handshake and broker-owned lifecycle posture remain authoritative.
- Confirm a dedicated broker-owned typed product lifecycle posture surface exists and that clients do not need to infer attach semantics from `Readiness.ready`, `VersionInfo`, or local IPC reachability.
- Confirm `Readiness` remains a subsystem-health summary, `VersionInfo` remains a build/bundle diagnostic surface, and `ProjectSubstratePostureGet` remains the canonical repository compatibility/remediation surface.
- Confirm healthy broker attach with blocked repository project-substrate posture lands in explicit diagnostics/remediation-only posture rather than hard-failing reconnect or allowing unsafe normal operation.
- Confirm bare `runecode` means `attach` and that `runecode attach`, `runecode start`, `runecode status`, `runecode stop`, and `runecode restart` behave as frozen by this change.
- Confirm `runecode status` is non-starting and reports broker-owned lifecycle posture when reachable, otherwise only the bootstrap-local fact that no live product instance is reachable.
- Confirm the normal user path no longer requires manual `runecode-broker serve-local` startup before TUI attach.
- Confirm `runecode-broker`, `runecode-launcher`, and other low-level binaries remain valid plumbing/admin/dev surfaces without remaining the canonical user lifecycle surface.
- Confirm session summary and attach UX keep session object lifecycle, projected session work posture, and client attachment state as distinct concepts.
- Confirm execution-specific reconnect/resume policy for project-substrate drift remains intentionally deferred to `CHG-2026-048-6b7a-session-execution-orchestration-v0` rather than being guessed heuristically in this change.

## Close Gate
Use the repository's standard verification flow before closing this change.
