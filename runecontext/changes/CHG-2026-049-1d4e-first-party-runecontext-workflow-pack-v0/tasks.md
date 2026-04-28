# Tasks

## Workflow Pack Foundation

- [ ] Define the product-shipped first-party workflow catalog for `v0` with stable reviewed built-in workflow identities.
- [ ] Forbid repository-local override or shadowing of built-in workflow identities in `v0`.
- [ ] Define canonical workflow family names, versions, and provenance expectations for built-in workflows.
- [ ] Keep built-in workflow selection broker-owned rather than ambient repository discovery.

## Drafting Workflows

- [ ] Define a first-party workflow that drafts change documents from user or autonomous prompts.
- [ ] Define a first-party workflow that drafts specs from user or autonomous prompts.
- [ ] Express first-party drafting flows through reviewed `WorkflowDefinition` selection/packaging plus reviewed `ProcessDefinition` executable graph structure rather than a built-in-only execution format.
- [ ] Keep draft outputs bound to canonical RuneContext project state and emitted as reviewable artifacts.
- [ ] Define the typed draft artifact shape and evidence linkage for change/spec drafting outputs.
- [ ] Define explicit draft promote/apply behavior so mutation of canonical `runecontext/` files is separate from draft generation.
- [ ] Keep draft promote/apply on the same broker-owned mutation, approval, audit, and verification path rather than a direct local write shortcut.
- [ ] Keep writable RuneContext scope explicit and narrow for draft promotion.
- [ ] Bind drafting workflows to validated project-substrate snapshot identity when project context is relevant.
- [ ] Fail closed for normal drafting flows when repository project-substrate posture is missing, invalid, non-verified, or unsupported.

## Approved-Change Implementation Workflow

- [ ] Define a first-party workflow that implements one reviewed implementation-input set containing one or more approved changes/specs by exact digest.
- [ ] Define the typed implementation-input-set artifact and exact binding fields for approved implementation execution.
- [ ] Keep implementation runs on the shared isolate-backed workflow path.
- [ ] Keep runtime execution authority on the broker-compiled immutable `RunPlan` rather than any built-in-only planner or runtime shortcut.
- [ ] Reuse shared approval, audit, verification, and git-gateway semantics where repository mutation is involved.
- [ ] Keep implementation mutation intent bound to reviewed workflow/process structure and compiled `RunPlan` authority rather than free-form prompt intent.
- [ ] Reuse the shared broker-owned dependency-fetch and offline-cache path when implementation runs need dependency material.
- [ ] Keep ordinary implementation execution from assuming direct workspace network access for package-manager fetches.
- [ ] Keep cached dependency consumption on the internal artifact-handoff path plus ordinary workspace execution rather than treating it as egress.
- [ ] Keep dependency scope enablement or expansion separate from ordinary dependency cache misses in workflow approval behavior.
- [ ] Keep the first end-to-end implementation slice public-registry-first and explicitly out of scope for private-registry credential flows.
- [ ] Forbid ordinary implementation execution from implicitly initializing, upgrading, or rewriting repository project substrate.
- [ ] Bind project-context-sensitive implementation runs to validated project-substrate snapshot identity.
- [ ] Bind project-context-sensitive implementation runs to the exact validated project-substrate digest in the compiled `RunPlan` when execution-sensitive project context matters.
- [ ] Bind implementation runs to exact approved-input digests, workflow/process digests, control inputs, and relevant repo-state identity so stale plans fail closed.
- [ ] Fail closed on incompatible drift in approved inputs, project substrate, or mutation-sensitive repo state rather than heuristically merging or silently continuing.
- [ ] Preserve the shared distinction between `waiting_operator_input` and `waiting_approval` for drafting and implementation workflow pauses.
- [ ] Block only the exact dependent scope and direct downstream work when operator input or formal approval is pending rather than halting unrelated eligible work.
- [ ] Keep dependency-aware continuation semantics compatible with CHG-050 without promising new parallel-execution behavior in the first built-in slice.
- [ ] Keep `approval_profile` and `autonomy_posture` separate so formal approval timing and operator-guidance frequency are not collapsed into one workflow-local mode.

