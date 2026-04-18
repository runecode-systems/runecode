# Design

## Overview
Define the ChatGPT-subscription provider path using official OAuth, auth-gateway isolation, and a user-installed Codex bridge runtime.

## Key Decisions
- This provider is post-MVP; MVP uses API-key based providers only.
- RuneCode maintains its own official OAuth client registration.
- Auth egress and model egress are separated.
- `secretsd` is the only long-lived secrets store; bridge runtimes must not persist credentials.
- No environment-variable token injection.
- RuneCode integrates only with officially supported, user-installed runtimes (no bundling/redistribution).
- External runtime identity/version are discovered and logged per request; contract tests validate the RPC surface.
- External runtime compatibility uses a tested range plus compatibility probe so RuneCode does not require updates for every vendor release.
- Sessions are ephemeral by default; persisted conversation state requires explicit manifest+policy enablement.
- Provider login, account linking, and auth posture are broker-mediated typed flows surfaced through guided TUI and straightforward CLI clients rather than runtime-local setup authority.

## Main Workstreams
- Official OAuth Client Registration
- Auth-Gateway Role (Auth Egress Only)
- Model-Gateway Bridge via Codex App-Server
- Policy + Audit Integration

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
