# Design

## Overview
Define Windows microVM runtime support with WHPX/Hyper-V acceleration, strict local IPC, and consistent audit semantics.

## Key Decisions
- Runtime support is distinct from CI portability; CI comes first.
- Windows uses OS-appropriate local IPC and permissions.
- Windows named pipes are a platform-specific transport/auth binding for the same logical broker API, not a Windows-only protocol fork.

## Main Workstreams
- Windows MicroVM Backend Implementation
- Windows Service + Local IPC
- Packaging + Prereqs
- CI/Testing Strategy

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
