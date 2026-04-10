# Tasks

## Role Set

- [ ] Implement `workspace-read`, `workspace-edit`, and `workspace-test` roles.
- [ ] Enforce explicit capability manifests for each role.
- [ ] Keep role naming and manifest taxonomy aligned with the shared policy `role_family` / concrete `role_kind` model.
- [ ] Freeze the role-to-executor policy matrix so each workspace role has explicit reviewed allowed action and executor boundaries.

## Execution Boundaries

- [ ] Implement constrained executors with allowlisted operations.
- [ ] Block shell passthrough behavior.
- [ ] Distinguish ordinary workspace executors from system-modifying execution using the shared `executor_class` model.
- [ ] Define typed reviewed `executor_id` contracts with explicit argv, working-directory, environment, timeout, and network rules.
- [ ] Fail closed on unknown executor shapes rather than treating them as ordinary workspace execution.
- [ ] Treat wrapper normalization and shell detection as defense-in-depth on top of the reviewed typed executor registry rather than the primary execution-authority model.
- [ ] Keep `workspace-read` from silently becoming a general command-execution role.
- [ ] Keep `workspace-edit` limited to typed workspace writes plus `workspace_ordinary` execution.
- [ ] Keep `workspace-test` limited to offline `workspace_ordinary` execution and workspace/build-output writes rather than implicit `system_modifying` authority.

## Offline Posture

- [ ] Enforce no direct network egress from workspace roles.
- [ ] Route required cross-boundary data movement through artifacts.

## Acceptance Criteria

- [ ] Role execution remains least-privilege and offline by default.
- [ ] Workspace roles cannot bypass runner, policy, or broker controls.
- [ ] Non-shell-passthrough is defined and enforced as typed executor usage rather than best-effort shell heuristics alone.
- [ ] Role kind and executor class remain distinct in policy and implementation rather than collapsing into one mixed vocabulary.
