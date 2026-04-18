# Design

## Overview
Define the GitHub Copilot subscription provider path using the shared bridge runtime contract and explicit auth isolation.

## Key Decisions
- This provider is post-MVP and uses the shared bridge/runtime protocol surface.
- Auth egress and model egress remain separated.
- Runtime compatibility is probe-driven and fail-closed.
- Bridge runtimes stay in explicit LLM-only mode with no workspace or patch capabilities.
- Provider setup, account-linking, and auth posture remain broker-owned typed flows rather than runtime-local authority.

## Main Workstreams
- Official Runtime Bridge Integration
- Auth-Gateway + Token Delivery
- Compatibility + Probe Policy
- Policy + Audit Integration

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
