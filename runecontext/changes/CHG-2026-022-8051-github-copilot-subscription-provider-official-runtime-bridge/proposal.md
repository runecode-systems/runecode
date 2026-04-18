## Summary
RuneCode can access Copilot models via an official local runtime bridge in LLM-only mode, with provider setup and account posture remaining broker-owned and surfaced consistently through thin TUI and CLI clients.

## Problem
This feature now has a canonical RuneContext change record, preserving the migrated planning content without relying on legacy Agent OS folders or path aliases.

## Proposed Change
- Official Runtime Bridge Integration.
- Auth-Gateway + Token Delivery.
- Compatibility + Probe Policy.
- Policy + Audit Integration.
- Broker-owned setup and provider account posture surfaced through guided TUI and straightforward CLI clients.

## Why Now
This work remains scheduled for v0.2, and keeping it on this canonical RuneContext change preserves direct roadmap-to-change traceability for later delivery and verification.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.

## Out of Scope
- Runtime implementation of the feature during this migration step.
- Re-introducing legacy Agent OS planning paths as canonical references.

## Impact
Keeps GitHub Copilot Subscription Provider (Official Runtime Bridge) reviewable as a RuneContext-native change, aligned with the reviewed auth, bridge, and git-gateway setup-authority decisions, and removes the need for a second semantics rewrite later.
