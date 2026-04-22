## Summary
RuneCode can start, supervise, and reconnect to its local control plane as one product lifecycle while sessions and runs persist beyond the life of the TUI.

## Problem
The current foundation already has trusted local services and a TUI client, but it still reads like a collection of components instead of one attachable local product. A user should not need to keep the TUI open to preserve work, and they should not have to manually reconstruct local service state after routine restarts.

Without an explicit feature, the system risks either staying too manual for normal use or introducing daemon-private lifecycle truth that bypasses the broker and existing typed contracts.

## Proposed Change
- Local control-plane bootstrap and supervision entry flows.
- Persistent session lifecycle and reconnect semantics.
- Broker-projected readiness and degraded-state posture for attachable clients.
- TUI and CLI attach/detach flows that remain thin clients of broker-owned state.
- A canonical top-level `runecode` user command that treats product bootstrap and attach as RuneCode semantics rather than exposing `runecode-broker` plumbing as the long-term user surface.
- Repo-scoped local product instances keyed by authoritative repository root so lifecycle, session persistence, and project-substrate posture stay aligned to the correct canonical project context.

## Why Now
This work now lands in `v0.1.0-alpha.7`, immediately after direct model access and verified project substrate work, because the next user-facing step is making RuneCode behave like one coherent local product rather than a manual component assembly.

Landing lifecycle and reconnect semantics before interactive execution also avoids a later split between "chat mode while the TUI is open" and "background product mode".

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.
- Broker remains the only public control-plane surface even when local bootstrap and supervision behavior grows.
- `v0` should freeze one local RuneCode product instance per authoritative repository root rather than a single per-user global broker identity.
- Reconnect should remain available for inspection and remediation even when current repository project-substrate posture blocks normal managed operation.
- Explicit `start`, `attach`, `stop`, `restart`, and non-starting `status` flows belong in this feature so the product lifecycle surface is complete enough for future clients and platform-specific service realizations.

## Out of Scope
- Remote or multi-host control-plane topologies.
- Re-introducing legacy Agent OS planning paths as canonical references.
- A second daemon-private user API for local lifecycle truth.
- Detailed execution-resume policy when project-substrate snapshot bindings drift across reconnect; `CHG-2026-048-6b7a-session-execution-orchestration-v0` remains responsible for execution-specific resume and continuation rules.
- Migrating every existing low-level broker subcommand under the new top-level command surface in this change; `runecode-broker`, `runecode-launcher`, and other low-level binaries remain valid plumbing, admin, and dev entrypoints.

## Impact
Turns the current trusted local components into an attachable repo-scoped product lifecycle where work can continue without the TUI staying open, while preserving broker-owned truth, typed read/write contracts, and a canonical RuneCode user command surface.

This also freezes the foundation that future CLI, TUI, dashboard, Windows, macOS, and service-manager work should build on:
- local bootstrap and supervision remain private trusted mechanics
- broker-owned typed lifecycle posture becomes the only public authority for attachability and degraded-state semantics
- repository project-substrate compatibility remains read-only during ordinary start, attach, reconnect, and status flows
- attach remains possible in diagnostics/remediation-only posture when services are healthy but normal operation is blocked by repository substrate state
