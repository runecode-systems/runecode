# Design

## Overview
Define the dedicated Git gateway for commit, push, and pull-request operations with outbound patch verification and scoped credentials.

## Key Decisions
- Git egress is treated as high risk and is isolated behind a gateway.
- Outbound verification must match signed patch artifacts.
- Git policy should use the shared typed gateway destination model so canonical repo identity and allowed operations are expressed through signed destination descriptors and allowlist entries rather than raw URL checks.

## Main Workstreams
- Git Target Allowlist Model
- Secretsd-Backed Credentials
- Patch Artifact Application + Outbound Verification
- PR Creation

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
