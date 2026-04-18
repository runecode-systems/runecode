## Summary
RuneCode ships first-party workflows that can draft RuneContext change and spec documents from prompts and implement approved changes through the same isolate-backed workflow engine, whether triggered from live chat or autonomous operation.

## Problem
The first usable product cut needs real, productive workflows before generic workflow authoring matters. But hard-coding those workflows outside the shared workflow substrate would create the wrong foundation and make later extensibility look like a second system.

## Proposed Change
- First-party change-drafting workflow.
- First-party spec-drafting workflow.
- First-party approved-change implementation workflow.
- Trigger surfaces for live chat and autonomous operation.
- Explicit reuse of canonical RuneContext state, workflow contracts, approvals, audit, and git flow bindings.

## Why Now
This work now lands in `v0.1.0-alpha.8`, after verified project substrate, session execution, and contract-first workflow binding are in place, because this is the point where RuneCode becomes meaningfully useful to a normal user.

Landing these as first-party workflows on the shared workflow foundation avoids a split between "special built-in behaviors" and "real workflows" before the product reaches its first usable cut.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.
- Approved-change implementation should remain compatible with the shared git-gateway, audit, and verification model rather than inventing a workflow-local repository mutation path.

## Out of Scope
- Generic custom-workflow authoring for arbitrary third-party workflow definitions.
- Re-introducing legacy Agent OS planning paths as canonical references.
- Bypassing the shared workflow substrate for built-in product flows.

## Impact
Creates the first productive built-in workflows for RuneCode while reinforcing the shared workflow substrate that future extensibility will build on.
