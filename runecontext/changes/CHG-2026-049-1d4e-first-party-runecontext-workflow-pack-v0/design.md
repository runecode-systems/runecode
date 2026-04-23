# Design

## Overview
Deliver first-party productive workflows on top of the same typed workflow substrate that future custom workflows will use.

## Key Decisions
- First-party workflows must use the shared workflow definition and binding substrate rather than a hard-coded side path.
- Drafting workflows operate on canonical `runecontext/` project state and should emit reviewable outputs rather than ambient local edits with no provenance.
- Approved-change implementation workflows must bind to the same approval, audit, git, and verification semantics as the rest of the control plane.
- The same workflow pack must be triggerable from interactive session turns and autonomous background execution.
- Direct human edits to canonical RuneContext files remain valid inputs; RuneCode must not assume it is the only author.
- First-party workflows should operate only against supported validated project substrate and must not implicitly initialize or upgrade repository substrate during ordinary workflow execution.
- Where project context matters, drafting and implementation workflows should bind to the validated project-substrate snapshot digest rather than to ambient repo state.
- If repository project-substrate posture is missing, invalid, non-verified, or unsupported, first-party workflow entry should route to diagnostics/remediation posture rather than normal drafting or implementation execution.
- First-party workflows must inherit the repo-scoped product lifecycle and canonical `runecode` attach/start flows established by `CHG-2026-047-c3e2-local-control-plane-bootstrap-persistent-session-lifecycle-v0`; they must not invent a built-in-only bootstrap or remediation path.

## First-Party Workflow Families

- Prompt -> change draft.
- Prompt -> spec draft.
- Approved changes -> implementation run.

Each family should preserve explicit artifact, approval, audit, and project-context linkage so the resulting work remains reviewable and verifiable.

## Project-Substrate Gate

- First-party workflow families should inherit the project-substrate contract and blocked-state rules from `CHG-2026-046-a91d-runecontext-verified-project-substrate-compatibility-lifecycle-v0`.
- Direct human edits remain valid repository inputs, but RuneCode-managed workflows must evaluate the resulting repository substrate posture before normal execution.
- Workflow execution must not silently initialize, normalize, or upgrade repository substrate just to make ordinary productive flows succeed.

Where `CHG-2026-047-c3e2-local-control-plane-bootstrap-persistent-session-lifecycle-v0` projects diagnostics/remediation-only attach because repository substrate blocks normal operation:
- users may still attach and inspect state through the canonical RuneCode product lifecycle
- first-party productive workflow entry must remain blocked until compatible project-substrate posture is restored
- built-in workflows must not attempt workflow-local bootstrap repair, substrate initialization, or upgrade as an implicit precondition for execution

## Main Workstreams
- Drafting Workflow Definitions.
- Approved-Change Implementation Workflow.
- Session and Autonomous Trigger Integration.
- Project-Substrate Snapshot Binding and Blocked-State Gating.
- Approval, Audit, Git, and Verification Binding.

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
