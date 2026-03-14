# RuneCode — Security-first AI coding: isolated execution, signed, auditable

[![CI](https://github.com/runecode-ai/runecode/actions/workflows/ci.yml/badge.svg)](https://github.com/runecode-ai/runecode/actions/workflows/ci.yml)
![Status: pre-alpha](https://img.shields.io/badge/status-pre--alpha-orange)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

RuneCode is a security-first agentic automation platform for software engineering.
It treats isolation and cryptographic provenance as co-equal pillars: work runs in tightly scoped isolates with deny-by-default capabilities, explicit artifact-based data movement, and a tamper-evident audit trail.

## Status

RuneCode is pre-alpha and not production-ready.

Implemented in this repo today:
- A protocol/schema bundle in `protocol/schemas/` with an authoritative manifest at `protocol/schemas/manifest.json`
- Shared JSON Schema object families for manifests, identities, approvals, artifacts/provenance, audit events/receipts, policy decisions, model request/response/streaming, detached signature envelopes, and shared errors
- Shared machine-consumed code registries for `error.code`, `policy_reason_code`, `approval_trigger_code`, and `audit_event_type`
- Shared fixtures in `protocol/fixtures/` validated in both Go and Node, including schema, stream-sequence, runtime-invariant, and canonicalization/hash cases
- CI guardrails for runner trust-boundary access and protocol parity

Still incremental / not implemented end-to-end yet:
- Broker runtime, policy evaluation, secrets handling, audit persistence/verification, and isolation backends remain scaffolded or are implemented in later specs

- Roadmap: `agent-os/product/roadmap.md`

## Protocol Foundation

`protocol/` is the current implemented foundation for cross-boundary contracts.

- Bundle ID: `runecode.protocol.v0`
- Source of truth: `protocol/schemas/manifest.json`
- Schema draft: JSON Schema `2020-12`
- Canonicalization profile: RFC 8785 JCS
- Top-level posture: exact `schema_id` + `schema_version`; unknown fields and unknown schema versions fail closed
- Shared fixtures: `protocol/fixtures/manifest.json`
- Cross-language verification: Go tests in `internal/protocolschema/` and Node tests in `runner/scripts/protocol-fixtures.test.js`

Current MVP object families cover:
- manifests: `RoleManifest`, `CapabilityManifest`
- identity and content addressing: `PrincipalIdentity`, `Digest`, `ArtifactReference`, `ProvenanceReceipt`
- audit and approvals: `AuditEvent`, `AuditReceipt`, `ApprovalRequest`, `ApprovalDecision`, `PolicyDecision`
- model traffic: `LLMRequest`, `LLMResponse`, `LLMStreamEvent`
- wrappers and shared errors: `SignedObjectEnvelope`, `Error`

## Why RuneCode

- **Isolation is the boundary:** risky work runs in tightly scoped isolates; the workflow runner is treated as untrusted.
- **Deny-by-default posture:** capabilities (egress, secrets, workspace writes) are explicit and intended to be policy-controlled.
- **Signed, auditable evidence:** the goal is a tamper-evident trail for actions, decisions, and artifacts (diffs/logs/results).
- **Explicit data movement:** handoffs are intended to happen via hash-addressed artifacts, not implicit shared state.

## Threat model (micro)

RuneCode is built around a pessimistic assumption: any single AI/agent component (including the workflow runner) can be compromised or behave maliciously.
The architecture aims to reduce blast radius and preserve forensics by:

- Separating trusted control-plane components from an untrusted workflow runner.
- Preventing any one component from having broad combined powers (network + workspace + long-lived secrets).
- Making cross-boundary interfaces schema-driven and auditable.

Design inspiration includes compartmentalization models (e.g., QubesOS), applied to agentic workflows.

## Security Model (High Level)

RuneCode is designed around two local trust domains:

- **Trusted domain:** Go control plane daemons + Go TUI client
- **Untrusted domain:** TS/Node workflow runner

Key invariants (design targets; enforcement is implemented incrementally):

- Deny-by-default capabilities; explicit opt-ins for higher-risk posture changes
- No single component combines public network egress + workspace access (especially RW) + long-lived secrets
- Cross-boundary communication is brokered and schema-validated (no ad-hoc JSON)

Details (diagram, allowed interfaces, prohibited bypasses, and CI guardrail): `docs/trust-boundaries.md`.

## Repository Layout

- `cmd/` — trusted Go binaries (launcher, broker, secretsd, auditd, TUI)
- `internal/` — trusted Go libraries
- `runner/` — untrusted TS/Node workflow runner package
- `protocol/` — authoritative schema bundle, shared registries, and cross-language fixtures for trusted/untrusted messages
- `tools/` — repo-local helper tools for deterministic checks and fixes
- `docs/` — trust-boundary contract and supporting design docs
- `agent-os/` — product/spec/standards documents (git-native system of record)

## Development

Canonical local workflow uses Nix + `just` (Nix `>= 2.18`):

```sh
nix develop -c just ci
```

Common commands:

```sh
just fmt
just lint
just test
just ci
```

Useful protocol-specific checks:

```sh
go test ./internal/protocolschema
cd runner && node --test scripts/protocol-fixtures.test.js
cd runner && npm run boundary-check
```

These checks are also covered by `just ci`.

Optional: enable automatic dev-shell entry with `direnv` + `nix-direnv`:

```sh
direnv allow
```

Non-Nix fallback (e.g., Windows): install Go 1.25.x, Node `>=22.22.1 <25` with npm, and `just`, then run:

```sh
just ci
```

## Components

The Go binaries in `cmd/` are still scaffolded and intentionally do not start network listeners.

Alongside those stubs, the repository already includes a working protocol/schema foundation with:
- manifest-verified schemas and registries
- cross-language fixture validation
- canonicalization/hash golden tests
- runner trust-boundary static checks

You can inspect their help output:

```sh
go run ./cmd/runecode-tui --help
go run ./cmd/runecode-launcher --help
go run ./cmd/runecode-broker --help
go run ./cmd/runecode-secretsd --help
go run ./cmd/runecode-auditd --help
```

## Docs

- Mission: `agent-os/product/mission.md`
- Roadmap: `agent-os/product/roadmap.md`
- Trust boundaries: `docs/trust-boundaries.md`
- Protocol schemas: `protocol/schemas/README.md`
- Protocol/schema spec: `agent-os/specs/2026-03-08-1039-protocol-schemas-v0/`
- Agent and AI contributor guidance: `AGENTS.md`

## Contributing

See `CONTRIBUTING.md`. DCO sign-off is required (`git commit -s`).

## Security

Please do not open public issues for security vulnerabilities. See [SECURITY.md](SECURITY.md).

## License

Apache-2.0. See `LICENSE` and `NOTICE`.
