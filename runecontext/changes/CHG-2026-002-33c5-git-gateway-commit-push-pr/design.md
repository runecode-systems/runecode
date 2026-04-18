# Design

## Overview
Define the dedicated Git gateway for commit, push, and pull-request operations as a high-risk outbound lane that stays on the shared gateway, approval, artifact, secrets, audit, and broker UX foundations rather than creating a git-only control plane.

This change now freezes the cross-feature contract decisions that later protocol, policy, broker, provider, CLI, and TUI work should inherit. Child implementation work may extend these contracts, but it should not reopen the authority model captured here.

## Key Decisions
- Git egress is treated as high risk and is isolated behind a gateway.
- Git remote mutation remains on the shared typed gateway foundation rather than introducing a git-only policy or approval path.
- Shared gateway semantics need a third operation class for remote state mutation, separate from request execution and scope change.
- `git_remote` should be treated as logical repository identity with exact-match semantics rather than raw transport URL or path-prefix policy.
- Outbound verification must bind canonical typed git requests plus referenced signed patch artifacts and expected result tree identity.
- Full typed git request objects are authoritative; any `GitRemoteMutationSummary` or equivalent summary/read-model object is derived-only UX data.
- No further git-lane implementation should use `GitRemoteMutationSummary` as an authority surface for policy, approval, audit, or execution.
- Push, tag, and pull-request remote mutation require exact final approval across approval profiles through a dedicated `git_remote_ops` trigger, and stage sign-off never substitutes for that final approval.
- Commit semantics are explicit through typed `GitCommitIntent`, but standalone commit remains a typed substep rather than a separate first-class action in this change.
- Repo-specific commit policy such as DCO is supported as artifact-managed policy, not as a RuneCode-wide default or ambient `git config` side effect.
- Git setup and configuration authority is broker-owned; TUI and CLI are thin adapters over the same typed setup and configuration flows.
- Broker request contract follows typed request-union operations (`prepare`, `get`, `execute`) with friendly CLI/TUI interaction layered above it.
- Trusted orchestration is implemented in Go, using native git for repo-local mutation and ref push, with Go provider adapters for provider-specific APIs.
- The first provider adapter is GitHub-only while typed request contracts remain provider-neutral from day one.

## Cross-Feature Foundation Decisions

### Shared Gateway Operation Model

- Keep git remote mutation on the shared `gateway_egress` action family so policy, approval, audit, and broker read models reuse one gateway path.
- Extend the shared gateway operation taxonomy or trait model to distinguish three classes:
  - request execution
  - scope change
  - remote state mutation
- Git ref update and pull-request creation belong to the remote-state-mutation class.
- Remote-state-mutation operations must bind `payload_hash` and `audit_context`.
- `quota_context` should remain operation-specific and optional rather than a mandatory requirement inherited from model or dependency lanes.

### Canonical Repository Identity

- `git_remote` must represent logical repository identity, not just a transport endpoint.
- Policy, approvals, leases, and audit should bind to the repository identity or descriptor digest rather than to a raw URL string.
- Transport URLs, git remotes, and provider API endpoints remain derived execution details below that logical identity.
- Matching semantics for git repositories should be exact repository identity matching, not generic host plus path-prefix matching.
- The design should leave room for distinct `base_repo` and `head_repo` identities in pull-request flows even if `v1` commonly targets the same repository.

### Canonical Ref Policy

- Repo policy must use canonical git refs such as `refs/heads/main` and `refs/tags/v1.2.3`.
- Signed allowlist rules should support exact ref matches and constrained prefix-glob namespace rules, not regex.
- Ref-update rules, tag rules, pull-request base-ref rules, and pull-request head namespace rules should stay distinct rather than being flattened into one ambiguous branch string.
- Force pushes, ref deletions, and other destructive or non-fast-forward ref mutations are denied by default in `v1`.

### Signed Patch Artifacts And Typed Git Requests

- Keep git patch artifacts in the existing `diffs` data class rather than creating a git-only data class.
- Define a typed signed patch artifact family that captures at least:
  - base commit hash
  - base tree hash
  - canonical patch payload
  - touched path inventory
  - expected result tree hash
- The expected result tree hash is the primary outbound verification truth because it is more reusable and less incidental than a full commit hash.
- Define provider-neutral typed request families whose canonical digest becomes the bound `payload_hash` for git remote mutation.
- The minimum typed request families are:
  - `GitRefUpdateRequest`
  - `GitPullRequestCreateRequest`
  - shared nested `GitCommitIntent`
- Typed request objects are the sole authority source for policy evaluation, approval binding, audit linkage, and execution.
- Any `GitRemoteMutationSummary` (or similar display/read projection) must be derived from a typed request object and must never be treated as authoritative input.
- Migration expectation: existing summary-authority paths are replaced by typed-request authority, while retaining summaries only for operator UX readability.
- `GitCommitIntent` should carry structured commit message, trailers, author identity, committer identity, and signoff identity.
- `GitRefUpdateRequest` should bind one repository identity, one target ref, expected old state, referenced patch artifact digests, shared commit intent, and expected result tree hash.
- `GitPullRequestCreateRequest` should bind base repo and ref, head repo and ref, title and body metadata, and the head commit or tree identity created from the approved patch flow.
- Tag creation or update, if included, should use the same typed ref-update contract rather than a separate ad hoc payload family.

### Approval Model

