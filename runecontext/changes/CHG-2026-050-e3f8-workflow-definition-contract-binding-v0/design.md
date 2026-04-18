# Design

## Overview
Split the contract-first workflow-definition and binding substrate from later authoring and accelerator work.

## Key Decisions
- `ProcessDefinition` is a typed, schema-validated composition surface, not a plugin system.
- Workflow definitions compose only approved existing step types and typed control-flow constructs; they do not add new privileged operations.
- Selected workflow definitions are signed, hash-bound inputs to policy, approval, and audit flows.
- JSON is the canonical on-disk and runtime format for workflow definitions, and JSON Schema is the single validation source of truth.
- Workflow definitions must reuse the shared workflow identity model, typed gate contract, executor model, approval split, and runner-to-broker checkpoint/result model rather than defining process-local variants.
- Workflows that compose git remote mutation must reuse shared typed git request families, signed patch artifacts, exact repository identity, and `git_remote_ops` approval semantics.
- Later authoring surfaces and shared-memory accelerators remain additive work on top of this contract substrate rather than being part of the contract definition itself.

## Shared Contract Reuse

### Identity and Attempts
- Workflow definitions compile into stable logical runtime identities such as `stage_id`, `step_id`, and `role_instance_id`.
- Retries and reruns use separate attempt identities rather than mutating logical scope IDs.

### Executor and Gate Reuse
- Workflow definitions may reference only reviewed typed executors already defined by the shared execution model.
- Workflow definitions must reuse the shared typed gate contract, including `gate_id`, `gate_kind`, `gate_version`, normalized inputs, and gate-evidence semantics.

### Approval, Audit, and Git Binding
- Workflow-defined execution must report progress through the shared runner-to-broker checkpoint/result contract.
- Stage sign-off and exact-action approval semantics remain shared and hash-bound.
- Workflow-composed git remote mutation must route through the same typed git request, patch artifact, repository identity, and exact-approval contracts as built-in git flows.

## Main Workstreams
- `ProcessDefinition` Contract.
- Validation + Canonicalization.
- Policy, Approval, Audit, and Git Binding.
- Contract Split from Later Authoring + Accelerators.

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
