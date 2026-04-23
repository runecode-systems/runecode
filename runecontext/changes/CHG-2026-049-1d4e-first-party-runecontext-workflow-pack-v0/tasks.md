# Tasks

## Drafting Workflows

- [ ] Define a first-party workflow that drafts change documents from user or autonomous prompts.
- [ ] Define a first-party workflow that drafts specs from user or autonomous prompts.
- [ ] Keep draft outputs bound to canonical RuneContext project state and emitted as reviewable artifacts.
- [ ] Bind drafting workflows to validated project-substrate snapshot identity when project context is relevant.
- [ ] Fail closed for normal drafting flows when repository project-substrate posture is missing, invalid, non-verified, or unsupported.

## Approved-Change Implementation Workflow

- [ ] Define a first-party workflow that implements one or more approved changes.
- [ ] Keep implementation runs on the shared isolate-backed workflow path.
- [ ] Reuse shared approval, audit, verification, and git-gateway semantics where repository mutation is involved.
- [ ] Forbid ordinary implementation execution from implicitly initializing, upgrading, or rewriting repository project substrate.
- [ ] Bind project-context-sensitive implementation runs to validated project-substrate snapshot identity.

## Trigger Surfaces

- [ ] Support triggering the workflow pack from live chat/session execution.
- [ ] Support triggering the same workflow pack from autonomous background execution.
- [ ] Keep both entry paths on the same canonical session and run model.
- [ ] Reuse the canonical `runecode` repo-scoped product lifecycle and attach/start flows rather than inventing a built-in-workflow bootstrap or attach path.
- [ ] Keep diagnostics/remediation-only attach from becoming workflow execution authorization when repository substrate blocks normal operation.

## Acceptance Criteria

- [ ] RuneCode can draft change and spec documents on top of canonical RuneContext state.
- [ ] RuneCode can implement approved changes through the shared isolate-backed workflow system.
- [ ] Live chat and autonomous operation can both trigger the same first-party workflow pack.
- [ ] Built-in productive workflows do not bypass shared approval, audit, verification, or git semantics.
- [ ] Built-in productive workflows inherit shared project-substrate gating and snapshot-binding semantics rather than inventing workflow-local project-context rules.
- [ ] Built-in productive workflows inherit the canonical repo-scoped product lifecycle and do not invent a second bootstrap, attach, or remediation path beside `runecode` and broker-owned lifecycle posture.
