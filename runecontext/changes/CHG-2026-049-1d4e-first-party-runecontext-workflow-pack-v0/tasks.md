# Tasks

## Workflow Pack Foundation

- [x] Define the product-shipped first-party workflow catalog for `v0` with stable reviewed built-in workflow identities.
- [x] Forbid repository-local override or shadowing of built-in workflow identities in `v0`.
- [x] Define canonical workflow family names, versions, and provenance expectations for built-in workflows.
- [x] Keep built-in workflow selection broker-owned rather than ambient repository discovery.

## Drafting Workflows

- [x] Define a first-party workflow that drafts change documents from user or autonomous prompts.
- [x] Define a first-party workflow that drafts specs from user or autonomous prompts.
- [x] Express first-party drafting flows through reviewed `WorkflowDefinition` selection/packaging plus reviewed `ProcessDefinition` executable graph structure rather than a built-in-only execution format.
- [x] Keep draft outputs bound to canonical RuneContext project state and emitted as reviewable artifacts.
- [x] Define the typed draft artifact shape and evidence linkage for change/spec drafting outputs.
- [x] Define explicit draft promote/apply behavior so mutation of canonical `runecontext/` files is separate from draft generation.
- [x] Keep draft promote/apply on the same broker-owned mutation, approval, audit, and verification path rather than a direct local write shortcut.
- [x] Keep writable RuneContext scope explicit and narrow for draft promotion.
- [x] Bind drafting workflows to validated project-substrate snapshot identity when project context is relevant.
- [x] Fail closed for normal drafting flows when repository project-substrate posture is missing, invalid, non-verified, or unsupported.

## Approved-Change Implementation Workflow

- [x] Define a first-party workflow that implements one reviewed implementation-input set containing one or more approved changes/specs by exact digest.
- [x] Define the typed implementation-input-set artifact and exact binding fields for approved implementation execution.
- [x] Keep implementation runs on the shared isolate-backed workflow path.
- [x] Keep runtime execution authority on the broker-compiled immutable `RunPlan` rather than any built-in-only planner or runtime shortcut.
- [x] Reuse shared approval, audit, verification, and git-gateway semantics where repository mutation is involved.
- [x] Keep implementation mutation intent bound to reviewed workflow/process structure and compiled `RunPlan` authority rather than free-form prompt intent.
- [x] Reuse the shared broker-owned dependency-fetch and offline-cache path when implementation runs need dependency material.
- [x] Keep ordinary implementation execution from assuming direct workspace network access for package-manager fetches.
- [x] Keep cached dependency consumption on the internal artifact-handoff path plus ordinary workspace execution rather than treating it as egress.
- [x] Keep dependency scope enablement or expansion separate from ordinary dependency cache misses in workflow approval behavior.
- [x] Keep the first end-to-end implementation slice public-registry-first and explicitly out of scope for private-registry credential flows.
- [x] Forbid ordinary implementation execution from implicitly initializing, upgrading, or rewriting repository project substrate.
- [x] Bind project-context-sensitive implementation runs to validated project-substrate snapshot identity.
- [x] Bind project-context-sensitive implementation runs to the exact validated project-substrate digest in the compiled `RunPlan` when execution-sensitive project context matters.
- [x] Bind implementation runs to exact approved-input digests, workflow/process digests, control inputs, and relevant repo-state identity so stale plans fail closed.
- [x] Fail closed on incompatible drift in approved inputs, project substrate, or mutation-sensitive repo state rather than heuristically merging or silently continuing.
- [x] Preserve the shared distinction between `waiting_operator_input` and `waiting_approval` for drafting and implementation workflow pauses.
- [x] Block only the exact dependent scope and direct downstream work when operator input or formal approval is pending rather than halting unrelated eligible work.
- [x] Keep dependency-aware continuation semantics compatible with CHG-050 without promising new parallel-execution behavior in the first built-in slice.
- [x] Keep `approval_profile` and `autonomy_posture` separate so formal approval timing and operator-guidance frequency are not collapsed into one workflow-local mode.

## Trigger Surfaces

- [x] Support triggering the workflow pack from live chat/session execution.
- [x] Support triggering the same workflow pack from autonomous background execution.
- [x] Support direct CLI entrypoints as thin adapters over the same broker-owned trigger and execution-state contracts.
- [x] Keep both entry paths on the same canonical session and run model.
- [x] Keep plain transcript append separate from workflow execution-trigger submission.
- [x] Reuse the shared turn-execution watch surfaces rather than introducing a workflow-local live-status channel.
- [x] Reuse the canonical `runecode` repo-scoped product lifecycle and attach/start flows rather than inventing a built-in-workflow bootstrap or attach path.
- [x] Keep diagnostics/remediation-only attach from becoming workflow execution authorization when repository substrate blocks normal operation.
- [x] Add broker-owned repo-scoped admission control and idempotency for built-in workflow triggers.
- [x] Guarantee at most one mutation-bearing shared-workspace run per authoritative repository root in `v0`.
- [x] Fail closed on unsafe overlap rather than relying on client-local dedupe or local transcript heuristics.

## Performance And Shared-Architecture Foundations

- [x] Persist and reuse compiled `RunPlan` authority rather than depending on repeated workflow/process rescans during routine execution.
- [x] Define canonical compile-cache keys for workflow/process/policy/project-context/input identity.
- [x] Keep built-in workflow/run-plan identity topology-neutral and independent from local paths or deployment shape.
- [x] Keep dependency-miss handling coalesced, bounded, and broker-owned.
- [x] Keep shared execution concurrency bounded and configurable under broker control.

## Acceptance Criteria

- [x] RuneCode can draft change and spec documents on top of canonical RuneContext state as reviewable artifacts.
- [x] RuneCode can explicitly promote reviewed drafts into canonical RuneContext files through the shared audited mutation path.
- [x] RuneCode can implement one reviewed implementation-input set containing one or more approved changes/specs through the shared isolate-backed workflow system.
- [x] Live chat, autonomous operation, and direct CLI entrypoints can all trigger the same first-party workflow pack through the same broker-owned execution contracts.
- [x] Built-in productive workflows use the refined CHG-050 authority chain rather than a special built-in runtime path.
- [x] Built-in workflow identities are product-shipped reviewed assets and cannot be overridden by repository-local content in `v0`.
- [x] Built-in productive workflows do not bypass shared approval, audit, verification, or git semantics.
- [x] Built-in productive workflows bind to exact approved inputs, workflow/process digests, control inputs, and validated project context so incompatible drift fails closed.
- [x] Built-in productive workflows inherit shared project-substrate gating and snapshot-binding semantics rather than inventing workflow-local project-context rules.
- [x] Built-in productive workflows inherit the canonical repo-scoped product lifecycle and do not invent a second bootstrap, attach, or remediation path beside `runecode` and broker-owned lifecycle posture.
- [x] Built-in productive workflows inherit the shared dependency-aware wait model so blocked work pauses only dependent scope and direct downstream work while unrelated eligible work may continue when allowed.
- [x] Built-in productive workflows remain compatible with `v0` DAG-only workflow/process semantics and do not require promised parallel runtime continuation in the first slice.
- [x] Built-in implementation workflows acquire dependency material through the shared dependency-fetch/offline-cache foundation rather than through workflow-local package-manager network access.
- [x] The first end-to-end built-in workflow slice remains compatible with the public-registry-first dependency-fetch posture.
- [x] The first built-in workflow slice establishes performance foundations that optimize one shared architecture across constrained and scaled environments rather than introducing environment-specific workflow architecture.
