## Summary
RuneCode turns canonical session objects into real work orchestration for both live chat and autonomous operation while keeping execution inside the same isolate-backed, broker-owned workflow architecture.

## Problem
The product already has session and transcript foundations plus a first-class chat route, but that is not yet the same thing as session-driven work execution. If live chat, background automation, and isolate-backed workflows are allowed to grow separately, the product will accumulate multiple ways to do the same work with different trust and lifecycle semantics.

## Proposed Change
- One shared session-to-execution trigger model for live chat and autonomous operation.
- Broker-owned orchestration from session turns into isolate-backed work.
- Canonical links from sessions to runs, approvals, artifacts, audit records, and project-context state.
- Partial-turn, reconnect, and wait/resume behavior aligned with existing broker and runner truth.
- Execution continuation and follow-up behavior that explicitly inherits the repo-scoped product lifecycle, broker-owned attach semantics, and diagnostics/remediation-only reconnect posture established by `CHG-2026-047-c3e2-local-control-plane-bootstrap-persistent-session-lifecycle-v0`.

## Why Now
This work now lands in `v0.1.0-alpha.7`, because once RuneCode has direct model access, verified project substrate, and persistent local lifecycle, the next user-facing step is making chat and autonomous modes drive the same real execution path.

Landing this before the first-party workflow pack keeps both interaction modes on one durable orchestration substrate instead of making workflow triggering a later bolt-on.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.
- Assistant output remains advisory until it passes through the normal workflow, policy, approval, and isolate-execution path.
- `CHG-2026-047-c3e2-local-control-plane-bootstrap-persistent-session-lifecycle-v0` defines the product-lifecycle foundation this change builds on: one repo-scoped product instance per authoritative repository root, canonical `runecode` attach/start flows, and broker-owned diagnostics/remediation-only attach when repository substrate blocks normal operation.

## Out of Scope
- Re-introducing legacy Agent OS planning paths as canonical references.
- Bypassing the isolate-backed workflow path for convenience in live chat.
- Creating separate orchestration semantics for autonomous mode.
- Redefining product bootstrap, product attach, or broker lifecycle posture semantics already frozen by `CHG-2026-047-c3e2-local-control-plane-bootstrap-persistent-session-lifecycle-v0`.

## Impact
Creates one durable execution substrate for live chat and autonomous operation, keeping sessions, runs, approvals, artifacts, audit records, and project context on the same broker-owned model.
