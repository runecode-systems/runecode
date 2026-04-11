# Design

## Overview
Implement the MVP workspace role set with strict capability boundaries and offline operation posture.

The recommended foundation is one authoritative executor registry in trusted Go, with `RunPlan` binding reviewed executors to workflow scopes. The runner may consume a read-only projection of that registry for dispatch validation, but it must not become a second source of executor policy truth.

## Key Decisions
- Role capabilities remain explicit and least-privilege.
- Workspace roles are offline and non-gateway.
- Command execution uses constrained executors, not raw shell passthrough.
- Workspace roles should use one concrete hyphenated role taxonomy (`workspace-read`, `workspace-edit`, `workspace-test`) aligned with shared policy role-kind semantics rather than a second role vocabulary.
- Execution policy should align with the shared `executor_class` split between ordinary workspace execution and system-modifying execution.
- Role kind and executor class remain distinct dimensions. Role kind describes least-privilege function; executor class describes action risk. The implementation must not collapse them into one overloaded label.
- Non-shell-passthrough means reviewed typed executors with explicit contracts, not arbitrary command strings, shell interpreters, or wrapper chains that hide shell behavior.
- Unknown executor shapes should fail closed rather than being heuristically treated as ordinary workspace execution.
- `workspace-test` remains an offline workspace role and should not silently become a system-modifying role.
- Workspace execution should occur only through `RunPlan`-bound reviewed executor entries rather than ad hoc executor selection by workflow-specific code.
- Trusted Go remains authoritative for executor registry contents. Runner-visible copies are read-only dispatch aids, not the source of authorization semantics.

## Role To Executor Matrix

### `workspace-read`
- Intended for read-oriented operations only.
- Should not perform workspace writes.
- Should not invoke general executor runs by default.
- If future read-only analyzers are introduced, they should be separate typed executors with no shell, no direct network, and no write authority.

### `workspace-edit`
- May perform workspace-scoped writes through typed write operations.
- May use `workspace_ordinary` executors that stay inside the declared workspace scope.
- Must not obtain `system_modifying` execution implicitly.

### `workspace-test`
- May use `workspace_ordinary` executors for local build/test/verification work.
- May write only within approved workspace/build-output scope.
- Remains offline.
- Must not implicitly obtain `system_modifying` execution. Any future system-modifying test/setup path must remain explicit, policy-mediated, and separately reviewable rather than bundled into ordinary workspace-test behavior.

## Typed Executor Model

- Every executable path should be represented by a reviewed `executor_id` with an explicit contract covering at least:
  - allowed argv shape
  - working-directory constraints
  - environment policy
  - network policy
  - timeout/output capture rules
- Executor policy should be registry-driven and typed first, heuristic second.
- Wrapper detection and launcher normalization are defense-in-depth; they must not be the primary authorization model.
- The trusted executor registry should be the only authoritative source for reviewed executor contracts. Any runner-visible projection must remain derived, versioned, and read-only.
- `RunPlan` should bind executor use to specific workflow scopes so ordinary execution is not selected dynamically from ambient tool availability.

## Non-Shell-Passthrough Definition

- The following should not qualify as ordinary workspace execution:
  - raw shell interpreters (`sh`, `bash`, `zsh`, `fish`, `pwsh`, `powershell`, `cmd`, and equivalents)
  - freeform command strings passed to a shell via `-c`, `/c`, or similar patterns
  - generic wrappers whose reviewed contract does not preserve a typed underlying executor identity
- The reviewed execution path should remain a typed executor request rather than an ambient shell string assembled by workflow-specific code.

## Executor Classification Model

- `workspace_ordinary` means:
  - workspace-scoped
  - offline
  - typed
  - non-privileged
  - no raw shell passthrough
- `system_modifying` means operations that can change host/global/system posture or otherwise exceed ordinary workspace scope, including:
  - out-of-workspace writes
  - system package installation
  - service/network/kernel/container configuration
  - persistent OS or user configuration changes
- Policy should evaluate `role_kind x action_kind x executor_class` as a shared reviewed matrix.

## Preferred Registry Shape

- The executor registry should define at least:
  - `executor_id`
  - `executor_class`
  - allowed `role_kind` values
  - reviewed argv-head shape
  - working-directory policy
  - environment allowlist policy
  - network posture
  - timeout and output-capture rules
- Registry evolution should remain explicit and reviewable rather than being inferred from runner behavior.

## Foundation Shortcuts To Avoid

- Do not let executor semantics live independently in trusted Go and the runner.
- Do not treat shell wrappers or heuristic detection as the primary execution-authority model.
- Do not allow workflow-specific code to choose tools outside the reviewed executor registry.
- Do not let `workspace-test` or `workspace-edit` grow implicit system-modifying authority through convenience wrappers.

## Main Workstreams
- Role definitions and capability manifests.
- Trusted executor registry and read-only runner projection rules.
- Artifact handoff and output contracts.
- Role-to-executor policy matrix and fail-closed executor classification.
