# Tasks

## Role + Run/Stage Policy Model

- [ ] Define the canonical effective policy context object that combines fixed invariants, the active role manifest, the active run capability manifest, the active stage capability manifest when present, and allowlist artifacts referenced by that active manifest set.
- [ ] Freeze composition precedence:
  - invariants apply first and cannot be relaxed
  - the role manifest defines the maximum role envelope
  - the run capability manifest narrows/enables within the role envelope
  - the stage capability manifest narrows/enables within the run envelope
  - only allowlist artifacts referenced by the active manifest set participate in evaluation
  - any capability or allowlist not present in the active signed context is denied fail-closed
- [ ] Bind all decisions to the compiled effective policy context hash rather than one raw source manifest:
  - `manifest_hash` means the hash of the compiled effective policy context
  - `policy_input_hashes` carry the contributing source manifest and allowlist hashes
- [ ] Define the MVP policy language and evaluation semantics:
  - declarative, schema-validated policy documents (no embedded scripting)
  - explicit precedence rules (`deny > require_human_approval > allow`)
  - stable reason codes and structured decision details
  - use the shared protocol error taxonomy/envelope for denials and failures (see `runecontext/specs/protocol-schema-bundle-v0.md`)
- [ ] Define policy loading/tamper resistance:
  - policy inputs are content-addressed and bound to signed manifests
  - policy evaluation rejects inputs that do not validate against the schema version

Parallelization: can be implemented in parallel with the broker once the policy decision request/response schema is fixed.

## Action Model + Role Taxonomy

- [ ] Define one canonical `ActionRequest` object family for policy evaluation.
- [ ] Define a closed `action_kind` registry and one typed payload family per action kind.
- [ ] Fail closed on unknown action kinds or unknown action-payload schema IDs.
- [ ] Canonicalize and hash `ActionRequest` with RFC 8785 JCS so `action_request_hash` is deterministic and shared across policy, approvals, audit, and broker read models.
- [ ] Freeze initial MVP action kinds for at least:
  - workspace writes
  - executor runs
  - artifact reads
  - excerpt/artifact promotion
  - gateway egress
  - backend posture changes
  - gate overrides
  - stage summary sign-off
- [ ] Keep `ActionRequest` as the canonical policy contract without forcing broker API request families to collapse into one generic transport envelope.
- [ ] Define role taxonomy across manifests, identities, audit, and policy:
  - `actor_kind`
  - `role_family = workspace | gateway`
  - concrete `role_kind` values such as `workspace-read`, `workspace-edit`, `workspace-test`, `model-gateway`, `auth-gateway`, `git-gateway`, `web-research`, `dependency-fetch`
- [ ] Eliminate overlapping generic role vocabulary so `role_kind` always names a concrete least-privilege role rather than a mixed family/subtype term.
- [ ] Freeze a reviewed `role_kind x action_kind x executor_class` policy matrix so workflow, runner, and TUI features reuse one execution-authorization model.

Parallelization: can be implemented in parallel with protocol schema work once the `ActionRequest` and role-identity contracts are fixed.

## Gateway Model

- [ ] Define a shared typed gateway-destination model:
  - `DestinationDescriptor` identifies the canonical external target
  - `GatewayScopeRule` (or equivalent allowlist entry) defines permitted operations, allowed egress data classes, and redirect posture
- [ ] Define MVP destination descriptor families:
  - `model_endpoint`
  - `auth_provider`
  - `git_remote`
  - `web_origin`
  - `package_registry`
- [ ] Keep gateway allowlists as signed, content-addressed inputs referenced by manifests; raw URL-based ad hoc policy decisions are forbidden.
- [ ] Keep gateway hardening invariants invariant-owned:
  - TLS requirements
  - SSRF/private-range blocking
  - DNS rebinding protections
  - redirect-only-to-allowlisted-destination rules

Parallelization: can be designed in parallel with gateway feature lanes once descriptor families and allowlist-entry shapes are fixed.

## Invariants (Fail Closed)

- [ ] Enforce MVP invariants:
  - no escalation-in-place
  - deny-by-default for network/filesystem/shell/secrets
  - network egress is a hard boundary:
    - workspace roles have zero direct network egress
    - public network egress is only allowed for explicit gateway roles (declared in signed manifests and enforced by policy)
      - model inference egress is only possible via the dedicated `model-gateway` role
      - other egress categories (git remote ops, web research, dependency fetch, provider auth) must be isolated behind dedicated gateway roles with strict allowlists and no workspace access
    - any non-gateway network egress attempt is denied and is not approvable
  - no single role combines workspace RW + public egress + long-lived secrets
