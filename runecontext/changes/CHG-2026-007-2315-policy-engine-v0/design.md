# Design

## Overview
Implement the core policy evaluator that enforces manifests, role invariants, and explicit approvals.

This change also fixes the canonical policy foundation that later broker, runner, gateway, TUI, approval-profile, and formal-spec work should build on. The goal is not only to ship MVP policy behavior, but to freeze the shared policy semantics early enough that later features do not invent parallel action, approval, gateway, or reason-code models.

## Key Decisions
- Deny-by-default everywhere; allow only via signed manifest.
- No automatic fallback to containers; container mode is explicit opt-in.
- MVP policy language is declarative and schema-validated; policy evaluation does not execute general-purpose code.
- Core security invariants are non-negotiable; any approval policy or UX setting may only tighten policy, never loosen it.
- Effective policy is compiled deterministically from fixed invariants, the active `RoleManifest`, the active run-scoped `CapabilityManifest`, the active stage-scoped `CapabilityManifest` when present, and content-addressed allowlist artifacts referenced by that active manifest set.
- Policy composition precedence is fixed: invariants -> role manifest max envelope -> run capability manifest -> stage capability manifest -> referenced allowlist inputs. Anything absent from the active signed context is denied fail-closed.
- `manifest_hash` in policy and approval flows is the hash of a canonical effective policy context object rather than one raw source manifest. Contributing source manifest and allowlist hashes remain explicit in `policy_input_hashes`.
- Policy evaluates one canonical typed `ActionRequest` contract. MVP uses one `ActionRequest` object family, a closed `action_kind` registry, and typed payload families per action kind; unknown action kinds or payload schema IDs fail closed.
- Approval scope splits into exact-action approval and stage sign-off. Exact-action approvals bind to one canonical `ActionRequest` hash; stage sign-off binds to one canonical stage summary hash and must be superseded when that summary changes.
- `ApprovalBoundScope` is operator-facing metadata derived from the signed approval contract, not the trust root by itself.
- Role identity uses distinct layers: `actor_kind`, `role_family`, and concrete `role_kind`. `role_kind` names least-privilege roles such as `workspace-read`, `workspace-edit`, `workspace-test`, `model-gateway`, `auth-gateway`, `git-gateway`, `web-research`, and `dependency-fetch` rather than mixing family and subtype terms.
- Network egress is a hard boundary: workspace roles are offline; public egress is only via explicit gateway roles, and non-gateway network egress is not approvable.
- Gateway allowlists use a two-layer typed model: `DestinationDescriptor` identifies the canonical external target and `GatewayScopeRule` (or equivalent allowlist entry) defines permitted operations, allowed egress data classes, and redirect posture. Raw URL-based ad hoc decisions are prohibited.
- Gateway hardening invariants remain invariant-owned: TLS requirements, SSRF/private-range blocking, DNS rebinding protections, and redirect-only-to-allowlisted-destination rules must not be delegated to UI or ad hoc runtime logic.
- MVP uses checkpoint-style approvals (stage sign-off and explicit posture changes) instead of per-action nags.
- MVP supports a single approval profile (`moderate`); later profile expansion lives in `runecontext/changes/CHG-2026-014-0c5d-approval-profiles-strict-permissive/`.
- Approval requests and decisions are typed, hash-bound to immutable inputs, signed, and time-bounded (TTL/expiry); stale approvals are invalid.
- The user-involvement slider/profile maps ordinary action categories to approval frequency, batching, TTL, and minimum assurance, but it must not lower the fixed assurance floor for a small set of high-blast-radius operations.
- The fixed hard floor uses a separate closed taxonomy from `approval_trigger_code`. MVP starts with `trust_root_change`, `security_posture_weakening`, `authoritative_state_reconciliation`, and `deployment_bootstrap_authority_change`.
- One action may match multiple hard-floor classes; policy uses the strongest resulting assurance floor.
- Stage sign-off remains profile-controlled rather than automatically belonging to the fixed hard floor.
- Delivery channel is advisory only; local TUI, remote TUI, or messaging delivery must all converge on the same signed approval contract.
- `ApprovalDecision` is signed by the approval authority after verifying any required user cryptographic assertion rather than by trusting the delivery channel or a free-floating button click.
- Policy decisions and failures use shared typed protocol objects, but they are not the same thing: successful evaluation always yields a `PolicyDecision`, including `deny`, while failed evaluation or failed approval consumption yields the shared protocol `Error` envelope.
- Reason-code ownership stays split: `policy_reason_code` explains the policy outcome, `approval_trigger_code` explains why human approval is required, and `error.code` explains system-level failures.
- Decision details use one common `PolicyEvaluationDetails` family plus typed `required_approval` payload families keyed to approval trigger type.
- Every policy decision is a typed persisted object with a stable digest and signed audit binding. MVP does not introduce a separate policy-signing authority.
- MVP policy implementation lives in a trusted Go package (`internal/policyengine`) with one context-compile step and one deterministic evaluation entrypoint from `ActionRequest` to `PolicyDecision`. Component-local structural checks stay local, but authorization semantics come from the shared engine.

