# Design

## Overview
Deliver first-party productive workflows on top of the same typed workflow substrate that future custom workflows will use.

## Key Decisions
- First-party workflows must use the shared workflow definition and binding substrate rather than a hard-coded side path.
- First-party workflow definitions in `v0` are product-shipped reviewed assets; repository-local content must not override or shadow built-in workflow identities.
- First-party workflows must adopt the refined CHG-050 authority split rather than a built-in-only shortcut:
  - `WorkflowDefinition` for workflow-facing selection and packaging
  - `ProcessDefinition` for executable graph structure
  - broker-compiled immutable `RunPlan` for runtime execution authority
- Drafting workflows operate on canonical `runecontext/` project state and should emit reviewable artifacts rather than ambient local edits with no provenance.
- Draft promote/apply is explicit workflow behavior; promotion into canonical RuneContext files must use the same audited mutation path and fail-closed policy posture as other broker-owned repository mutation.
- Approved-change implementation workflows must bind to the same approval, audit, git, and verification semantics as the rest of the control plane.
- The same workflow pack must be triggerable from interactive session turns, autonomous background execution, and direct CLI entrypoints that remain thin adapters over the same broker-owned contracts.
- First-party workflow execution must enter through the broker-owned execution-trigger and turn-execution contracts from `CHG-2026-048-6b7a-session-execution-orchestration-v0` rather than through plain transcript append or a workflow-local live-status channel.
- Direct human edits to canonical RuneContext files remain valid inputs; RuneCode must not assume it is the only author.
- First-party workflows should operate only against supported validated project substrate and must not implicitly initialize or upgrade repository substrate during ordinary workflow execution.
- Where project context matters, drafting and implementation workflows should bind to the validated project-substrate snapshot digest rather than to ambient repo state.
- If repository project-substrate posture is missing, invalid, non-verified, or unsupported, first-party workflow entry should route to diagnostics/remediation posture rather than normal drafting or implementation execution.
- First-party workflows must inherit the repo-scoped product lifecycle and canonical `runecode` attach/start flows established by `CHG-2026-047-c3e2-local-control-plane-bootstrap-persistent-session-lifecycle-v0`; they must not invent a built-in-only bootstrap or remediation path.
- First-party workflows must preserve the shared distinction between `waiting_operator_input` and `waiting_approval` rather than collapsing ordinary operator guidance into formal approval state.
- First-party workflows must preserve the shared split between `approval_profile` and `autonomy_posture`; approval frequency and operator-question frequency are separate controls.
- Pending operator input or formal approval must block only the exact dependent scope and direct downstream work that cannot proceed safely, while unrelated eligible work may continue when the shared plan, policy, coordination state, and project-substrate posture allow it.
- Built-in workflow definitions/process graphs should encode dependency-aware continuation and scoped blocking, but the first built-in slice does not by itself promise parallel execution of unrelated eligible scopes.
- Repo-scoped admission control and idempotency must be broker-owned rather than client-local; `v0` must guarantee at most one mutation-bearing shared-workspace run per authoritative repository root while keeping future concurrency work additive.
- First-party implementation workflows that require dependency material must reuse the shared broker-owned dependency-fetch and offline-cache contracts from `CHG-2026-024-acde-deps-fetch-offline-cache`; they must not rely on ordinary workspace package-manager internet access or workflow-local cache authority.
- First-party implementation workflows should treat dependency scope enablement or expansion as the approval-bearing event and should not turn ordinary dependency cache misses into workflow-local approval prompts.
- The first end-to-end implementation slice is public-registry-first; private-registry credential flows remain out of scope until the shared auth-ready dependency foundation is extended explicitly.
- RuneCode keeps one topology-neutral workflow authority model across constrained local hardware and later scaled deployments; performance work must optimize the shared model rather than introduce environment-specific architecture paths.

## Shared Workflow Substrate Alignment

- First-party workflows should be authored as reviewed workflow-facing definitions that bind to reviewed executable process graphs; they must not define a built-in-only execution format.
- First-party executable structure should remain compatible with the `v0` DAG-only posture of `CHG-2026-050-e3f8-workflow-definition-contract-binding-v0`.
- Project-context-sensitive built-in workflow execution should bind the validated project-substrate digest in the compiled `RunPlan`, not only in high-level trigger or summary surfaces.
- Built-in workflow execution should rely on broker-owned compilation and signed selection/compilation evidence rather than ambient repository discovery or built-in-only runtime shortcuts.

