## Summary
RuneCode can perform commit, push, and pull-request operations through a dedicated `git-gateway` role that consumes typed signed patch artifacts, requires exact final approval for remote mutations, binds policy to logical repository identity plus signed ref allowlists, and exposes broker-owned git setup and configuration surfaces consistently through both TUI and CLI thin clients.

## Problem
The repository already has shared gateway, approval, audit, artifact, lease, broker, and TUI foundations, but the high-risk git lane is still underspecified where later implementation and follow-on features are most likely to drift.

Without freezing those details now, later work is likely to accrete the wrong authority model:
- transport-URL or path-prefix policy instead of logical repository identity
- weak patch binding based on freeform intent or local workspace state instead of canonical typed hashes
- git-specific policy and approval exceptions instead of extending the shared gateway model
- repo policy hidden in mutable local settings instead of signed artifacts and manifests
- setup and auth flows that are CLI-only, TUI-only, or daemon-private rather than broker-owned typed control-plane behavior
- commit metadata and repo-specific commit requirements such as DCO inferred from ambient `git config` instead of typed policy and identity contracts

## Proposed Change
- Refine the shared gateway foundation so git remote mutation stays on the shared `gateway_egress` path and uses a third shared gateway operation class for remote state mutation rather than a git-only exception path.
- Treat `git_remote` as logical repository identity with exact-match semantics, while keeping transport URLs and provider API endpoints below that identity.
- Define canonical git ref allowlists using signed rules over full refs, with destructive ref mutations denied by default in `v1`.
- Keep patch artifacts in the existing `diffs` data class, but define a typed signed patch artifact family bound by base identity and expected result tree hash.
- Define provider-neutral typed git request families, including `GitRefUpdateRequest`, `GitPullRequestCreateRequest`, and shared `GitCommitIntent`, and bind remote mutation through canonical request hashes plus referenced patch artifact digests.
- Require exact final approval for push, tag, and pull-request remote mutation across approval profiles using a dedicated `git_remote_ops` trigger; stage sign-off remains a prerequisite and never a substitute for the final remote-mutation approval.
- Use `secretsd` as the only long-lived credential store and issue repo-scoped, operation-scoped, action-bound short-lived leases for git provider access.
- Extend shared gateway audit evidence for `git_egress`, including matched allowlist entry identity, destination identity, artifact digests, result tree identity, bytes, timing, and outcome.
- Make authoritative repo and ref policy artifact-managed only, including optional repo-specific commit rules such as DCO, while keeping authoritative user and account setup broker-managed and exposing both through thin TUI and CLI clients.

## Why Now
This work now lands in `v0.1.0-alpha.5`, after the audit, artifact, policy, broker, and scoped-credential foundations exist, so RuneCode can finish the secure local development loop before MVP without revisiting authority, approval, and trust-boundary semantics later. Freezing these contracts now also gives future provider, TUI, CLI, attestation, and richer git-lane work one durable foundation instead of several partially compatible interpretations.

## Assumptions
- `runecontext/changes/*` remains the canonical planning surface for this repository.
- RuneCode owns the end-user command and UX surfaces while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- The TUI remains a strict broker client implemented with the same Bubble Tea and Lip Gloss architecture and standards as the rest of the TUI rather than a privileged setup side channel.
- Operator CLI commands remain straightforward thin adapters over the same typed broker and control-plane semantics rather than a second authority surface.
- Repo-specific commit policy, including DCO when required by a target repository, is supported as artifact-managed policy rather than as a RuneCode-wide default.

## Out of Scope
- Runtime implementation detail during this planning update.
- Standalone commit-only as a separate first-class policy action in this change; commit remains an explicit typed substep inside git remote-mutation requests unless a later roadmap item proves a separate action family is needed.
- Force push, ref deletion, and other destructive or non-fast-forward ref mutations in `v1`.
- A second long-lived secrets store, environment-variable secret injection, or CLI-arg secret injection.
- Re-introducing legacy Agent OS planning paths as canonical references.

## Impact
This change becomes the durable source of truth for the git gateway's authority model: shared gateway semantics, exact approval binding, logical repo identity, typed patch and git request contracts, artifact-managed repo policy, broker-owned setup surfaces, and thin TUI and CLI clients. Capturing those details now removes the need for a second semantics rewrite later and gives future git-related features a stronger foundation to build on.
