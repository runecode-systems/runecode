# Tasks

## Approval Profile Model (Post-MVP)

- [ ] Define additional approval profiles:
  - `strict`: maximize human involvement (frequent, granular approvals)
  - `permissive`: minimize interruptions while preserving the same security invariants
- [ ] Define how profiles are selected:
  - profile is an explicit field in the run/stage capability manifest (signed input)
  - the system fails closed on unknown profile values
- [ ] Adding a new profile value is a protocol-visible schema change.
  - Version-bump every object family that constrains or surfaces the profile enum, starting with the run/stage capability manifest and any typed summaries that expose the active profile.
- [ ] Profiles must never convert `deny -> allow`; they only affect whether an otherwise-allowed action requires explicit human approval.
- [ ] Define the non-negotiable invariant set that profiles cannot bypass.
- [ ] Define profile mappings against canonical policy `action_kind` values rather than ad hoc feature-local action labels.
- [ ] Define profile mappings for ordinary actions:
  - approval frequency
  - minimum assurance level
  - batching rules
  - TTL/expiry defaults
- [ ] Keep the fixed hard-floor categories from `runecontext/changes/CHG-2026-007-2315-policy-engine-v0/` outside profile control.
- [ ] Keep profile behavior aligned with the policy split between exact-action approvals and stage sign-off.
- [ ] Keep gate overrides explicit approvals across all profiles and do not batch them into ambient milestone sign-off.
- [ ] Keep `git_remote_ops` explicit exact-action approvals across all profiles and do not batch them into stage sign-off, milestone approval, or ambient acknowledgment.
- [ ] Keep `git_remote_ops` approval payload binding aligned with canonical repository identity, target refs, referenced patch artifact digests, expected result tree hash, and canonical action request hash.
- [ ] Keep the minimum assurance floor for `git_remote_ops` at least `reauthenticated` across all profiles.
- [ ] Keep profile mappings aligned with the shared executor-class model so stricter or more permissive timing does not blur `workspace_ordinary` versus `system_modifying` actions.

Cross-cutting approval lifecycle rules (applies to all profiles):
- [ ] Approvals are typed, signed, and hash-bound to immutable inputs (manifest hash + request hash + relevant artifact hashes).
- [ ] Approvals have explicit TTL/expiry; stale approvals are invalid and must be re-requested.
- [ ] Remote approvals become authoritative only through the same signed approval artifact and assurance model as local approvals; delivery channel alone is never sufficient.

Parallelization: can be designed in parallel with policy engine work; depends on stable approval request/decision schemas.

## Strict Profile Semantics

- [ ] Define which action categories require approval in `strict` mode (illustrative):
  - step start/resume
  - workspace writes
  - command execution (even via allowlisted executors)
  - artifact publication beyond the current step
  - all egress-related opt-ins (model, auth, git, web)
- [ ] Define batching rules to prevent UX deadlocks (e.g., approve N related writes in one approval request).
- [ ] Keep exact-action approvals and stage sign-off distinct even when `strict` increases approval frequency.

Parallelization: can be designed in parallel with TUI work; it depends on structured approval payloads and clear reason codes.

## Permissive Profile Semantics

- [ ] Define `permissive` mode as approve at milestones while keeping the same enforcement boundaries:
  - stage manifest sign-off remains required
  - posture-changing actions (e.g., container backend, new egress scopes) remain explicit approvals
  - gate overrides remain explicit approvals
  - when git-gateway exists: require an explicit final approval for git remote state changes (push/tag/PR creation)
- [ ] Ensure `permissive` mode does not introduce batch, milestone, or durable pre-approval semantics for `git_remote_ops`; each approved remote mutation remains exact-action and hash-bound.
- [ ] Ensure `permissive` does not silently convert `workspace-test` or similar ordinary workspace roles into `system_modifying` execution without explicit exact-action approval.

Parallelization: can be designed in parallel with workflow runner work; it depends on the policy engine being the only pause/resume authority.

## Policy + Runner + TUI Integration

- [ ] Extend the policy engine to interpret `strict` and `permissive` profiles.
- [ ] Ensure the workflow runner pauses only on policy-returned `require_human_approval` decisions.
- [ ] Ensure the TUI can:
  - display the active profile
  - display the required assurance level
  - explain why an approval is required (reason codes + structured payload)
  - show what changes if approved
  - support the same approval semantics whether the decision was delivered locally or remotely
- [ ] Keep profile-driven approval semantics aligned with canonical `policy_reason_code`, `approval_trigger_code`, and hard-floor classes rather than inventing profile-local status vocabularies.
- [ ] Keep broker-visible run and approval summaries that surface active profile or required assurance aligned with the same schema/versioning rules.
- [ ] Keep profile behavior aligned with shared gate-override semantics and typed gate-evidence-linked review flows.
- [ ] Keep blocked project-substrate posture outside profile control; profiles must not convert diagnostics/remediation-only project posture into normal operation.

Parallelization: can be implemented in parallel across policy/runner/TUI as long as the approval schema contract is fixed first.

## Acceptance Criteria

- [ ] The system supports `strict` and `permissive` profiles with deterministic behavior and clear audit events.
- [ ] Profiles cannot weaken core invariants (deny-by-default, no escalation-in-place, no host mounts, no unsafe role capability combinations).
- [ ] Profiles must never convert `deny -> allow`.
- [ ] Attempting to use an unknown profile value fails closed.
- [ ] Profiles do not weaken the fixed minimum assurance floor for hard-floor operations.
- [ ] Profiles do not weaken exact-action binding or assurance for `git_remote_ops`, including canonical repo identity, target refs, patch artifact digests, and expected result tree hash.
- [ ] Profiles do not weaken blocked project-substrate posture or convert diagnostics/remediation-only repository substrate states into ordinary execution.

Profile hardening follow-up (pre-MVP foundation):
- [ ] Ensure backend posture approval gating fails closed when profile-specific approval payload derivation is unavailable.
  - `evaluateContainerBackendSelection` must not return a no-op match (`PolicyDecision{}, false`) when `requires_opt_in=true` and no profile mapping resolves.
  - Preserve the invariant that profile expansion cannot silently convert a required-approval posture into an implicit allow path.