## Workflow Pack Provenance And Catalog

- Built-in workflow identities should resolve through one broker-owned reviewed catalog bundled with the product rather than through ambient repository discovery.
- `v0` must keep built-in workflow ID resolution deterministic and supportable:
  - workflow IDs are stable product-owned identifiers
  - workflow versions are product-reviewed packaging versions
  - repository content may inform execution as approved inputs, but it must not replace the built-in workflow catalog as workflow authority
- Future custom workflow support should be additive through an explicit separate registration/catalog path rather than by allowing repository-local shadowing of built-in workflow IDs.

## Drafting Artifact And Promote Model

- Drafting should be artifact-first from the beginning.
- Prompt -> change draft and prompt -> spec draft should produce reviewable typed artifacts bound to:
  - prompt or triggering request identity
  - selected workflow/process identity
  - validated project-substrate digest when project context matters
  - resulting draft artifact digest(s)
- Promotion into canonical RuneContext files should remain explicit workflow behavior rather than an implicit side effect of draft generation.
- Promote/apply steps must:
  - reuse the shared broker-owned mutation, approval, audit, and verification foundations
  - write only reviewed narrow canonical RuneContext paths required by the selected workflow
  - fail closed rather than silently broadening writable scope or mutating unrelated planning surfaces
- Drafting must not create a second ambient local-edit lane that bypasses reviewed artifacts, broker-owned evidence, or replay-safe mutation semantics.

## Approved-Change Implementation Input Contract

- Approved-change implementation should consume one reviewed implementation-input-set artifact rather than a single ambient path or free-form prompt.
- That implementation-input set should support one or more approved inputs from day one so the authority model does not have to be redesigned later when implementation spans multiple approved changes or specs.
- The authoritative input set should bind exact approved digests for items such as:
  - change documents
  - spec documents
  - implementation planning artifacts when later introduced
- `v0` may still execute conservatively in one shared workspace run; supporting a reviewed set of approved inputs does not imply track concurrency, worktree execution, or scheduler promises in this change.

## Mutation Authority And Writable Scope

- Draft promote/apply and approved-change implementation must reuse one canonical broker-owned repository-mutation model rather than introducing separate drafting and implementation write paths.
- Mutation-bearing workflow steps should bind exact writable intent to reviewed workflow/process structure plus compiled `RunPlan` authority.
- Narrow writable-path scope should be explicit and reviewed for each built-in workflow family; writable scope must not expand dynamically from prompt text or runner-local heuristics.
- Where later git remote mutation is part of the workflow, those actions must remain on the shared git-gateway and exact-approval path from `CHG-2026-002-33c5-git-gateway-commit-push-pr`.

## First-Party Workflow Families

- Prompt -> change draft.
- Prompt -> spec draft.
- Draft promote/apply -> canonical RuneContext file mutation.
- Approved implementation-input set -> implementation run.

Each family should preserve explicit artifact, approval, audit, and project-context linkage so the resulting work remains reviewable and verifiable.

For the approved-change implementation family specifically:
- dependency availability should be requested through the shared broker-owned dependency-fetch path before ordinary workspace execution consumes that material
- cached dependency material should be consumed through broker-mediated internal artifact handoff and derived read-only materialization
- the first end-to-end built-in implementation slice should remain compatible with the public-registry-first dependency-fetch posture

## Trigger Surfaces And Admission Control

- Live chat, autonomous execution, and direct CLI entrypoints should remain thin adapters over the same broker-owned trigger, execution-state, and watch contracts.
- Direct CLI entrypoints should exist in `v0`, but they must not gain separate lifecycle rules, planning logic, or approval shortcuts.
- Built-in workflow routing must remain broker-owned. Clients may request a workflow family or operation, but workflow selection, admission, dedupe, and continuation authority remain in the trusted control plane.
- Repo-scoped admission control should fail closed on unsafe overlap.
- `v0` must guarantee at most one active mutation-bearing shared-workspace run per authoritative repository root.
- Broker-owned idempotency and dedupe should key at least on request identity plus relevant bound workflow and input identity so interactive retries and autonomous retries do not create silent duplicate work.
- The broker may later admit non-mutating or independently safe work more aggressively, but that is a scheduling optimization on the same authority model rather than a separate architecture path.

## Execution Binding And Invalidation

