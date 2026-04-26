# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm first-party workflows use the shared workflow substrate rather than a hard-coded product-only execution path.
- Confirm drafting workflows operate on canonical RuneContext project state and keep outputs reviewable.
- Confirm approved-change implementation stays on the shared isolate-backed workflow path and reuses approval, audit, verification, and git semantics.
- Confirm approved-change implementation reuses the shared broker-owned dependency-fetch and offline-cache path when dependency material is needed.
- Confirm built-in workflows do not rely on ordinary workspace package-manager internet access or workflow-local dependency cache authority.
- Confirm cached dependency use inside workspace execution is modeled as broker-mediated internal artifact handoff rather than egress.
- Confirm workflow approval behavior keeps dependency scope enablement or expansion separate from ordinary dependency cache misses.
- Confirm first-party workflow execution enters through the shared execution-trigger and turn-execution contracts rather than plain transcript append or a workflow-local live-status channel.
- Confirm first-party workflows bind to validated project-substrate snapshot identity where project context is relevant.
- Confirm missing, invalid, non-verified, and unsupported repository substrate posture routes workflow entry to diagnostics/remediation rather than ordinary execution.
- Confirm ordinary workflow execution does not silently initialize or upgrade repository project substrate.
- Confirm live chat and autonomous entry surfaces trigger the same workflow pack.
- Confirm first-party workflows preserve `waiting_operator_input` versus `waiting_approval` and keep `approval_profile` separate from `autonomy_posture`.
- Confirm pending operator input or formal approval blocks only dependent scope and direct downstream work, while unrelated eligible work can continue when allowed.
- Confirm the roadmap and change text both place this feature in `v0.1.0-alpha.8`.
- Confirm first-party workflows reuse the canonical repo-scoped product lifecycle and do not introduce a built-in-only bootstrap, attach, or remediation path.
- Confirm diagnostics/remediation-only attach does not become workflow execution authorization when repository substrate blocks normal operation.
- Confirm the first end-to-end built-in workflow slice remains compatible with the public-registry-first dependency-fetch posture.

## Close Gate
Use the repository's standard verification flow before closing this change.
