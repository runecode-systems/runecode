## Summary
RuneCode defines a contract-first workflow-definition and binding foundation for reusable built-in and future custom workflows without introducing new privileged operations or promising new runtime parallel-execution semantics.

This change now explicitly freezes the recommended long-term split:
- `ProcessDefinition` is the authoritative executable workflow graph.
- `WorkflowDefinition` is the reviewed workflow selection, packaging, and policy-binding surface over reviewed process graphs.
- `RunPlan` is the only runtime execution authority consumed by the runner.

## Problem
`CHG-2026-017` mixed two different scopes: the beta-critical workflow definition and binding foundation, and the later authoring/accelerator work that should remain additive.

If the contract-first foundation is not split out, the first productive workflow pack risks either landing on a special-case path or waiting on later authoring and accelerator work that does not need to block the first usable release.

The current repository also already contains placeholder `WorkflowDefinition` and `ProcessDefinition` protocol objects plus a trusted `RunPlan` compiler, but the authority split is still too weak for later features to build on safely. Without clarifying that split now, later work risks drifting into:
- two peer planning inputs that both look executable
- broker-side plan reconstruction from trusted artifacts instead of one canonical compiled plan
- workflow-local wait, dependency, project-context, or git semantics that bypass shared control-plane contracts
- topology-specific shortcuts that behave differently on small devices versus larger deployments

## Proposed Change
- Make `ProcessDefinition` the typed schema-validated executable graph surface for workflow composition.
- Make `WorkflowDefinition` the reviewed selection, packaging, and binding surface over process graphs rather than a second peer execution authority.
- Define typed control-flow as a normalized DAG of stable scopes and dependency edges, not a plugin system, nested DSL, or implicit runtime convention.
- Keep `v0` DAG-only with no general loops or cycles; retries and reruns continue to use separate attempt identities.
- Keep selected definitions hash-bound through canonical RFC 8785 JCS bytes and bind them into a broker-owned signed compilation/selection record rather than relying on ambient repository state.
- Make one broker-owned immutable `RunPlan` artifact the canonical runtime execution contract for a run.
- Reuse shared identity, executor, gate, approval, audit, runner-report, project-substrate, dependency-fetch, offline-cache, and git-request contracts rather than defining workflow-local variants.
- Encode dependency-aware continuation and scoped waits so the broker can distinguish blocked work from unrelated eligible work without forcing a whole-workflow stop model.
- Preserve one topology-neutral architecture across constrained local devices and larger vertically or horizontally scaled deployments; performance tuning must not fork trust or contract semantics.
- Keep generic authoring UX, shared-memory accelerators, and later track-aware execution additive on top of this substrate.

## Why Now
This work now lands in `v0.1.0-alpha.8`, because the first productive workflow pack needs a durable reusable contract foundation before built-in and later custom workflows can share one execution model.

Splitting the contract-first substrate from later authoring and accelerator work avoids coupling the first usable product cut to scope that should remain additive after `v0.1.0-beta.1`.

Locking the authority model now also avoids a more expensive future migration where first-party workflows, custom workflows, session execution, dependency fetch, and track decomposition would otherwise each grow partially overlapping planning semantics.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.
- Later generic authoring and accelerator work should build on this contract foundation rather than redefining it.
- The same logical control-plane architecture should serve Raspberry Pi-class local use and larger deployments; topology may change implementation tuning but not object identity, authority splits, or security semantics.

## Out of Scope
- Generic workflow-authoring UX and review flows.
- Shared-memory accelerators for derived artifacts.
- Guaranteed parallel execution of eligible workflow scopes in `v0`.
- General loops, cycles, or workflow-local re-entrant control-flow semantics in `v0`.
- Workflow-local package-manager fetch authority, workflow-local project-substrate lifecycle semantics, or workflow-local git remote mutation payload families.
- Re-introducing legacy Agent OS planning paths as canonical references.

## Impact
Creates the durable workflow-definition and binding substrate needed for both the first productive built-in workflows and later generic workflow extensibility without making either path a special case.

This now also freezes that workflow definitions must treat dependency material through the same shared contracts as other high-sensitivity surfaces:
- typed dependency-fetch request identity rather than raw lockfile bytes or tool-private cache state
- broker-owned dependency fetch and cache authority rather than runner-local network access
- broker-mediated internal artifact handoff for offline cached dependency use rather than treating cached dependency material as egress

This change also freezes several foundation-level implementation decisions that later work must inherit rather than reopen:
- one executable graph authority (`ProcessDefinition`)
- one workflow selection/binding authority (`WorkflowDefinition`)
- one runtime execution authority (`RunPlan`)
- one stable logical scope identity model (`stage_id`, `step_id`, `role_instance_id`)
- one shared wait vocabulary and approval split
- one topology-neutral performance posture where bounded execution, compact canonical contracts, and broker-owned authority replace environment-specific architecture forks
