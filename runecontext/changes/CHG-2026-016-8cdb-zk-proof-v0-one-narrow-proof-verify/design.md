# Design

## Overview
Select and deliver one narrow zero-knowledge proof workflow with deterministic verification and artifact storage.

## Key Decisions
- ZK is used for integrity attestations of deterministic computations/records, not for proving arbitrary reasoning.
- Proof generation is an explicit workflow step; verification is always deterministic.
- The first ZK proof ships only if a concrete proving system can be selected with acceptable performance; otherwise release is deferred rather than weakening the contract.
- If a proof statement depends on project context, it should bind the validated project-substrate snapshot identity rather than ambient repository assumptions.
- If a proof statement depends on runtime execution identity, it should bind the attested runtime identity seam established by `CHG-2026-030-98b8-isolate-attestation-v0`, using signed runtime-image descriptor identity, persisted reviewed launch evidence, and attestation evidence or verification where relevant rather than ambient platform-specific runtime assumptions.
- If a proof statement depends on external audit anchoring, it should bind the canonical `AuditSegmentSeal` subject plus authoritative anchor receipt identity, canonical target descriptor identity, and typed sidecar proof references rather than raw transport URLs, flattened target-local summaries, or exported-copy artifacts.

## Main Workstreams
- Pick the First Proof Statement
- Choose Proving System + Libraries
- Proof Artifact Format + Storage
- CLI Integration

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
