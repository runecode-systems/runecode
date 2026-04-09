# Design

## Overview
Define the signing and verification pipeline for isolate images and toolchains with fail-closed enforcement, aligned to the digest-addressed runtime image model established by `CHG-2026-009-1672-launcher-microvm-backend-v0`.

## Key Decisions
- Image/toolchain signing keys are separate from manifest signing.
- Enforcement is fail-closed.
- Signing and launch enforcement should operate on a digest-addressed `RuntimeImageDescriptor` rather than mutable tags, loose file paths, or ad hoc per-platform boot references.
- Verification and audit should record both the descriptor digest and the concrete boot component digests actually used.
- The image descriptor model should preserve hooks for later attestation/measurement evidence rather than requiring another image-identity rewrite.

## Main Workstreams
- Signing Key Hierarchy
- Runtime Image Descriptor Alignment
- Build + Publication Pipeline
- Launcher Enforcement
- Audit + Verification Integration

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