- [ ] Define a closed `hard_floor_operation_class` taxonomy separate from `approval_trigger_code`:
  - `trust_root_change`
  - `security_posture_weakening`
  - `authoritative_state_reconciliation`
  - `deployment_bootstrap_authority_change`
- [ ] Allow one action to match multiple hard-floor classes and require the strongest resulting assurance floor.
- [ ] Define a closed `executor_class` taxonomy that distinguishes ordinary workspace executors from system-modifying execution.
- [ ] Ordinary workspace executors are workspace-scoped, offline, typed, and non-privileged; raw shell is not implicitly ordinary.
- [ ] System-modifying execution includes host/global state changes, out-of-workspace writes, system package installs, service/network/kernel/container configuration, and persistent OS/user config changes.
- [ ] Ensure policy preserves the shared rule that ordinary `workspace-test` behavior does not silently inherit `system_modifying` authority.
- [ ] Split dependency behavior cleanly:
  - network fetch/cache-fill is a `dependency-fetch` action
  - offline use of cached read-only dependencies inside the workspace is ordinary workspace execution
- [ ] Rename or clarify the current dependency-install trigger language so it does not blur gateway network fetch with offline consumption.

Parallelization: can be implemented in parallel with gateway specs; invariants are the shared non-negotiable core.

## Approval Policy (MVP: Moderate)

- [ ] Define an approval policy model that controls when an otherwise-allowed action requires explicit human approval.
- [ ] Approval policy/profile is a signed input (part of the run/stage capability manifest).
- [ ] Keep approval scope split between exact-action approval and stage sign-off:
  - exact-action approvals bind one canonical `ActionRequest` hash
  - stage sign-off binds one canonical stage summary hash
  - stage summary changes supersede stale sign-off requests and require re-issuance
- [ ] Keep `ApprovalBoundScope` as operator-facing bound-scope metadata derived from the signed approval contract rather than the trust root by itself.
- [ ] Define the user-involvement slider/profile as a policy mapping for ordinary actions:
  - approval frequency
  - minimum assurance level
  - batching rules
  - TTL/expiry defaults
- [ ] Define `approval_assurance_level` handling for policy outputs and approval requirements.
- [ ] Approval profiles affect only *when* explicit human approval is required for actions already allowed by invariants + the signed manifest:
  - profiles must never convert `deny -> allow`
  - profiles may add additional `require_human_approval` decisions and/or additional denies, but cannot expand capabilities
- [ ] The fixed hard floor for high-blast-radius operations is not slider-controlled and cannot be lowered by profile selection:
  - trust-root changes
  - security-posture weakening
  - authoritative restore/import/reconciliation
  - deployment/bootstrap authority changes
- [ ] Stage sign-off remains profile-controlled rather than automatically belonging to the fixed hard floor.

Approval lifecycle semantics (MVP):
- [ ] Every approval request and decision is typed, signed, and binds to immutable inputs:
  - exact-action approval requests include `{manifest_hash, action_request_hash, relevant_artifact_hashes}`
  - stage sign-off requests bind the stage summary hash plus the active manifest/policy context and relevant artifact hashes
  - approval decision references the approval request hash (no free-floating “approve by label”)
- [ ] Reserve and propagate an `approval_assertion_hash` or equivalent hook so decisions can bind verified step-up assurance evidence when required.
- [ ] Expiry/timeout:
  - approval requests have an explicit TTL (default: 30 minutes) and expire deterministically
  - expired approvals cannot be used; the runner must re-request approval
- [ ] Staleness:
  - if any bound hash changes while awaiting approval, the pending approval is invalidated and must be re-issued
- [ ] Delivery channel is advisory metadata only and must not by itself satisfy approval assurance requirements.
- [ ] Define one common `PolicyEvaluationDetails` family plus typed `required_approval` payload families keyed to approval trigger type.
- [ ] Typed approval details must provide enough context for safe review, including scope, why approval is required, changes if approved, effect if denied/deferred, security-posture impact, blocked work, expiry, and related hashes/artifacts.
- [ ] MVP implements a single approval profile: `moderate`:
  - approvals are checkpoint-style, not per-micro-action:
    - stage capability manifest sign-off (always; includes a structured summary of requested high-risk capability categories)
    - reduced-assurance opt-ins (e.g., container backend)
    - gate overrides
    - enabling gateway egress and/or expanding egress scope (MVP focus: enabling third-party model egress via `model-gateway` and expanding allowed model egress data classes beyond the baseline `spec_text` only)
  - moderate trigger categories (must be explicit in the stage manifest and surfaced at sign-off):
    - file writes outside the workspace volume/root (outside the declared workspace path allowlist)
    - secret access (issuing leases from `secretsd`)
    - dependency fetch/cache enablement and dependency installation that requires gateway egress or system-modifying execution
    - system-modifying command execution (beyond ordinary workspace edit/test executors)
    - gateway egress scope changes (enable a gateway role; change allowlists; expand allowed egress data classes)
    - actions wholly inside the workspace sandbox and within the signed manifest execute without intermediate approvals