## Effective Policy Context
- Effective policy context is the canonical compiled input to evaluation and replay.
- The compiled context contains:
  - fixed invariants
  - active role manifest identity and allowed role envelope
  - active run capability manifest opt-ins
  - active stage capability manifest opt-ins when stage-scoped policy applies
  - referenced allowlist artifacts and their digest identities
  - approval profile and any fixed hard-floor mappings
- `manifest_hash` is the digest of this compiled context, not a shortcut for one source document.
- `policy_input_hashes` list the contributing role manifest, capability manifests, allowlists, and other immutable policy inputs that produced the compiled context.
- Decision precedence remains explicit deny -> require_human_approval -> allow.

## Action Model
- `ActionRequest` is the canonical policy input contract and the source of `action_request_hash`.
- MVP keeps one shared `ActionRequest` family with:
  - common scope and identity metadata
  - `action_kind`
  - `action_payload_schema_id`
  - typed `action_payload`
  - bound `relevant_artifact_hashes`
- Initial action kinds should cover at least:
  - workspace writes
  - executor runs
  - artifact reads
  - excerpt/artifact promotion
  - gateway egress
  - backend posture changes
  - gate overrides
  - stage summary sign-off
- `ActionRequest` is the canonical policy contract, but broker API request families do not need to collapse into one generic transport envelope.

## Role Taxonomy
- `actor_kind` identifies who is acting (`user`, `daemon`, `role_instance`, `local_client`, `external_runtime`).
- `role_family` identifies the invariant bucket (`workspace` or `gateway`).
- `role_kind` identifies the concrete least-privilege role.
- Family-level invariants stay simple:
  - workspace roles are offline
  - gateway roles do not receive ambient workspace read-write access
- Concrete role behavior stays role-kind specific so `model-gateway`, `auth-gateway`, `git-gateway`, `web-research`, and `dependency-fetch` do not collapse into one generic gateway permission bucket.

## Gateway Model
- Gateway policy uses typed `DestinationDescriptor` families for canonical external identity.
- MVP descriptor kinds should cover at least:
  - `model_endpoint`
  - `auth_provider`
  - `git_remote`
  - `web_origin`
  - `package_registry`
- Gateway allowlist entries (`GatewayScopeRule` or equivalent) define permitted operations, allowed egress data classes, redirect posture, and bounded response semantics where needed.
- Manifests bind allowlist artifacts by digest so runtime policy does not need to trust raw request URLs or hostnames as ad hoc authority.

## Approval Model
- Exact-action approvals bind one canonical `ActionRequest` hash plus the current effective policy context and relevant artifact hashes.
- Stage sign-off binds one canonical stage summary hash plus the current effective policy context and relevant artifact hashes.
- If the stage summary changes before approval is consumed, the previous sign-off request becomes stale and a new approval request supersedes it.
- Typed approval payloads remain specific to approval trigger type rather than one freeform details blob.
- MVP approval payload families should cover at least:
  - stage sign-off
  - reduced-assurance backend opt-in
  - gate override
  - gateway egress scope change
  - out-of-workspace write
  - secret lease access
  - system-modifying execution
  - excerpt promotion

## Executor and Dependency Semantics
- Command execution uses a closed `executor_class` taxonomy rather than loosely defined "shell" behavior.
- Ordinary workspace executors are workspace-scoped, offline, typed, and non-privileged.
- Raw shell or other system-modifying execution is never implicitly treated as an ordinary workspace executor.
- System-modifying execution includes host/global state changes, out-of-workspace writes, system package installs, service/network/kernel/container configuration, and persistent OS/user config changes.
- Dependency behavior is intentionally split:
  - network fetch/cache-fill is a `dependency-fetch` action
  - offline use of cached read-only dependencies inside the workspace is ordinary workspace execution

## Decision Outputs
- `PolicyDecision` remains the canonical successful evaluation output.
- `PolicyEvaluationDetails` should expose:
  - bound scope
  - matched manifest hashes
  - matched allowlist refs
  - matched capability IDs
  - enforced invariants
  - approval trigger codes
  - hard-floor classes
  - computed minimum assurance
- `required_approval` payloads should always include enough structured context for safe review, including scope, why approval is required, changes if approved, effect if denied or deferred, security-posture impact, blocked work, expiry, and related hashes/artifacts.
- One primary `policy_reason_code` is recorded per decision. Secondary contributing factors remain in typed details rather than in competing primary codes.
- Policy decision digests and primary reason codes should be directly reusable in audit records and broker/operator read models.

## Implementation Boundary
- `internal/policyengine` should own:
  - compiling effective policy context from manifests, allowlists, and invariant tables
  - deterministic evaluation from `ActionRequest` to `PolicyDecision`
  - stable reason-code selection and typed details emission
- Existing artifact-specific policy logic is a seed rule set, not the long-term home of general policy semantics.
- Artifact store, launcher, broker, and other trusted components may retain local integrity checks and structural validation, but allow/deny/approval-required semantics should route through the shared engine.

## Main Workstreams
- Effective Policy Context + Action Model
- Role Taxonomy + Gateway Model
- Invariants + Hard-Floor Taxonomy
- Approval Policy (MVP: Moderate)
- Backend Selection Rules
- Decision Outputs + Audit Binding
- Shared Policy Engine Package

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
