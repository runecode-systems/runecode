# Design

## Overview
Add a coherent local bootstrap and reconnect model on top of the existing brokered control plane.

## Key Decisions
- Bootstrap and supervision remain local product mechanics; they must not become a second public authority surface.
- Broker remains the canonical operator-facing surface for readiness, version, session, and run truth.
- Sessions and runs must persist beyond the life of the TUI so clients can detach and later reconnect safely.
- Attach, detach, and reconnect flows should be defined in topology-neutral terms so later Windows and macOS service implementations do not require a contract rewrite.
- Degraded or blocked local-service posture must be explicit and broker-projected rather than inferred from client-local heuristics.

## Lifecycle Model

- Local product lifecycle should cover at least:
  - start and attach
  - detach while work continues
  - reconnect to active sessions and runs
  - blocked or degraded readiness states
  - clean stop and restart
- Local supervision detail may remain implementation-local, but attachable clients should only depend on broker-visible posture and canonical session/run state.

## Main Workstreams
- Local Bootstrap and Supervision Entry Flows.
- Persistent Session Catalog and Reconnect Semantics.
- Broker-Protected Readiness and Degraded-State Projection.
- TUI and CLI Attach/Detach UX.

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
