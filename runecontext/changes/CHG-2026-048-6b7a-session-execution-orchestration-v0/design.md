# Design

## Overview
Bind canonical session turns to real isolate-backed work orchestration.

## Key Decisions
- Live chat and autonomous operation must share one broker-owned session-to-execution path.
- Session, run, approval, artifact, and audit identities remain canonical broker-visible objects rather than client-local orchestration state.
- Assistant turns may propose workflow actions, but all real repository or project mutation still flows through the shared workflow, policy, approval, and isolate-execution path.
- Session wait, reconnect, and partial-turn behavior should reuse existing broker and runner durability semantics rather than inventing chat-local lifecycle rules.
- Session execution should bind to verified project context when project state is relevant.

## Session Trigger Model

- A session turn should be able to:
  - request model completion only
  - enqueue workflow selection or planning
  - attach to an existing run or approval wait
- The same canonical trigger model should work whether the source is:
  - an interactive user turn
  - an autonomous background turn
  - a resume or follow-up turn after reconnect

## Main Workstreams
- Shared Session-to-Execution Trigger Model.
- Broker-Orchestrated Session Linking to Runs and Approvals.
- Partial-Turn, Wait, and Reconnect Semantics.
- TUI and Client Integration Over Existing Typed Contracts.

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
