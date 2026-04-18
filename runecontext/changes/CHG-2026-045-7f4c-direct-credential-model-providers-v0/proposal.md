## Summary
RuneCode supports operator-entered OpenAI-compatible and Anthropic-compatible model endpoints plus API credentials through the same secure provider substrate that later OAuth and bridge-runtime features will reuse.

## Problem
The secure provider umbrella already freezes strong shared trust-boundary rules, but the remaining roadmap only exposes provider access through later OAuth and bridge lanes.

That leaves the product without a strong pre-beta path for real remote model access. A narrow API-key-only shortcut would also be the wrong foundation: it would duplicate provider setup, readiness, compatibility, audit, and request-routing logic that later OAuth and bridge features should inherit instead of reimplementing.

## Proposed Change
- Shared provider profile and auth-material model.
- Direct credential setup and secret custody.
- OpenAI-compatible and Anthropic-compatible adapter families below the canonical typed model boundary.
- Broker/TUI/CLI compatibility, readiness, and audit surfaces.
- Explicit reuse by later OAuth and bridge-provider lanes.

## Why Now
This work now lands in `v0.1.0-alpha.6`, because the first usable end-to-end product cut needs real remote model access before later interactive execution and workflow-pack features can be valuable.

Landing direct credentials on the same durable provider substrate that later OAuth and bridge features will reuse avoids a second semantics rewrite after the product already depends on provider setup, compatibility, and audit behavior.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.
- `secretsd` remains the only long-lived credential store and `model-gateway` remains the only model-egress role.
- Later OAuth and bridge-runtime providers should extend the same provider-profile and auth-material contracts rather than creating separate provider setup semantics.

## Out of Scope
- Browser-based OAuth login or refresh flows.
- Bridge-runtime handoff protocols.
- Re-introducing legacy Agent OS planning paths as canonical references.
- Making provider SDK payloads, environment variables, or daemon-private setup state authoritative.

## Impact
Creates the first usable remote-model access path for RuneCode while preserving one durable provider architecture for direct credentials now and richer provider integrations later.
