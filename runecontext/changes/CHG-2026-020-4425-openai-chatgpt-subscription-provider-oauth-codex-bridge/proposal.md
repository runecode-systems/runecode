## Summary
RuneCode can access GPT models via a ChatGPT subscription OAuth flow without expanding the trust boundary, using broker-mediated setup and auth flows surfaced consistently through TUI and CLI thin clients rather than runtime-local authority.

## Problem
This feature now has a canonical RuneContext change record, preserving the migrated planning content without relying on legacy Agent OS folders or path aliases.

## Proposed Change
- Official OAuth Client Registration.
- Auth-Gateway Role (Auth Egress Only).
- Model-Gateway Bridge via Codex App-Server.
- Policy + Audit Integration.
- Broker-mediated setup and account-linking flows exposed through guided TUI and straightforward CLI clients.

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
Keeps OpenAI ChatGPT Subscription Provider (OAuth + Codex Bridge) reviewable as a RuneContext-native change, aligned with the reviewed auth, bridge, and git-gateway setup-authority decisions, and removes the need for a second semantics rewrite later.