- Git remote mutation is an exact-action approval boundary regardless of active approval profile.
- Add a dedicated `git_remote_ops` approval trigger code rather than overloading the broader gateway scope-change trigger.
- The required approval payload for git remote mutation should bind at least:
  - repository identity
  - target refs
  - referenced patch artifact digests
  - expected result tree hash
  - commit or pull-request metadata summary
- Baseline human assurance for remote mutation should be at least `reauthenticated`.
- Stage sign-off may enable the lane, but it must never authorize the final push, tag, or pull-request creation by itself.

### Commit Identity And Repository Commit Policy

- Commit identity must be explicit typed input, not ambient process or machine `git config`.
- Repo-specific commit policy such as DCO must be modeled as repository policy rather than as a RuneCode-wide invariant.
- When a target repository requires DCO, the gateway should render the `Signed-off-by:` trailer deterministically from structured signoff identity rather than trusting arbitrary freeform message text.
- The same typed trailer model should support other future repository-specific trailer rules without rewriting the core git request contracts.
- Cryptographic commit signing is a later additive feature and is not part of the `v1` foundation.

### Secrets And Lease Binding

- `secretsd` remains the only long-lived credential store.
- Git provider access uses short-lived repo-scoped, operation-scoped, action-bound leases or derived tokens.
- Git leases should bind to canonical repository identity, allowed operation set, `action_request_hash`, and the policy context hash.
- If a provider cannot enforce the full narrow scope at issuance time, the trusted runtime must still enforce the narrower approved scope and fail closed on mismatch.
- No environment-variable secret injection or CLI-arg secret injection is allowed.

### Audit Evidence

- Extend the shared gateway audit model with a dedicated `git_egress` event family.
- The allowlist model should expose stable matched rule identity so audit can attribute not only the allowlist digest but also the specific matched entry.
- Git audit events should include the shared gateway network fields plus git proof material:
  - matched allowlist digest and entry identity
  - destination descriptor identity
  - operation
  - referenced patch artifact digests
  - expected result tree hash
  - observed ref-update or pull-request outcome
  - bytes
  - timing
  - outcome
  - `action_request_hash`
  - `policy_decision_hash`

### Setup And Configuration Surfaces

- Authoritative repository policy configuration is artifact-managed only.
- Repo and ref allowlists, destructive-op posture, and repo-specific commit requirements such as DCO must become authoritative only through signed artifacts and manifests.
- TUI and CLI may inspect those policies and may offer explicit reviewed authoring flows that materialize proposed artifact or manifest changes, but they must not directly mutate policy truth through ad hoc settings.
- Authoritative user and account configuration such as linked provider account, commit identity profiles, and auth posture should be broker-managed through typed control-plane APIs.
- Broker APIs for git remote mutation should expose typed request-union verbs (`prepare`, `get`, `execute`) so orchestration and UX clients consume one canonical request lifecycle.
- The TUI should offer guided interactive git setup, but it must do so as a normal Bubble Tea and Lip Gloss route or pane flow under the same root shell plus child-model architecture and broker API rules as the rest of the TUI.
- The CLI should remain straightforward and support headless, automation, recovery, and constrained environments as a thin adapter over the same broker-owned flows.
- Provider auth bootstrap should align with the auth-gateway lane:
  - browser-oriented flow for normal local desktop use where supported
  - device-code style or equivalent flow for headless or constrained environments where supported
- If a provider ever requires manual token entry as a fallback, that should only happen through trusted interactive prompts surfaced by the TUI or CLI over typed broker APIs rather than through environment variables or command-line flags.
- Local convenience state such as recent repositories, preferred setup view, or last-inspected provider posture is non-authoritative client state.

### Explicit `v1` Boundaries

- Standalone commit-only remains out of scope for this change as a separate first-class action family.
- Git remote mutation must fail closed on remote drift; `v1` does not silently merge, rebase, or force push to make the approved patch fit.
- Sparse or partial checkout should be the default execution posture for the gateway runtime.
- Provider-specific pull-request APIs live below the provider-neutral typed pull-request request contract.
- GitHub is the first provider adapter in scope for runtime delivery; additional providers are additive follow-on work beneath the same provider-neutral request contracts.

## Main Workstreams
- Shared Gateway Contract Refinement
- Git Repository Identity + Allowlist Model
- Typed Patch Artifact + Git Request Contracts
- Exact-Approval + Commit Policy Integration
- Secretsd-Backed Credential Scope
- Runtime Outbound Verification + Trusted Go Orchestration + Provider Adapters
- Broker Setup and Configuration Surfaces for TUI and CLI

## Sequencing Notes

- Freeze the shared gateway operation traits before adding git request verbs so git does not become a one-off exception path.
- Freeze logical repository identity and canonical ref policy before provider adapters or secrets scope work so push and pull-request flows share the same policy identity.
- Freeze the typed patch artifact and typed git request families before approval, audit, and runtime verification work so all three bind the same hashes.
- Freeze and migrate away from summary-as-authority (`GitRemoteMutationSummary`) before landing additional git-lane behavior so all later work inherits typed-request authority.
- Freeze exact git remote approval semantics and repository commit-policy rendering before TUI or CLI setup UX lands so UX flows do not invent their own authority model.
- Freeze repo-scoped lease binding and shared `git_egress` audit evidence before provider-specific runtime integration so later features build on one durable proof model.

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