- [ ] Later approval-profile expansion lives in `runecontext/changes/CHG-2026-014-0c5d-approval-profiles-strict-permissive/`, and the dedicated `git-remote-ops` trigger category lives in `runecontext/changes/CHG-2026-002-33c5-git-gateway-commit-push-pr/`.

Parallelization: can be implemented in parallel with the workflow runner pause/resume work as long as the approval request/decision schemas are shared.

## Backend Selection Rules

- [ ] MicroVM is the default backend when available.
- [ ] Container backend is only allowed with an explicit opt-in recorded as an approval + audit event.
- [ ] The system must not automatically fall back from microVM to containers.

Parallelization: can be implemented in parallel with launcher backends; it depends only on emitting explicit posture + approval events.

## Decision Outputs

- [ ] Standardize policy decisions:
  - `allow | deny | require_human_approval`
  - stable reason codes
  - structured “required approvals” payloads
- [ ] Define reason-code ownership explicitly:
  - `policy_reason_code` explains the policy outcome
  - `approval_trigger_code` explains why approval is required
  - `error.code` explains failed evaluation or failed approval-consumption behavior
- [ ] Successful evaluation must always yield a `PolicyDecision`, including `deny`; only failed evaluation or failed approval consumption uses the shared protocol `Error` envelope.
- [ ] Decision artifacts must include hashes of all evaluated inputs (`manifest_hash`, `action_request_hash`, `relevant_artifact_hashes`, and other `policy_input_hashes`).
- [ ] Record one primary `policy_reason_code` per decision and keep secondary contributing factors in typed details rather than competing primary codes.
- [ ] Keep `required_approval` payloads aligned with the broker local API approval-summary/bound-scope model so the broker can expose operator-facing approval objects without semantic reshaping.
- [ ] Keep gate-override and reduced-assurance-backend approvals exact-action-bound rather than ambient feature-local exceptions.
- [ ] Persist every policy decision as a typed object with a stable digest and signed audit binding; do not add a separate policy-signing authority in MVP.

Parallelization: can be implemented in parallel with protocol schemas; it depends on a stable decision artifact schema and shared error taxonomy.

## Shared Policy Engine Boundary

- [ ] Implement MVP policy evaluation in a trusted Go package such as `internal/policyengine` rather than a separate daemon.
- [ ] Provide one context-compile step that builds effective policy context from manifests, allowlists, and invariant tables.
- [ ] Provide one deterministic evaluation entrypoint from `ActionRequest` to `PolicyDecision`.
- [ ] Keep component-local structural or integrity checks near their owning component, but route allow/deny/approval-required semantics through the shared engine.
- [ ] Migrate existing artifact-specific policy rules into reusable engine rule sets so `internal/artifacts` is not the long-term home of general policy semantics.

Parallelization: can be implemented in parallel with broker, artifact-store, and runner work once the shared evaluation contract is fixed.

## Acceptance Criteria

- [ ] Every canonical `ActionRequest` is evaluated deterministically and produces a policy decision artifact.
- [ ] `manifest_hash` replays the compiled effective policy context, and contributing source manifest/allowlist hashes remain explicit in `policy_input_hashes`.
- [ ] Policy evaluation does not execute arbitrary code and is deterministic for identical inputs.
- [ ] Unknown action kinds, role kinds, destination descriptor kinds, profile values, or payload schema IDs fail closed.
- [ ] Container usage is blocked unless explicitly opted in and recorded.
- [ ] Violations are auditable and do not partially execute.
- [ ] With the `moderate` approval profile, a typical offline edit+gate step can execute without intermediate approvals once the stage manifest is signed.
- [ ] Exact-action approvals and stage sign-off approvals bind to canonical hashes and become stale when those bound hashes change.
- [ ] Gateway egress decisions are based on typed destination descriptors and signed allowlist inputs rather than raw URL-based ad hoc evaluation.
- [ ] High-blast-radius operations enforce the fixed approval-assurance floor regardless of the selected user-involvement profile.
