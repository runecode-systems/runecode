# Tasks

## Drafting Workflows

- [ ] Define a first-party workflow that drafts change documents from user or autonomous prompts.
- [ ] Define a first-party workflow that drafts specs from user or autonomous prompts.
- [ ] Express first-party drafting flows through reviewed `WorkflowDefinition` selection/packaging plus reviewed `ProcessDefinition` executable graph structure rather than a built-in-only execution format.
- [ ] Keep draft outputs bound to canonical RuneContext project state and emitted as reviewable artifacts.
- [ ] Bind drafting workflows to validated project-substrate snapshot identity when project context is relevant.
- [ ] Fail closed for normal drafting flows when repository project-substrate posture is missing, invalid, non-verified, or unsupported.

## Approved-Change Implementation Workflow

- [ ] Define a first-party workflow that implements one or more approved changes.
- [ ] Keep implementation runs on the shared isolate-backed workflow path.
- [ ] Keep runtime execution authority on the broker-compiled immutable `RunPlan` rather than any built-in-only planner or runtime shortcut.
- [ ] Reuse shared approval, audit, verification, and git-gateway semantics where repository mutation is involved.
- [ ] Reuse the shared broker-owned dependency-fetch and offline-cache path when implementation runs need dependency material.
- [ ] Keep ordinary implementation execution from assuming direct workspace network access for package-manager fetches.
- [ ] Keep cached dependency consumption on the internal artifact-handoff path plus ordinary workspace execution rather than treating it as egress.
- [ ] Keep dependency scope enablement or expansion separate from ordinary dependency cache misses in workflow approval behavior.
- [ ] Forbid ordinary implementation execution from implicitly initializing, upgrading, or rewriting repository project substrate.
- [ ] Bind project-context-sensitive implementation runs to validated project-substrate snapshot identity.
- [ ] Bind project-context-sensitive implementation runs to the exact validated project-substrate digest in the compiled `RunPlan` when execution-sensitive project context matters.
- [ ] Preserve the shared distinction between `waiting_operator_input` and `waiting_approval` for drafting and implementation workflow pauses.
- [ ] Block only the exact dependent scope and direct downstream work when operator input or formal approval is pending rather than halting unrelated eligible work.
- [ ] Keep dependency-aware continuation semantics compatible with CHG-050 without promising new parallel-execution behavior in the first built-in slice.
- [ ] Keep `approval_profile` and `autonomy_posture` separate so formal approval timing and operator-guidance frequency are not collapsed into one workflow-local mode.

## Trigger Surfaces

- [ ] Support triggering the workflow pack from live chat/session execution.
- [ ] Support triggering the same workflow pack from autonomous background execution.
- [ ] Keep both entry paths on the same canonical session and run model.
- [ ] Keep plain transcript append separate from workflow execution-trigger submission.
- [ ] Reuse the shared turn-execution watch surfaces rather than introducing a workflow-local live-status channel.
- [ ] Reuse the canonical `runecode` repo-scoped product lifecycle and attach/start flows rather than inventing a built-in-workflow bootstrap or attach path.
- [ ] Keep diagnostics/remediation-only attach from becoming workflow execution authorization when repository substrate blocks normal operation.

## Acceptance Criteria

- [ ] RuneCode can draft change and spec documents on top of canonical RuneContext state.
- [ ] RuneCode can implement approved changes through the shared isolate-backed workflow system.
- [ ] Live chat and autonomous operation can both trigger the same first-party workflow pack.
- [ ] Built-in productive workflows use the refined CHG-050 authority chain rather than a special built-in runtime path.
- [ ] Built-in productive workflows do not bypass shared approval, audit, verification, or git semantics.
- [ ] Built-in productive workflows inherit shared project-substrate gating and snapshot-binding semantics rather than inventing workflow-local project-context rules.
- [ ] Built-in productive workflows inherit the canonical repo-scoped product lifecycle and do not invent a second bootstrap, attach, or remediation path beside `runecode` and broker-owned lifecycle posture.
- [ ] Built-in productive workflows inherit the shared dependency-aware wait model so blocked work pauses only dependent scope and direct downstream work while unrelated eligible work may continue when allowed.
- [ ] Built-in productive workflows remain compatible with `v0` DAG-only workflow/process semantics and do not require promised parallel runtime continuation in the first slice.
- [ ] Built-in implementation workflows acquire dependency material through the shared dependency-fetch/offline-cache foundation rather than through workflow-local package-manager network access.
- [ ] The first end-to-end built-in workflow slice remains compatible with the public-registry-first dependency-fetch posture.
