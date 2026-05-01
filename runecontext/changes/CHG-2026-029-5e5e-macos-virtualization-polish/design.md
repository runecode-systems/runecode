# Design

## Overview
Improve macOS virtualization reliability, packaging, and UX without changing the core security model.

## Key Decisions
- Any macOS-specific backend changes must preserve the same capability model.
- UX must keep backend kind, runtime isolation assurance, provisioning/binding posture, and audit posture explicit rather than flattening them into one generic “assurance” label.
- Backend kind and runtime posture must remain aligned with the shared broker run-summary/run-detail contract rather than becoming platform-specific UI metadata.
- macOS runtime support must consume the same published immutable signed runtime assets, boot-profile contracts, trusted-admission rules, and verified local cache semantics as other platforms rather than introducing a macOS-specific runtime signing or asset-admission path.
- macOS runtime support must preserve the same supported runtime trust posture as Linux:
  - valid attestation required for supported production and user-facing runtime operation
  - fail closed on unavailable, invalid, replayed, or freshness-deficient attestation
  - no macOS-specific TOFU fallback, manual override, or platform exception
- HVF-backed QEMU or a later Virtualization.framework implementation must preserve the same backend-neutral runtime-image identity, launch/session/attachment semantics, hardening posture model, launch-evidence model, and isolate-session audit payload semantics established by the shared runtime foundation.
- HVF-backed QEMU or a later Virtualization.framework implementation must also preserve the same additive attestation evidence and verification model rather than defining a macOS-specific trust path.
- macOS-specific runtime launchers, packaging, and permissions must preserve one local RuneCode product instance per authoritative repository root rather than redefining lifecycle around host-global helpers or platform runtime artifacts.
- macOS bootstrap artifacts, local IPC reachability, and platform packaging state remain private realization mechanics; broker-owned product lifecycle posture remains the operator-facing truth.
- The canonical `runecode` lifecycle surface established by `CHG-2026-047-c3e2-local-control-plane-bootstrap-persistent-session-lifecycle-v0` remains unchanged on macOS even if local trusted realization differs from Linux.
- HVF, Virtualization.framework, and macOS packaging mechanics remain private realization evidence and must not become part of published runtime identity or a second signing trust root.
- Platform reliability or capability limitations on macOS must surface as fail-closed prerequisite or attestation-unavailable posture, not as a supported TOFU downgrade path.

## Main Workstreams
- HVF Reliability + UX
- Optional Virtualization.framework Backend
- Packaging + Permissions

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
