# Tasks

## Broker-Owned Track Decomposition Model

- [ ] Define a broker-owned implementation-track model with stable track identity, dependency edges, and readiness/blocking posture.
- [ ] Support explicit track declarations from approved change/spec/implementation inputs.
- [ ] Support inferred candidate tracks when explicit track declarations are absent.
- [ ] Make explicit track declarations authoritative over inferred grouping.
- [ ] Persist inferred decomposition as a broker-owned proposed execution-plan artifact rather than a hidden heuristic.
- [ ] Carry enough confidence or overlap-risk information for operator review and orchestration policy.

## Git Worktree Execution Lifecycle

- [ ] Define when low-coupling implementation tracks are eligible for isolated git-worktree execution.
- [ ] Keep worktree paths, local branch names, and related filesystem mechanics implementation-private and non-authoritative.
- [ ] Define broker-owned worktree create, health, cleanup, reuse, and crash-recovery behavior.
- [ ] Reuse shared broker-owned dependency-fetch and offline-cache authority across tracks instead of creating per-worktree dependency cache ownership.
- [ ] Keep canonical dependency identity bound to reviewed dependency requests and resolved units rather than worktree paths, unpacked trees, or package-manager-local cache directories.
- [ ] Define how eligible worktrees receive broker-mediated offline dependency materialization or equivalent artifact handoff for execution.
- [ ] Keep canonical linkage from track execution to sessions, runs, approvals, artifacts, audit records, and project-context bindings.
- [ ] Define explicit integration and verification flow for combining track outputs.

## Dependency-Aware Partial Blocking

- [ ] Block only the directly affected track and direct downstream dependent tracks when operator input or formal approval is pending.
- [ ] Allow unrelated eligible tracks to continue only when active plan, dependency graph, broker policy, coordination state, and project-substrate posture all permit it.
- [ ] Support multiple simultaneous pending waits without collapsing them into one global blocked state.
- [ ] Ensure resolution of one wait resumes only the affected track(s) and newly unblocked dependents.

## Policy, Approval, And Autonomy Controls

- [ ] Reuse the canonical approval-profile model for formal approval frequency.
- [ ] Reuse a separate broker-owned autonomy-posture model for operator-question frequency and autonomous continuation posture.
- [ ] Ensure track decomposition/scheduling can pause for operator input when inference confidence is low or overlap risk is high.
- [ ] Ensure no track-execution path mints or substitutes for signed human approval decisions.
- [ ] Ensure dependency scope enablement or expansion remains on the shared approval-bearing checkpoint model rather than becoming a per-worktree or per-track approval surface.

## Project-Context Binding And Drift

- [ ] Bind project-context-sensitive tracks to the validated project-substrate snapshot digest used for planning/execution.
- [ ] Preserve per-track project-context linkage rather than assuming one ambient project state across parallel work.
- [ ] Fail closed or surface blocked/remediation posture when a project-context-sensitive track's bound validated snapshot digest drifts incompatibly.

## Acceptance Criteria

- [ ] RuneCode can represent implementation work as explicit or inferred broker-owned tracks with dependency edges.
- [ ] Git worktrees are used as the preferred isolation substrate for eligible low-coupling tracks without becoming public object identity.
- [ ] Pending operator input or approval blocks only dependent tracks and direct downstream work; unrelated eligible tracks may continue when allowed.
- [ ] Multiple pending waits may coexist and resolve independently.
- [ ] Explicit track declarations override inferred grouping.
- [ ] Inferred decomposition remains reviewable and auditable through a broker-owned proposed execution-plan artifact.
- [ ] Track execution reuses shared policy, approval, audit, lifecycle, and validated project-context binding models instead of inventing parallel semantics.
- [ ] Track execution reuses shared dependency-fetch and offline-cache contracts so worktrees consume derived dependency artifacts without becoming authoritative dependency cache owners.
