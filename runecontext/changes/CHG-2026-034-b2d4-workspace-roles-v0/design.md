# Design

## Overview
Implement the MVP workspace role set with strict capability boundaries and offline operation posture.

## Key Decisions
- Role capabilities remain explicit and least-privilege.
- Workspace roles are offline and non-gateway.
- Command execution uses constrained executors, not raw shell passthrough.
- Workspace roles should use one concrete hyphenated role taxonomy (`workspace-read`, `workspace-edit`, `workspace-test`) aligned with shared policy role-kind semantics rather than a second role vocabulary.
- Execution policy should align with the shared `executor_class` split between ordinary workspace execution and system-modifying execution.

## Main Workstreams
- Role definitions and capability manifests.
- Executor contract and allowlist rules.
- Artifact handoff and output contracts.