## Trigger Surfaces

- [ ] Support triggering the workflow pack from live chat/session execution.
- [ ] Support triggering the same workflow pack from autonomous background execution.
- [ ] Support direct CLI entrypoints as thin adapters over the same broker-owned trigger and execution-state contracts.
- [ ] Keep both entry paths on the same canonical session and run model.
- [ ] Keep plain transcript append separate from workflow execution-trigger submission.
- [ ] Reuse the shared turn-execution watch surfaces rather than introducing a workflow-local live-status channel.
- [ ] Reuse the canonical `runecode` repo-scoped product lifecycle and attach/start flows rather than inventing a built-in-workflow bootstrap or attach path.
- [ ] Keep diagnostics/remediation-only attach from becoming workflow execution authorization when repository substrate blocks normal operation.
- [ ] Add broker-owned repo-scoped admission control and idempotency for built-in workflow triggers.
- [ ] Guarantee at most one mutation-bearing shared-workspace run per authoritative repository root in `v0`.
- [ ] Fail closed on unsafe overlap rather than relying on client-local dedupe or local transcript heuristics.

## Performance And Shared-Architecture Foundations

- [ ] Persist and reuse compiled `RunPlan` authority rather than depending on repeated workflow/process rescans during routine execution.
- [ ] Define canonical compile-cache keys for workflow/process/policy/project-context/input identity.
- [ ] Keep built-in workflow/run-plan identity topology-neutral and independent from local paths or deployment shape.
- [ ] Keep dependency-miss handling coalesced, bounded, and broker-owned.
- [ ] Keep shared execution concurrency bounded and configurable under broker control.

## Acceptance Criteria

- [ ] RuneCode can draft change and spec documents on top of canonical RuneContext state as reviewable artifacts.
- [ ] RuneCode can explicitly promote reviewed drafts into canonical RuneContext files through the shared audited mutation path.
- [ ] RuneCode can implement one reviewed implementation-input set containing one or more approved changes/specs through the shared isolate-backed workflow system.
- [ ] Live chat, autonomous operation, and direct CLI entrypoints can all trigger the same first-party workflow pack through the same broker-owned execution contracts.
- [ ] Built-in productive workflows use the refined CHG-050 authority chain rather than a special built-in runtime path.
- [ ] Built-in workflow identities are product-shipped reviewed assets and cannot be overridden by repository-local content in `v0`.
- [ ] Built-in productive workflows do not bypass shared approval, audit, verification, or git semantics.
- [ ] Built-in productive workflows bind to exact approved inputs, workflow/process digests, control inputs, and validated project context so incompatible drift fails closed.
- [ ] Built-in productive workflows inherit shared project-substrate gating and snapshot-binding semantics rather than inventing workflow-local project-context rules.
- [ ] Built-in productive workflows inherit the canonical repo-scoped product lifecycle and do not invent a second bootstrap, attach, or remediation path beside `runecode` and broker-owned lifecycle posture.
- [ ] Built-in productive workflows inherit the shared dependency-aware wait model so blocked work pauses only dependent scope and direct downstream work while unrelated eligible work may continue when allowed.
- [ ] Built-in productive workflows remain compatible with `v0` DAG-only workflow/process semantics and do not require promised parallel runtime continuation in the first slice.
- [ ] Built-in implementation workflows acquire dependency material through the shared dependency-fetch/offline-cache foundation rather than through workflow-local package-manager network access.
- [ ] The first end-to-end built-in workflow slice remains compatible with the public-registry-first dependency-fetch posture.
- [ ] The first built-in workflow slice establishes performance foundations that optimize one shared architecture across constrained and scaled environments rather than introducing environment-specific workflow architecture.
