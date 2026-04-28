## Summary
RuneCode ships first-party workflows that can draft RuneContext change and spec documents from prompts as reviewable artifacts, explicitly promote those drafts into canonical `runecontext/` state, and implement approved changes through the same isolate-backed workflow engine whether triggered from live chat, direct CLI entrypoints, or autonomous operation.

## Problem
The first usable product cut needs real, productive workflows before generic workflow authoring matters. But hard-coding those workflows outside the shared workflow substrate would create the wrong foundation and make later extensibility look like a second system.

## Proposed Change
- Product-shipped first-party workflow pack in `v0`; reviewed built-in workflow identities are non-overridable by repository-local content.
- First-party change-drafting workflow.
- First-party spec-drafting workflow.
- Explicit draft promote/apply workflow behavior so drafting remains artifact-first and canonical RuneContext file mutation remains reviewable, auditable, and policy-bound rather than ambient local editing.
- First-party approved-change implementation workflow.
- Approved-change implementation consumes one reviewed implementation-input set that may contain one or more approved change/spec inputs by exact digest rather than relying on ambient repository planning state.
- First-party workflows expressed through the refined contract-first authority chain from `CHG-2026-050-e3f8-workflow-definition-contract-binding-v0`:
  - `WorkflowDefinition` as workflow-facing selection/packaging
  - `ProcessDefinition` as executable graph
  - broker-compiled immutable `RunPlan` as runtime execution authority
- Trigger surfaces for live chat, autonomous operation, and direct CLI entrypoints as thin adapters over the same broker-owned contracts.
- Explicit reuse of the broker-owned session-execution contract from `CHG-2026-048-6b7a-session-execution-orchestration-v0`, including distinct execution-trigger submission, turn-execution watch surfaces, and validated project-substrate snapshot binding for project-context-sensitive work.
- Explicit reuse of separate `approval_profile` and `autonomy_posture` controls so formal approval timing and operator-guidance frequency remain distinct.
- Repo-scoped admission control, idempotency, and fail-closed overlap handling so built-in workflows do not race each other through hidden client-local heuristics.
- `v0` guarantees at most one mutation-bearing shared-workspace run per authoritative repository root while still preserving the shared scoped-blocking model and leaving future concurrency work additive.
- Dependency-aware partial blocking so pending operator input or formal approval pauses only dependent workflow scope and direct downstream work, while unrelated eligible work may continue when plan, policy, coordination, and project-substrate posture allow it.
- `v0` built-in workflows preserve DAG-only workflow/process semantics and shared scoped-blocking rules without promising new parallel-execution behavior in the first built-in slice.
- Explicit reuse of canonical RuneContext state, workflow contracts, approvals, audit, and git flow bindings.
- Explicit reuse of the broker-owned dependency-fetch and offline-cache foundation so implementation workflows acquire dependency material without workspace internet access and without inventing workflow-local package-manager cache semantics.
- Public-registry-first end-to-end implementation slice; private-registry credential flows remain out of scope for the first built-in slice.
- Explicit reuse of the repo-scoped product lifecycle and canonical `runecode` user surface so built-in workflows do not invent a second bootstrap, attach, or remediation path.
- Performance foundations built into the same shared architecture for all deployment scales: persisted compiled `RunPlan` reuse, canonical compile-cache keys, streaming artifact IO, coalesced identical dependency misses, bounded/configurable broker concurrency, and topology-neutral identity that does not depend on local paths or deployment shape.

## Why Now
This work now lands in `v0.1.0-alpha.8`, after verified project substrate, session execution, and contract-first workflow binding are in place, because this is the point where RuneCode becomes meaningfully useful to a normal user.

Landing these as first-party workflows on the shared workflow foundation avoids a split between "special built-in behaviors" and "real workflows" before the product reaches its first usable cut.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.
- Approved-change implementation should remain compatible with the shared git-gateway, audit, and verification model rather than inventing a workflow-local repository mutation path.
- `CHG-2026-047-c3e2-local-control-plane-bootstrap-persistent-session-lifecycle-v0` defines the canonical repo-scoped product lifecycle and diagnostics/remediation-only attach posture this workflow pack must inherit rather than bypass.
- The first end-to-end implementation workflow slice should remain compatible with the public-registry-first dependency-fetch foundation rather than depending on private-registry credential flows.
- Direct human edits to canonical RuneContext files remain valid repository inputs, but drift against bound planning, approval, or project-substrate inputs must fail closed rather than being merged heuristically.
- The same overall workflow authority model should hold on constrained local hardware and later scaled deployments; performance improvements optimize the shared architecture instead of introducing environment-specific architecture paths.

## Out of Scope
- Generic custom-workflow authoring for arbitrary third-party workflow definitions.
- Re-introducing legacy Agent OS planning paths as canonical references.
- Bypassing the shared workflow substrate for built-in product flows.
- Repository-local override or shadowing of product-shipped built-in workflow identities in `v0`.
- Pulling private-registry credentialed dependency flows into the first end-to-end built-in implementation slice.
- Introducing a separate small-device-vs-scaled-deployment architecture path.

## Impact
Creates the first productive built-in workflows for RuneCode while reinforcing both the shared workflow substrate and the canonical repo-scoped RuneCode product lifecycle that future extensibility will build on.

This now also makes the first productive implementation workflows accurate with the clarified dependency foundation:
- dependency fetch is a broker-owned gateway/cache concern
- cached dependency use inside workspace execution is offline internal artifact handoff, not egress
- ordinary cache misses do not become workflow-local approval events

It also ensures the first built-in workflows do not drift away from the refined workflow foundation:
- built-in workflow selection/packaging remains on `WorkflowDefinition`
- executable structure remains on `ProcessDefinition`
- runner-consumed runtime authority remains the broker-compiled immutable `RunPlan`

And it freezes the long-term product-facing foundation before generic extensibility work lands:
- built-ins are reviewed product-shipped workflow assets, not ambient repo discovery
- drafting is artifact-first with explicit promote/apply semantics
- implementation runs bind to exact approved input sets rather than ambient planning files alone
- direct CLI entrypoints, live chat, and autonomous triggers remain thin adapters over the same broker-owned execution contracts
- performance and scale work optimize one topology-neutral control-plane architecture instead of fragmenting the product model by deployment size
