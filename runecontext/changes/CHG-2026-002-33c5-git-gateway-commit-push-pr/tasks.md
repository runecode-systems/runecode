# Tasks

## Shared Gateway Contract Refinement

- [ ] Extend the shared gateway operation taxonomy or trait model to distinguish request execution, scope change, and remote-state-mutation semantics.
- [ ] Keep git remote mutation on the shared `gateway_egress` path rather than introducing a git-only action family or policy lane.
- [ ] Require `payload_hash` and `audit_context` for remote-state-mutation gateway operations while keeping `quota_context` optional and operation-specific.
- [ ] Add stable git audit-event typing and matched allowlist entry identity support needed for later audit attribution.

Parallelization: can be designed in parallel with policy-engine gateway work, but it should freeze before git-specific request verbs or runtime adapters land.

## Git Repository Identity + Allowlist Model

- [ ] Model `git_remote` as logical repository identity with exact-match semantics rather than raw URL or path-prefix policy decisions.
- [ ] Keep transport URLs and provider API endpoints below the logical repository identity so push and pull-request flows share one policy identity.
- [ ] Model git destinations through the shared typed `DestinationDescriptor` and allowlist-entry pattern rather than a git-only destination shape.
- [ ] Define signed git ref policy using canonical refs, exact matches, and constrained prefix-glob namespace rules rather than branch-name heuristics or regex.
- [ ] Keep ref-update rules, tag rules, pull-request base-ref rules, and pull-request head namespace constraints distinct.
- [ ] Default deny force pushes, ref deletions, and other destructive or non-fast-forward ref mutations in `v1`.
- [ ] Keep repo and ref allowlists, destructive-op posture, and repo-specific commit requirements such as DCO artifact-managed rather than mutable local settings.

Parallelization: can be designed in parallel with protocol schema and policy-engine destination work; exact repository identity semantics must freeze before secrets scope and provider adapters.

## Typed Patch Artifact + Git Request Contracts

- [ ] Keep git patch artifacts in the `diffs` data class while defining a typed signed patch artifact family.
- [ ] Bind each patch artifact to base commit hash, base tree hash, canonical patch payload, touched path inventory, and expected result tree hash.
- [ ] Define provider-neutral typed git request families for `GitRefUpdateRequest` and `GitPullRequestCreateRequest`.
- [ ] Define shared `GitCommitIntent` carrying structured commit message, trailers, author identity, committer identity, and signoff identity.
- [ ] Keep standalone commit as a typed substep inside git remote-mutation requests rather than a separate first-class action in this change.
- [ ] Bind git remote-mutation `payload_hash` to canonical typed git request identity rather than to local workspace state.
- [ ] Ensure tag creation or update, if supported in `v1`, uses the same typed ref-update contract.

Parallelization: can be implemented in parallel with artifact-store work once patch artifact shape and git request hashes are frozen.

## Exact Approval + Commit Policy Integration

- [ ] Define a dedicated `git_remote_ops` approval trigger for push, tag, and pull-request remote mutation.
- [ ] Require exact final approval for git remote mutation across approval profiles, with baseline assurance at least `reauthenticated`.
- [ ] Ensure stage sign-off can enable the lane but cannot substitute for final remote-mutation approval.
- [ ] Bind required-approval payloads to repository identity, target refs, referenced patch artifact digests, expected result tree hash, and commit or pull-request metadata summary.
- [ ] Support repo-specific commit policy such as DCO as artifact-managed repository policy rather than as a RuneCode-wide invariant.
- [ ] Render `Signed-off-by:` and other repository-required trailers deterministically from structured identity when policy requires them.

Parallelization: can be implemented in parallel with approval-profile work once typed git request contracts and repository identity semantics are stable.

## Secretsd-Backed Credential Scope

- [ ] Keep `secretsd` as the only long-lived credential store for git provider access.
- [ ] Issue repo-scoped, operation-scoped, action-bound short-lived leases or derived tokens for git access.
- [ ] Bind git leases to canonical repository identity, allowed operation set, `action_request_hash`, and policy context hash.
- [ ] Add revocation support for active git-related leases.
- [ ] If a provider cannot enforce the full narrow scope at issuance time, enforce the narrower approved scope in trusted runtime and fail closed on mismatch.
- [ ] Forbid environment-variable secret injection and CLI-arg secret injection.

Parallelization: can be implemented in parallel with `secretsd` lease work once repository identity and operation taxonomy are frozen.

## Runtime Outbound Verification + Pull-Request Provider Adapters

- [ ] Consume the signed typed patch artifact in the git gateway runtime.
- [ ] Apply patches in a sparse or partial checkout by default.
- [ ] Verify expected old state before mutating the repository.
- [ ] Verify observed outbound result tree hash matches the approved typed request and signed patch artifact before completing remote mutation.
- [ ] Fail closed on remote drift rather than silently rebasing, merging, or force-pushing.
- [ ] Create pull requests through provider APIs beneath the provider-neutral typed pull-request contract.
- [ ] Attach run artifacts, gate results, and related evidence as structured metadata where the provider contract allows it.
- [ ] Audit remote git operations with standard gateway fields plus git proof fields such as matched allowlist entry identity, artifact digests, result tree hash, bytes, timing, and outcome.

Parallelization: provider-specific adapters can proceed in parallel once runtime verification, git request contracts, and audit evidence requirements are stable.

## Broker Setup And Configuration Surfaces For TUI And CLI

- [ ] Add broker-owned typed setup and configuration APIs for git provider account state, commit identity profiles, auth posture, and non-policy git control-plane state.
- [ ] Keep authoritative repository policy on artifact-managed surfaces only; TUI and CLI may inspect and prepare reviewed policy changes but must not directly mutate policy truth through ad hoc settings.
- [ ] Implement guided interactive git setup in the TUI using Bubble Tea and Lip Gloss under the existing root shell plus child-model architecture and broker API rules.
- [ ] Implement straightforward CLI thin adapters over the same broker-owned setup and configuration flows for headless, automation, recovery, and constrained environments.
- [ ] Align provider auth bootstrap with the auth-gateway lane, including browser-oriented login for normal use and device-code style or equivalent flows for headless environments where supported.
- [ ] If manual token entry is ever required as a provider fallback, keep it limited to trusted interactive prompts over typed broker APIs rather than flags or environment variables.
- [ ] Keep local convenience state such as recent repositories or preferred setup views non-authoritative.

Parallelization: TUI and CLI clients can be built in parallel once broker setup contracts are frozen; neither should define authority or policy semantics locally.

## Acceptance Criteria

- [ ] Git operations are impossible from workspace roles.
- [ ] Git remote mutation remains on the shared typed gateway path rather than a git-only policy or approval path.
- [ ] Repository identity is logical and exact-match, not a raw transport URL or path-prefix heuristic.
- [ ] Repo and ref policy, destructive-op posture, and repo-specific commit rules are authoritative only when expressed through signed artifacts or manifests.
- [ ] Push, tag, and pull-request remote mutation require exact final approval through `git_remote_ops`, and stage sign-off alone cannot authorize them.
- [ ] Outbound verification blocks pushes and pull requests whose observed repository result does not match the approved typed request, signed patch artifact, and expected result tree hash.
- [ ] Long-lived provider auth material remains isolated to `secretsd`, and git credentials are short-lived, repo-scoped, operation-scoped, and action-bound.
- [ ] TUI guided setup and CLI setup remain thin clients of broker-owned typed flows, with no client-local policy authority.
