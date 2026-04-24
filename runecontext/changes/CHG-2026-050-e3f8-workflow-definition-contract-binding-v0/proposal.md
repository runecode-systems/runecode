## Summary
RuneCode defines a contract-first workflow-definition and binding foundation for reusable built-in and future custom workflows without introducing new privileged operations or parallel execution semantics.

## Problem
`CHG-2026-017` mixed two different scopes: the beta-critical workflow definition and binding foundation, and the later authoring/accelerator work that should remain additive.

If the contract-first foundation is not split out, the first productive workflow pack risks either landing on a special-case path or waiting on later authoring and accelerator work that does not need to block the first usable release.

## Proposed Change
- `ProcessDefinition` object family for workflow composition.
- Validation and canonicalization for workflow definitions.
- Shared identity, executor, gate, approval, and runner-binding reuse.
- Typed control-flow and wait constructs that can represent branch-local `waiting_operator_input` versus `waiting_approval` and dependency-aware continuation without inventing workflow-local lifecycle semantics.
- Policy, audit, and git-contract binding.
- Explicit split from later authoring and shared-memory accelerator work.

## Why Now
This work now lands in `v0.1.0-alpha.8`, because the first productive workflow pack needs a durable reusable contract foundation before built-in and later custom workflows can share one execution model.

Splitting the contract-first substrate from later authoring and accelerator work avoids coupling the first usable product cut to scope that should remain additive after `v0.1.0-beta.1`.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.
- Later generic authoring and accelerator work should build on this contract foundation rather than redefining it.

## Out of Scope
- Generic workflow-authoring UX and review flows.
- Shared-memory accelerators for derived artifacts.
- Re-introducing legacy Agent OS planning paths as canonical references.

## Impact
Creates the durable workflow-definition and binding substrate needed for both the first productive built-in workflows and later generic workflow extensibility without making either path a special case.
