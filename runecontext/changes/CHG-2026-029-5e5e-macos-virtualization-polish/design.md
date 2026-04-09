# Design

## Overview
Improve macOS virtualization reliability, packaging, and UX without changing the core security model.

## Key Decisions
- Any macOS-specific backend changes must preserve the same capability model.
- UX must keep backend kind, runtime isolation assurance, provisioning/binding posture, and audit posture explicit rather than flattening them into one generic “assurance” label.
- Backend kind and runtime posture must remain aligned with the shared broker run-summary/run-detail contract rather than becoming platform-specific UI metadata.
- HVF-backed QEMU or a later Virtualization.framework implementation must preserve the same backend-neutral launch/session/attachment semantics, hardening posture model, and isolate-session audit payload semantics established by the Linux-first microVM backend.

## Main Workstreams
- HVF Reliability + UX
- Optional Virtualization.framework Backend
- Packaging + Permissions

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
