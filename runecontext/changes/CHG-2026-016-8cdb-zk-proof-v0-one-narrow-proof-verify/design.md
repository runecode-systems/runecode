# Design

## Overview
Select and deliver one narrow zero-knowledge proof workflow with deterministic verification and artifact storage.

## Key Decisions
- ZK is used for integrity attestations of deterministic computations/records, not for proving arbitrary reasoning.
- Proof generation is an explicit workflow step; verification is always deterministic.
- The first ZK proof ships only if a concrete proving system can be selected with acceptable performance; otherwise release is deferred rather than weakening the contract.
- If a proof statement depends on project context, it should bind the validated project-substrate snapshot identity rather than ambient repository assumptions.
- If a proof statement depends on runtime execution identity, it should bind the signed runtime-image descriptor identity and reviewed launch evidence rather than ambient platform-specific runtime assumptions.

## Main Workstreams
- Pick the First Proof Statement
- Choose Proving System + Libraries
- Proof Artifact Format + Storage
- CLI Integration

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
