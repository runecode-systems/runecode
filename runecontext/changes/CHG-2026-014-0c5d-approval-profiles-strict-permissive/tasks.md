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

Parallelization: can be designed in parallel with TUI work; it depends on structured approval payloads and clear reason codes.

## Permissive Profile Semantics

- [ ] Define `permissive` mode as approve at milestones while keeping the same enforcement boundaries:
  - stage manifest sign-off remains required
  - posture-changing actions (e.g., container backend, new egress scopes) remain explicit approvals
  - gate overrides remain explicit approvals
  - when git-gateway exists: require an explicit final approval for git remote state changes (push/tag/PR creation)

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

Parallelization: can be implemented in parallel across policy/runner/TUI as long as the approval schema contract is fixed first.

## Acceptance Criteria

- [ ] The system supports `strict` and `permissive` profiles with deterministic behavior and clear audit events.
- [ ] Profiles cannot weaken core invariants (deny-by-default, no escalation-in-place, no host mounts, no unsafe role capability combinations).
- [ ] Profiles must never convert `deny -> allow`.
- [ ] Attempting to use an unknown profile value fails closed.
- [ ] Profiles do not weaken the fixed minimum assurance floor for hard-floor operations.

Profile hardening follow-up (pre-MVP foundation):
- [ ] Ensure backend posture approval gating fails closed when profile-specific approval payload derivation is unavailable.
  - `evaluateContainerBackendSelection` must not return a no-op match (`PolicyDecision{}, false`) when `requires_opt_in=true` and no profile mapping resolves.
  - Preserve the invariant that profile expansion cannot silently convert a required-approval posture into an implicit allow path.
