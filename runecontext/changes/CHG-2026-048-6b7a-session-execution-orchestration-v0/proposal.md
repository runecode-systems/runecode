## Summary
RuneCode turns canonical session objects into real work orchestration for both live chat and autonomous operation while keeping execution inside the same isolate-backed, broker-owned workflow architecture.

## Problem
The product already has session and transcript foundations plus a first-class chat route, but that is not yet the same thing as session-driven work execution. If live chat, background automation, and isolate-backed workflows are allowed to grow separately, the product will accumulate multiple ways to do the same work with different trust and lifecycle semantics.

## Proposed Change
- One shared session-to-execution trigger model for live chat and autonomous operation.
- Broker-owned orchestration from session turns into isolate-backed work.
- Canonical links from sessions to runs, approvals, artifacts, audit records, and project-context state.
- Partial-turn, reconnect, and wait/resume behavior aligned with existing broker and runner truth.

## Why Now
This work now lands in `v0.1.0-alpha.7`, because once RuneCode has direct model access, verified project substrate, and persistent local lifecycle, the next user-facing step is making chat and autonomous modes drive the same real execution path.

Landing this before the first-party workflow pack keeps both interaction modes on one durable orchestration substrate instead of making workflow triggering a later bolt-on.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.
- Assistant output remains advisory until it passes through the normal workflow, policy, approval, and isolate-execution path.

## Out of Scope
- Re-introducing legacy Agent OS planning paths as canonical references.
- Bypassing the isolate-backed workflow path for convenience in live chat.
- Creating separate orchestration semantics for autonomous mode.

## Impact
Creates one durable execution substrate for live chat and autonomous operation, keeping sessions, runs, approvals, artifacts, audit records, and project context on the same broker-owned model.
