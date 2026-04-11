# Design

## Overview
Define the dependency-fetch role and offline cache so workspace roles stay offline while builds can consume fetched artifacts.

## Key Decisions
- Inputs are minimal and low-sensitivity (lockfiles only).
- Outputs are read-only artifacts.
- Dependency fetch should use the shared typed gateway destination/allowlist model, while offline consumption of cached dependencies remains ordinary workspace execution rather than implicit egress.
- This split must also preserve the shared executor-class model: gateway-backed dependency fetch is not ordinary workspace execution, while offline cached dependency use inside the workspace remains `workspace_ordinary` execution.

## Main Workstreams
- Dependency Fetch Gateway Contract
- Offline Cache Artifact Model
- Policy + Audit Integration

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
