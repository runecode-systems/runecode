# Trust Boundaries

## Overview

Runecode has two trust domains separated by a hard boundary:

| Domain | Components | Trust Level |
|--------|------------|-------------|
| **Trusted** | Go control plane + TUI | Full access to host secrets, policy, audit |
| **Untrusted** | TS/Node workflow runner | Zero direct access to secrets or host state |

```
┌─────────────────────────────────────────────────────────────┐
│                      TRUSTED DOMAIN                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │   launcher   │  │    broker    │  │      TUI         │  │
│  │  (Go binary) │  │  (Go binary) │  │   (Go binary)    │  │
│  └──────────────┘  └──────────────┘  └──────────────────┘  │
│  ┌──────────────┐  ┌──────────────┐                        │
│  │   secretsd   │  │    auditd    │                        │
│  │  (Go binary) │  │  (Go binary) │                        │
│  └──────────────┘  └──────────────┘                        │
│                         │                                   │
│            ┌────────────┴────────────┐                     │
│            │  broker local API       │                     │
│            │  (schema-validated)     │                     │
│            └────────────┬────────────┘                     │
└─────────────────────────┼───────────────────────────────────┘
                          │
                          │  schema-validated messages
                          │  (no ad-hoc JSON)
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                    UNTRUSTED DOMAIN                         │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              runner (TS/Node)                        │  │
│  │           workflow execution                         │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Allowed Interfaces

Cross-boundary communication is strictly limited to:

1. **Broker Local API** — The only communication channel between trusted and untrusted domains
2. **Protocol Schemas** — Schema definitions in `protocol/schemas/` define all valid message formats
3. **Protocol Fixtures** — Test fixtures in `protocol/fixtures/` for cross-boundary validation

## Prohibited Bypasses

The following MUST NEVER happen:

- Node runner reading host secrets directly (files, env vars, CLI args)
- Node runner accessing trusted Go internal state via filesystem mounts
- Ad-hoc JSON communication outside schema validation
- Imports from `cmd/` or `internal/` in runner source code
- Direct socket/file access to trusted daemons bypassing the broker

## Enforcement

This boundary is enforced at multiple layers:

| Enforcement Point | Owner Spec |
|-------------------|------------|
| Broker local API auth + schema validation | `agent-os/specs/2026-03-08-1039-broker-local-api-v0/` |
| Deterministic policy decisions | `agent-os/specs/2026-03-08-1039-policy-engine-v0/` |
| Runtime isolation (no host filesystem mounts) | `agent-os/specs/2026-03-08-1039-launcher-microvm-backend-v0/` |
| Optional container backend | `agent-os/specs/2026-03-08-1039-container-backend-opt-in-v0/` |
| CI boundary guardrail | This spec (`npm run boundary-check`) |

## CI Guardrail

The runner includes a mechanical boundary check (`npm run boundary-check`) that fails CI if:
- Runner source files anywhere under `runner/` (excluding `node_modules/`, `dist/`, and guardrail test/tooling files) import from `../../internal/*` or `../../cmd/*`
- Runner source files reference trusted code paths outside allowed `protocol/` access (`protocol/schemas/` and `protocol/fixtures/` only)
- No runner source files are found (fail closed)

This check is intentionally best-effort static analysis, not the final security boundary.
Authoritative runtime enforcement is provided by broker auth/schema validation, policy decisions, and isolation backends in follow-on specs.
