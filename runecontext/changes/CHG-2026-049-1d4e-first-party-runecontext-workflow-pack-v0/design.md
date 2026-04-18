# Design

## Overview
Deliver first-party productive workflows on top of the same typed workflow substrate that future custom workflows will use.

## Key Decisions
- First-party workflows must use the shared workflow definition and binding substrate rather than a hard-coded side path.
- Drafting workflows operate on canonical `runecontext/` project state and should emit reviewable outputs rather than ambient local edits with no provenance.
- Approved-change implementation workflows must bind to the same approval, audit, git, and verification semantics as the rest of the control plane.
- The same workflow pack must be triggerable from interactive session turns and autonomous background execution.
- Direct human edits to canonical RuneContext files remain valid inputs; RuneCode must not assume it is the only author.

## First-Party Workflow Families

- Prompt -> change draft.
- Prompt -> spec draft.
- Approved changes -> implementation run.

Each family should preserve explicit artifact, approval, audit, and project-context linkage so the resulting work remains reviewable and verifiable.

## Main Workstreams
- Drafting Workflow Definitions.
- Approved-Change Implementation Workflow.
- Session and Autonomous Trigger Integration.
- Approval, Audit, Git, and Verification Binding.

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