- Project-context-sensitive execution must bind the exact validated project-substrate digest used for execution-sensitive context.
- Built-in workflow compilation and approval reuse should bind a full authority tuple rather than a narrower summary identity alone.
- The recommended authority tuple includes at least:
  - workflow pack version
  - `WorkflowDefinition` digest
  - `ProcessDefinition` digest
  - `approval_profile`
  - `autonomy_posture`
  - approved input-set digest(s)
  - validated project-substrate digest
  - target repository identity
  - relevant base tree or commit identity when mutation-sensitive execution depends on repo state
- If any bound input drifts incompatibly between planning, approval, and execution, the built-in workflow must fail closed by requiring re-evaluation, recompilation, and where relevant new approval rather than attempting heuristic merge or stale-plan continuation.
- Direct human edits remain valid repository inputs, but they do not authorize stale compiled plans or stale approvals to proceed.

## Project-Substrate Gate

- First-party workflow families should inherit the project-substrate contract and blocked-state rules from `CHG-2026-046-a91d-runecontext-verified-project-substrate-compatibility-lifecycle-v0`.
- Direct human edits remain valid repository inputs, but RuneCode-managed workflows must evaluate the resulting repository substrate posture before normal execution.
- Workflow execution must not silently initialize, normalize, or upgrade repository substrate just to make ordinary productive flows succeed.

Where `CHG-2026-047-c3e2-local-control-plane-bootstrap-persistent-session-lifecycle-v0` projects diagnostics/remediation-only attach because repository substrate blocks normal operation:
- users may still attach and inspect state through the canonical RuneCode product lifecycle
- first-party productive workflow entry must remain blocked until compatible project-substrate posture is restored
- built-in workflows must not attempt workflow-local bootstrap repair, substrate initialization, or upgrade as an implicit precondition for execution

## Dependency Foundation For Implementation Workflows

- Implementation workflows should declare dependency needs through shared reviewed dependency request identity or equivalent references rather than treating package-manager-local state as authoritative.
- Broker-owned dependency fetch/cache logic should decide hit, miss, fetch, wait, and artifact handoff behavior; the runner must not become the authority for dependency availability or cache truth.
- Ordinary dependency cache misses should result in broker-owned fetch or broker-owned wait behavior, not ambient workspace network access and not workflow-local approval prompts.
- Private-registry dependency support is not part of the first end-to-end workflow slice, but the foundational contracts must remain auth-ready:
  - registry identity remains separate from auth material
  - cache keys never include secret material
  - the runner never receives registry credentials

## Wait Semantics And Execution State

- Built-in workflows must preserve the shared split between:
  - `waiting_operator_input`
  - `waiting_approval`
- Dependency unavailability and blocked project posture may surface through the broader broker-owned wait vocabulary, but built-in workflow definitions must not invent workflow-local wait kinds or approval shortcuts.
- Scoped blocking must remain a property of the reviewed dependency graph compiled into `RunPlan` plus broker-owned execution state rather than a client-local interpretation of transcript context.

## Performance And Topology-Neutrality

- The first implementation slice should establish the durable performance foundation rather than relying on later architecture rewrites.
- Recommended first-slice performance requirements:
  - persist and reuse compiled `RunPlan` authority instead of repeatedly rescanning workflow/process artifacts
  - compile-cache by canonical workflow/process/policy/project-context/input identity
  - keep workflow/process/run-plan objects compact and explicit rather than inference-heavy
  - stream artifact IO rather than buffering large payloads in memory when avoidable
  - coalesce identical in-flight dependency misses
  - keep concurrency bounded and broker-controlled
  - keep all logical identity topology-neutral and independent from host-local paths, runtime directories, or deployment shape
- These optimizations must strengthen the same control-plane architecture on a Raspberry Pi, a workstation, or later shared/horizontally scaled deployments rather than splitting product semantics by environment.

## Main Workstreams
- Workflow Pack Catalog And Provenance.
- Drafting Workflow Definitions + Process Graphs.
- Draft Artifact Promote/Apply Path.
- Approved-Change Implementation Workflow.
- Session and Autonomous Trigger Integration.
- Direct CLI Trigger Adapters.
- Repo-Scoped Admission Control And Idempotency.
- Execution Binding And Drift Invalidation.
- Project-Substrate Snapshot Binding and Blocked-State Gating.
- Approval, Audit, Git, and Verification Binding.
- Dependency-Aware Wait and Continuation Semantics.
- Performance Foundations For Shared Architecture.

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
