## Summary
RuneCode ships first-party workflows that can draft RuneContext change and spec documents from prompts and implement approved changes through the same isolate-backed workflow engine, whether triggered from live chat or autonomous operation.

## Problem
The first usable product cut needs real, productive workflows before generic workflow authoring matters. But hard-coding those workflows outside the shared workflow substrate would create the wrong foundation and make later extensibility look like a second system.

## Proposed Change
- First-party change-drafting workflow.
- First-party spec-drafting workflow.
- First-party approved-change implementation workflow.
- Trigger surfaces for live chat and autonomous operation.
- Explicit reuse of the broker-owned session-execution contract from `CHG-2026-048-6b7a-session-execution-orchestration-v0`, including distinct execution-trigger submission, turn-execution watch surfaces, and validated project-substrate snapshot binding for project-context-sensitive work.
- Explicit reuse of separate `approval_profile` and `autonomy_posture` controls so formal approval timing and operator-guidance frequency remain distinct.
- Dependency-aware partial blocking so pending operator input or formal approval pauses only dependent workflow scope and direct downstream work, while unrelated eligible work may continue when plan, policy, coordination, and project-substrate posture allow it.
- Explicit reuse of canonical RuneContext state, workflow contracts, approvals, audit, and git flow bindings.
- Explicit reuse of the broker-owned dependency-fetch and offline-cache foundation so implementation workflows acquire dependency material without workspace internet access and without inventing workflow-local package-manager cache semantics.
- Explicit reuse of the repo-scoped product lifecycle and canonical `runecode` user surface so built-in workflows do not invent a second bootstrap, attach, or remediation path.

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

## Out of Scope
- Generic custom-workflow authoring for arbitrary third-party workflow definitions.
- Re-introducing legacy Agent OS planning paths as canonical references.
- Bypassing the shared workflow substrate for built-in product flows.

## Impact
Creates the first productive built-in workflows for RuneCode while reinforcing both the shared workflow substrate and the canonical repo-scoped RuneCode product lifecycle that future extensibility will build on.

This now also makes the first productive implementation workflows accurate with the clarified dependency foundation:
- dependency fetch is a broker-owned gateway/cache concern
- cached dependency use inside workspace execution is offline internal artifact handoff, not egress
- ordinary cache misses do not become workflow-local approval events
