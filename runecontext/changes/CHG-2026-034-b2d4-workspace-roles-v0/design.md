# Design

## Overview
Implement the MVP workspace role set with strict capability boundaries and offline operation posture.

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

## Main Workstreams
- Role definitions and capability manifests.
- Executor contract and allowlist rules.
- Artifact handoff and output contracts.
- Role-to-executor policy matrix and fail-closed executor classification.
