# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm first-party workflows use the shared workflow substrate rather than a hard-coded product-only execution path.
- Confirm built-in workflow identities resolve through product-shipped reviewed workflow assets and cannot be shadowed by repository-local content in `v0`.
- Confirm first-party workflows use the refined CHG-050 authority chain: `WorkflowDefinition` selection/packaging, `ProcessDefinition` executable graph structure, and broker-compiled immutable `RunPlan` runtime authority.
- Confirm drafting workflows operate on canonical RuneContext project state and keep outputs reviewable as typed artifacts.
- Confirm draft promote/apply remains explicit and uses the shared mutation, approval, audit, and verification path rather than a drafting-only local-write shortcut.
- Confirm approved-change implementation stays on the shared isolate-backed workflow path and reuses approval, audit, verification, and git semantics.
- Confirm approved-change implementation consumes one reviewed implementation-input set containing one or more approved items by exact digest rather than ambient planning state alone.
- Confirm approved-change implementation reuses the shared broker-owned dependency-fetch and offline-cache path when dependency material is needed.
- Confirm built-in workflows do not rely on ordinary workspace package-manager internet access or workflow-local dependency cache authority.
- Confirm cached dependency use inside workspace execution is modeled as broker-mediated internal artifact handoff rather than egress.
- Confirm workflow approval behavior keeps dependency scope enablement or expansion separate from ordinary dependency cache misses.
- Confirm the first end-to-end implementation slice is public-registry-first and does not depend on private-registry credential flows.
- Confirm first-party workflow execution enters through the shared execution-trigger and turn-execution contracts rather than plain transcript append or a workflow-local live-status channel.
- Confirm direct CLI entrypoints, live chat, and autonomous execution are thin adapters over the same broker-owned trigger and execution-state contracts.
- Confirm first-party workflows bind to validated project-substrate snapshot identity where project context is relevant.
- Confirm project-context-sensitive built-in execution binds the exact validated project-substrate digest in the compiled `RunPlan` rather than relying only on ambient repository state or high-level summaries.
- Confirm built-in workflow planning, approval reuse, and execution bind exact workflow/process/input/control identity strongly enough that incompatible drift forces re-evaluation rather than stale continuation.
- Confirm missing, invalid, non-verified, and unsupported repository substrate posture routes workflow entry to diagnostics/remediation rather than ordinary execution.
- Confirm ordinary workflow execution does not silently initialize or upgrade repository project substrate.
- Confirm live chat and autonomous entry surfaces trigger the same workflow pack.
- Confirm first-party workflows preserve `waiting_operator_input` versus `waiting_approval` and keep `approval_profile` separate from `autonomy_posture`.
- Confirm pending operator input or formal approval blocks only dependent scope and direct downstream work, while unrelated eligible work can continue when allowed.
- Confirm the first built-in workflow slice preserves CHG-050 dependency-aware continuation semantics without promising new parallel-execution behavior.
- Confirm repo-scoped admission control and idempotency are broker-owned and that `v0` guarantees at most one mutation-bearing shared-workspace run per authoritative repository root.
- Confirm the roadmap and change text both place this feature in `v0.1.0-alpha.8`.
- Confirm first-party workflows reuse the canonical repo-scoped product lifecycle and do not introduce a built-in-only bootstrap, attach, or remediation path.
- Confirm diagnostics/remediation-only attach does not become workflow execution authorization when repository substrate blocks normal operation.
- Confirm the first end-to-end built-in workflow slice remains compatible with the public-registry-first dependency-fetch posture.
- Confirm compiled `RunPlan` reuse, compile-cache identity, dependency-miss coalescing, and bounded broker-controlled concurrency are implemented as shared-architecture optimizations rather than environment-specific workflow paths.

## Close Gate
Use the repository's standard verification flow before closing this change.
