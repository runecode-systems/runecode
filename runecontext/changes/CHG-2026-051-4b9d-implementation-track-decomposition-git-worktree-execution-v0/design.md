# Design

## Overview
Define a broker-owned model for decomposing implementation work into low-coupling tracks and executing eligible tracks in isolated git worktrees without weakening policy, approval, lifecycle, or project-context guarantees.

## Key Decisions
- Track decomposition must remain broker-owned execution truth rather than client-local, transport-local, or agent-local hidden scheduling state.
- Explicit track declarations from change/spec/implementation inputs take precedence over inferred decomposition.
- Inferred track grouping must become a broker-owned proposed execution-plan artifact rather than a hidden heuristic.
- Git worktrees are the preferred isolation substrate for low-coupling parallel implementation tracks, but they are not mandatory for every implementation plan.
- Worktree execution should remain fail closed: if overlap risk, dependency ambiguity, or project-context drift makes safe parallelization unclear, RuneCode should pause for operator input or fall back to a more conservative execution mode.
- Pending operator input or formal approval should block only the directly affected track and direct downstream dependent tracks; unrelated eligible tracks may continue only when the active plan, dependency graph, policy, coordination state, and project-substrate posture allow it.
- Multiple pending waits may coexist simultaneously; resolution of one wait resumes only the affected track(s) and newly unblocked dependents.
- Track execution, worktree lifecycle, and final integration must preserve canonical links to sessions, runs, approvals, artifacts, audit records, and validated project-context bindings.
- Worktree paths, branch names, and local filesystem mechanics remain implementation-private and must not become public object identity.
- Shared-workspace concurrency remains a distinct concern from isolated-worktree execution; this change should not silently collapse the two models.

## Track Decomposition Model

### Explicit And Inferred Tracks
- Track declarations may come explicitly from:
  - approved change documents
  - approved specs
  - operator-supplied implementation docs or equivalent canonical inputs
- When explicit track declarations are absent, RuneCode may infer candidate tracks from approved implementation inputs.
- Explicit declarations always override inferred grouping.

### Proposed Execution Plan Artifact
- Inferred decomposition must be materialized as a broker-owned proposed execution-plan artifact.
- That artifact should carry at least:
  - stable track identities
  - track source kind (`explicit_declared` or `inferred_proposed`)
  - dependency edges
  - readiness / blocked posture
  - confidence or overlap-risk summary sufficient for operator review and orchestration policy
- This keeps decomposition reviewable, resumable, and auditable.

### Decomposition Safety Posture
- RuneCode should identify low-coupling tracks only when the resulting plan is safe enough to execute under reviewed policy and orchestration rules.
- If decomposition confidence is low or coupling/overlap risk is high, RuneCode should:
  - pause for operator input when autonomy posture requires it, or
  - execute more conservatively instead of forcing parallelization

## Git Worktree Execution Model

### Preferred Isolation Substrate
- Git worktrees are the preferred isolation substrate for eligible low-coupling implementation tracks because they reduce accidental cross-track interference while preserving normal Git review and recovery semantics.
- Worktree execution should be used only when:
  - overlap risk is low
  - dependency edges are explicit enough for safe scheduling
  - policy and coordination state allow track execution
  - project-context-sensitive bindings remain valid

### Worktree Identity And Privacy
- Track, session, and run identities remain canonical broker-owned objects.
- Worktree paths, local branch names, and related local Git mechanics remain implementation-private and non-authoritative.
- Broker-facing contracts may expose small operator-facing summaries for diagnostics, but must not require local path identity as the public contract.

### Integration And Cleanup
- Final integration of track results must be explicit and auditable rather than an ambient heuristic merge.
- Worktree cleanup, reuse, and crash recovery should remain broker-owned lifecycle mechanics rather than client-local convenience memory.
- Final verification should treat track integration as a reviewed execution phase rather than assuming that isolated track success automatically implies integrated success.

## Partial Blocking And Wait Propagation

- Pending operator input or approval should block only:
  - the directly affected track
  - direct downstream tracks that cannot safely proceed without that decision
- Unrelated eligible tracks may continue only when:
  - the active plan explicitly permits them
  - the dependency graph marks them ready
  - broker policy does not require the pending decision for them
  - coordination state does not block them
  - project-substrate posture allows them
- Multiple pending waits may coexist.
- Resolving one wait resumes only the affected track(s) and newly unblocked dependents; it must not implicitly authorize unrelated waiting tracks.

This keeps "always try to keep useful work moving" aligned with the fail-closed model rather than turning it into an unconditional scheduler guarantee.

## Relationship To Session Execution Orchestration

- This change should build on `CHG-2026-048-6b7a-session-execution-orchestration-v0` rather than redefining partial waits, operator input, approval semantics, or turn execution state.
- Session execution orchestration freezes the core rule that pending user input is dependency-aware partial blocking rather than a whole-system stop signal.
- This change extends that rule across explicit or inferred implementation tracks and isolated worktree execution.

## Policy, Approval, And Autonomy Controls

- Formal approval frequency remains under the canonical approval-profile model.
- Operator-question frequency remains under a separate autonomy-posture or equivalent broker-owned orchestration control.
- Track decomposition and track scheduling may use those controls to decide whether to:
  - proceed automatically
  - pause for operator input
  - raise formal human approval where policy requires it
- This change must not introduce system-authored approval decisions or a second approval authority.

## Project-Context Binding And Drift

- Project-context-sensitive tracks should bind to the validated project-substrate snapshot digest used at track planning/execution time.
- Each track should preserve explicit linkage to the bound validated snapshot digest rather than assuming one ambient project state across all parallel work.
- If the bound validated snapshot digest drifts incompatibly for a project-context-sensitive track, that track must fail closed or surface blocked/remediation posture rather than continuing on stale assumptions.

## Shared Contract Alignment

- Track execution should reuse shared workflow identity, policy, approval, audit, and project-context contracts rather than inventing track-local variants of those authority surfaces.
- Any future track-aware workflow/process definition additions should build on the workflow-definition substrate from `CHG-2026-050-e3f8-workflow-definition-contract-binding-v0` rather than creating a second planning format.
- First-party approved-change implementation should be able to adopt this track model later without inventing workflow-pack-local decomposition semantics.

## Main Workstreams
- Broker-Owned Track Decomposition Model
- Proposed Execution-Plan Artifact
- Git Worktree Execution Lifecycle
- Dependency-Aware Partial Blocking And Resume
- Track-Level Policy, Approval, And Project-Context Binding
- Integration, Verification, And Cleanup Semantics

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
