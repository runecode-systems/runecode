# Design

## Overview
Define schema-validated workflow composition and rebuildable shared-memory accelerators without adding new privileged operations.

## Key Decisions
- `ProcessDefinition` is a typed, schema-validated composition surface, not a plugin system.
- Custom workflows compose an allowlist of existing RuneCode step types; they do not add new privileged operations.
- Selected process definitions are signed, hash-bound inputs to policy, approval, and audit flows.
- `ProcessDefinition` uses JSON as its runtime and canonical on-disk format.
- JSON Schema is the single validation source of truth for `ProcessDefinition` objects.
- Future authoring adapters must normalize to the same RFC 8785 JCS canonical JSON bytes before validation and hashing; the authored `ProcessDefinition` surface remains object-rooted even though shared canonicalization semantics are now defined repo-wide.
- Shared memory is a rebuildable accelerator for derived artifacts only; authoritative state remains in the run DB, artifact store, and audit trail.
- `ProcessDefinition` must reuse the shared workflow identity model established by the workflow runner project rather than inventing process-local IDs or retry semantics.
- Custom workflows must reuse the shared typed gate contract, executor model, approval split, and runner->broker checkpoint/result model rather than defining process-local variants.
- Custom workflows that compose git remote mutation must reuse the shared typed git request families, signed patch artifact contracts, exact repository identity model, and `git_remote_ops` approval trigger rather than defining process-local git semantics.

## Shared Contract Reuse

### Identity + Attempt Semantics
- Process definitions should name stable logical workflow scopes that compile into shared runtime identities such as:
  - `stage_id`
  - `step_id`
  - `role_instance_id`
- Retries and reruns should use separate attempt identities rather than mutating those logical scope identities.

### Executor Reuse
- Custom workflows may reference only reviewed typed executors already defined by the shared workspace/gateway execution model.
- Process definitions must not introduce arbitrary shell strings, raw command passthrough, or unreviewed executor contracts.
- Process definitions must not introduce ad hoc git mutation steps, raw git transport payloads, or process-local repository-policy mutation channels.

### Gate Reuse
- Custom workflows must reuse the shared typed gate contract:
  - stable `gate_id`
  - explicit `gate_kind`
  - explicit `gate_version`
  - declared normalized inputs
  - shared gate-attempt and gate-evidence semantics
- Gate ordering and checkpoint placement should be explicit in the process definition so later runs, approvals, and TUI surfaces reuse one model.

### Approval + Runner Contract Reuse
- Custom workflows must preserve the shared approval split between exact-action approvals and stage sign-off.
- Stage sign-off should continue to bind one canonical stage summary hash and become stale when that hash changes.
- Process-defined execution should report progress through the shared runner->broker checkpoint/result contract rather than a process-local status channel.
- If a workflow composes git remote mutation, it must preserve exact-action approval semantics for `git_remote_ops`; process-level milestones or stage sign-off cannot substitute for final remote-mutation approval.
- Workflow-defined git steps must bind the same canonical hashes as built-in git flows, including typed git request hash, referenced patch artifact digests, and expected result tree hash.

## Main Workstreams
- `ProcessDefinition` Contract
- Validation + Canonicalization
- Shared-Memory Accelerators
- Policy, Approval, and Audit Binding
- Authoring + UX Surfaces

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
