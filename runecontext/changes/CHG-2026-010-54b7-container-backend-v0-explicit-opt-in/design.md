# Design

## Overview
Define the explicit reduced-assurance container backend, including opt-in UX, hardened defaults, artifact movement, policy integration, and reuse of the shared runtime contracts established by `CHG-2026-009-1672-launcher-microvm-backend-v0`.

## Key Decisions
- Containers are never a silent fallback; they require explicit opt-in and acknowledgment.
- The active backend kind, runtime isolation assurance, provisioning/binding posture where applicable, and audit posture are treated as first-class operator/audit data and must not be collapsed into one generic “assurance” string.
- Container networking is isolated by default (no egress); any allowed egress is enforced via explicit network namespace + firewall/proxy rules, not convention.
- The container backend should reuse the same backend-neutral logical seams as the microVM backend where applicable, including launch intent, attachment planning, hardening posture recording, and terminal reporting, rather than inventing container-only control-plane contracts.
- `backend_kind` remains operator-facing and topology-neutral (`container`, not runtime implementation names such as Docker/Podman/runc).
- Container-specific runtime details remain implementation evidence, not public run identity.

## Main Workstreams
- Shared Backend Contract Alignment
- Opt-In UX + Audit
- Hardened Container Baseline
- No Host Mounts + Artifact Movement
- Policy Integration

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
