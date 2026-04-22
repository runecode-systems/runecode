# Tasks

## Local Bootstrap and Supervision

- [ ] Define a normal local product bootstrap flow for starting and supervising the trusted local control-plane services.
- [ ] Keep supervision and service orchestration local-only rather than exposing a second public lifecycle API.
- [ ] Preserve topology-neutral client contracts so later platform-specific service managers remain additive.
- [ ] Keep repository project-substrate discovery and compatibility evaluation read-only during start, attach, reconnect, and status flows.
- [ ] Forbid bootstrap from silently initializing, upgrading, or rewriting repository project substrate.
- [ ] Freeze one local RuneCode product instance per authoritative repository root rather than a single per-user global broker identity.
- [ ] Define a trusted private bootstrap/resolver layer that resolves authoritative repo root, derives repo-scoped local runtime mechanics, safely detects stale local runtime artifacts, ensures a live broker exists, and validates broker identity before attach.
- [ ] Keep pidfiles, lockfiles, runtime directories, socket files, and launcher-private supervision state advisory-only rather than authoritative lifecycle truth.
- [ ] Keep socket paths, runtime directories, state-root paths, and other host-local runtime details out of boundary-visible product identity.
- [ ] Keep the public lifecycle model targeted at the logical RuneCode product instance for a repo rather than the exact trusted process graph used by the current platform.

## Persistent Session Lifecycle

- [ ] Persist enough canonical session metadata to reconnect safely after TUI close or local process restart.
- [ ] Keep sessions and linked runs durable beyond the life of any one UI client.
- [ ] Define safe attach, detach, reconnect, and restart behavior in terms of broker-owned truth.
- [ ] Keep client attachment state, workbench-local recents/pins, and local layout state non-authoritative for session lifecycle and session identity.
- [ ] Separate session object lifecycle from projected session work posture and from client presence state.
- [ ] Extend session summary/read-model planning with a distinct broker-projected work-posture surface rather than overloading existing session object status.
- [ ] Freeze reconnect-at-product-layer semantics so blocked project-substrate posture still allows inspection and remediation attach, while execution-specific resume policy remains owned by `CHG-2026-048-6b7a-session-execution-orchestration-v0`.

## Broker-Projected Posture

- [ ] Project blocked, degraded, and ready posture through broker-visible status rather than client-local guesswork.
- [ ] Keep readiness and version surfaces sufficient for attachable TUI and CLI clients while reserving canonical project-substrate posture for the dedicated broker-owned typed surface established by `CHG-2026-046-a91d-runecontext-verified-project-substrate-compatibility-lifecycle-v0`.
- [ ] Keep daemon-private supervision health non-authoritative for operator UX.
- [ ] Distinguish local service health from repository project-substrate posture so healthy services with blocked repo substrate still surface diagnostics/remediation-only behavior.
- [ ] Add a dedicated broker-owned typed product lifecycle posture surface instead of relying on `Readiness.ready`, `VersionInfo`, local IPC reachability, or bootstrap-local heuristics as the attach contract.
- [ ] Ensure the product lifecycle posture surface projects stable product instance identity, lifecycle generation or equivalent restart identity, attach mode, normalized lifecycle posture, and stable degraded/blocked reason codes.
- [ ] Freeze attachability and normal-operation permission as distinct concepts so healthy broker attach can coexist with diagnostics/remediation-only posture when repository project-substrate is blocked.
- [ ] Keep `Readiness` as subsystem-health summary, `VersionInfo` as build/bundle diagnostics, and `ProjectSubstratePostureGet` as canonical repository compatibility/remediation truth rather than collapsing them into one catch-all status contract.

## TUI and CLI UX

- [ ] Support attach/detach/reconnect flows in the TUI without making the TUI authoritative for lifecycle truth.
- [ ] Provide equivalent CLI entry and recovery flows as thin adapters over the same broker-owned model.
- [ ] Route blocked repository substrate states to broker-owned diagnostics and remediation flows rather than implicit bootstrap repair.
- [ ] Introduce a canonical top-level `runecode` command and freeze bare `runecode` as `attach`.
- [ ] Add explicit top-level `runecode attach`, `runecode start`, `runecode status`, `runecode stop`, and `runecode restart` flows over the shared repo-scoped product model.
- [ ] Keep `runecode status` non-starting; if no live broker is reachable, it may report only the bootstrap-local fact that no live product instance is reachable rather than silently starting one.
- [ ] Keep `runecode-broker`, `runecode-launcher`, and other low-level binaries available as plumbing/admin/dev entrypoints without letting them remain the canonical user-facing lifecycle surface.
- [ ] Remove the normal-user requirement to manually start `runecode-broker serve-local` before opening the TUI.

## Acceptance Criteria

- [ ] RuneCode can start as one coherent local product lifecycle and later reconnect to active sessions and runs.
- [ ] Sessions and runs continue safely when the TUI is closed.
- [ ] Broker remains the only public control-plane authority surface.
- [ ] Client-visible lifecycle readiness and project-substrate blocked-state posture are broker-projected, topology-neutral, and do not rely on client-local heuristics.
- [ ] The canonical user-facing product command is `runecode`, and normal attach/start/stop/restart/status flows no longer require manual `runecode-broker serve-local` sequencing.
- [ ] Repo-scoped product instance identity is derived from authoritative repository root rather than from socket path, runtime directory, or other host-local runtime details.
- [ ] Healthy broker attach with blocked repository project-substrate posture lands in explicit diagnostics/remediation-only behavior rather than hard-failing reconnect or continuing normal managed operation.
- [ ] Session summaries and attach UX preserve the distinction between session object lifecycle, projected session work posture, and client attachment state.
