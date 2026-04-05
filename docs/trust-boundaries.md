# Trust Boundaries

## Overview

Runecode has two trust domains separated by a hard boundary:

| Domain | Components | Trust Level |
| --- | --- | --- |
| **Trusted** | Go control plane daemons + TUI client | Privileged local components with least-privilege separation (only `secretsd` stores long-lived secrets; the TUI must not receive secret values; other daemons run without secrets and with restricted OS permissions) |
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
│  │            workflow runner                           │  │
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
- Imports or repo-root path references to `cmd/`, `internal/`, or `tools/` in runner source code
- Absolute filesystem path literals in runner source (Unix absolute paths, Windows drive-letter paths, or UNC paths), except allowed protocol schema/fixture paths
- Direct socket/file access to trusted daemons bypassing the broker

## Enforcement

This boundary is enforced at multiple layers:

| Enforcement Point | Owner Artifact |
| --- | --- |
| Broker local API auth + schema validation | `runecontext/changes/CHG-2026-008-62e1-broker-local-api-v0/` |
| Deterministic policy decisions | `runecontext/changes/CHG-2026-007-2315-policy-engine-v0/` |
| Runtime isolation (no host filesystem mounts) | `runecontext/changes/CHG-2026-009-1672-launcher-microvm-backend-v0/` |
| Optional container backend | `runecontext/changes/CHG-2026-010-54b7-container-backend-v0-explicit-opt-in/` |
| CI boundary guardrail | This spec (`npm run boundary-check`) |

## Crypto Foundation Readiness Hooks

The trusted daemons expose minimal fail-closed validation hooks so later feature work can reuse one reviewed contract without re-inventing parsing and posture checks:

- `runecode-broker promote-excerpt` now requires a signed approval-decision envelope and validates it against broker-owned trusted verifier records before approval consumption.
- `runecode-launcher validate-isolate-binding` validates TOFU isolate session bindings (`run_id`, `isolate_id`, `session_id`, `session_nonce`, digest bindings, and key identity profile).
- `runecode-auditd validate-signer-evidence` validates signer scope/purpose and isolate binding evidence for isolate-attributed events.
- `runecode-secretsd validate-sign-request` validates sign-request preconditions (purpose/scope and closed posture enums) before signing.

These hooks are not final runtime wiring for all features, but they are authoritative trusted-domain validation surfaces used to fail closed and keep upcoming `CHG-2026-003/006/008/009/031/033` integration deterministic.

## CI Guardrail

The runner includes a mechanical boundary check (`npm run boundary-check`) that fails CI if:
- Runner source files anywhere under `runner/` (excluding `node_modules/`, `dist/`, `.git/`, `.turbo/`, `coverage/`, and guardrail test/tooling files) import from `../../internal/*` or `../../cmd/*`
- Runner source files reference repo-root trusted or restricted paths outside allowed `protocol/` access (`protocol/schemas/` and `protocol/fixtures/` only), including `cmd/`, `internal/`, and `tools/`
- Runner source files use absolute path references unless they resolve under allowed protocol roots (`protocol/schemas/` and `protocol/fixtures/`)
- No runner source files are found (fail closed)

This check is intentionally best-effort static analysis, not the final security boundary.
Authoritative runtime enforcement is provided by broker auth/schema validation, policy decisions, and isolation backends captured in the corresponding RuneContext changes.
