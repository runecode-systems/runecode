## Summary
RuneCode supports operator-entered OpenAI-compatible and Anthropic-compatible model endpoints plus API credentials through broker-owned provider profiles, secret-safe CLI/TUI setup flows, and the same secure provider substrate that later OAuth and bridge-runtime features will reuse.

## Problem
The secure provider umbrella already freezes strong shared trust-boundary rules, but the remaining roadmap only exposes provider access through later OAuth and bridge lanes.

That leaves the product without a strong pre-beta path for real remote model access. A narrow API-key-only shortcut would also be the wrong foundation: it would duplicate provider setup, readiness, compatibility, audit, and request-routing logic that later OAuth and bridge features should inherit instead of reimplementing.

Leaving provider-profile identity, secret entry, model-catalog authority, or compatibility posture implicit would create the same rewrite risk under a different name. The first usable direct-credential lane must therefore freeze those contracts now rather than letting them emerge from temporary CLI flags, SDK-shaped payloads, or one-off TUI flows.

## Proposed Change
- Shared provider-profile and auth-material substrate with stable provider-profile identity across direct-credential, OAuth, and bridge-runtime lanes.
- Broker-owned setup-session and secret-ingress flows so CLI and TUI remain thin clients of one trusted setup model.
- Secret-safe operator entry for endpoint configuration and API credentials without making ordinary typed broker request or response objects, CLI args, or environment variables carry raw secret values.
- OpenAI-compatible Chat Completions and Anthropic Messages adapter families beneath the canonical typed model boundary, with future adapter expansion additive rather than rewriting provider profiles or request semantics.
- Broker/TUI/CLI setup, inspection, readiness, compatibility, and audit surfaces that distinguish configuration, credential, connectivity, and compatibility posture explicitly.
- Manual allowlisted model identity remains canonical; provider discovery and probe results remain advisory inputs rather than authority.
- Explicit reuse by later OAuth and bridge-provider lanes.

## Why Now
This work now lands in `v0.1.0-alpha.6`, because the first usable end-to-end product cut needs real remote model access before later interactive execution and workflow-pack features can be valuable.

Landing direct credentials on the same durable provider substrate that later OAuth and bridge features will reuse avoids a second semantics rewrite after the product already depends on provider setup, compatibility, and audit behavior.

Freezing provider-profile identity, secret-ingress rules, readiness posture, and v0 adapter scope before remote model access lands also keeps the pre-beta release from depending on a setup model that later has to be split into profile, auth, and compatibility contracts after the product is already in use.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.
- `secretsd` remains the only long-lived credential store and `model-gateway` remains the only model-egress role.
- Later OAuth and bridge-runtime providers should extend the same provider-profile and auth-material contracts rather than creating separate provider setup semantics.
- Popular SDKs and public API docs may inform adapter implementation, but provider SDK payloads are not authoritative control-plane contracts.
- Broker-visible provider setup and readiness contracts are defined first, and CLI/TUI flows are thin adapters over those contracts rather than separate authority surfaces.

## Out of Scope
- Browser-based OAuth login or refresh flows.
- Bridge-runtime handoff protocols.
- Re-introducing legacy Agent OS planning paths as canonical references.
- Making provider SDK payloads, environment variables, daemon-private setup state, or provider model-discovery APIs authoritative.
- Collapsing direct credentials into a single global provider settings blob.

## Impact
Creates the first usable remote-model access path for RuneCode while preserving one durable provider architecture for direct credentials now and richer provider integrations later.

It also freezes the foundation that future provider work should inherit: stable provider-profile identity, explicit auth-material separation, broker-owned setup authority, secret-safe CLI/TUI onboarding, canonical model selection authority, and broker-projected compatibility posture.
