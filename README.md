# RuneCode — Security-first AI coding: isolated execution, signed, auditable

[![CI](https://github.com/runecode-ai/runecode/actions/workflows/ci.yml/badge.svg)](https://github.com/runecode-ai/runecode/actions/workflows/ci.yml)
[![Status: alpha.4 release](https://img.shields.io/badge/status-alpha.4%20release-orange)](runecontext/project/roadmap.md)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

RuneCode is a security-first agentic automation platform for software engineering.
It treats isolation and cryptographic provenance as co-equal pillars: work runs in tightly scoped isolates with deny-by-default capabilities, explicit artifact-based data movement, and a tamper-evident audit trail.

## Status

The latest published release is `v0.1.0-alpha.4`, and the repository mainline already includes additional alpha.5 work in progress.
RuneCode remains pre-production: the signed, tag-driven release pipeline exists, but the shipped Go binaries are still scaffold-heavy and not feature-complete.

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
- Trusted control-plane services compile immutable `RunPlan` contracts from workflow and process definitions; the untrusted runner consumes that plan as a thin kernel and reports progress through typed broker APIs rather than inventing its own planning truth

Details (diagram, allowed interfaces, prohibited bypasses, and CI guardrail): `docs/trust-boundaries.md`.

## Repository Layout

- `cmd/` — trusted Go binaries (launcher, broker, secretsd, auditd, TUI)
- `formal/` — checked-in formal specifications and model-checking assets
- `internal/` — trusted Go libraries
- `nix/` — canonical release metadata, build definitions, and flake checks
- `runner/` — untrusted TS/Node workflow runner package
- `protocol/` — authoritative schema bundle, shared registries, and cross-language fixtures for trusted/untrusted messages
- `tools/` — repo-local helper tools for deterministic checks and fixes
- `docs/` — trust-boundary contract and supporting design docs
- `runecontext/` — canonical project context, standards, changes, specs, decisions, and bundles

## Install

The official release channel is GitHub Releases.

- Canonical unsigned release artifacts come from `nix build --no-link .#release-artifacts`
- Published release assets are signed and attested in GitHub Actions
- Supported targets: Linux (`amd64`, `arm64`), macOS (`amd64`, `arm64`), Windows (`amd64`, `arm64`)
- Requires `gh` and `cosign`

Quick verified install for Linux and macOS:

```bash
set -euo pipefail

REPO="runecode-ai/runecode"
# Newest published release, including prereleases during pre-alpha.
# Ordered by creation date; assumes no out-of-order backport releases.
VERSION="$(gh release list --repo "$REPO" --exclude-drafts --limit 1 --json tagName --jq '.[0].tagName')"

if [ -z "$VERSION" ]; then
  printf 'no published release found for %s\n' "$REPO" >&2
  exit 1
fi

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) printf 'unsupported architecture: %s\n' "$ARCH" >&2; exit 1 ;;
esac

case "$OS" in
  linux|darwin) ;;
  *) printf 'unsupported operating system: %s\n' "$OS" >&2; exit 1 ;;
esac

ARCHIVE="runecode_${VERSION}_${OS}_${ARCH}.tar.gz"
WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

cd "$WORKDIR"

gh release download "$VERSION" --repo "$REPO" \
  --pattern "$ARCHIVE" \
  --pattern "$ARCHIVE.sig" \
  --pattern "$ARCHIVE.pem" \
  --pattern "SHA256SUMS" \
  --pattern "SHA256SUMS.sig" \
  --pattern "SHA256SUMS.pem"

cosign verify-blob \
  --certificate-identity "https://github.com/${REPO}/.github/workflows/release.yml@refs/tags/${VERSION}" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  --signature "SHA256SUMS.sig" \
  --certificate "SHA256SUMS.pem" \
  "SHA256SUMS"

cosign verify-blob \
  --certificate-identity "https://github.com/${REPO}/.github/workflows/release.yml@refs/tags/${VERSION}" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  --signature "${ARCHIVE}.sig" \
  --certificate "${ARCHIVE}.pem" \
  "$ARCHIVE"

if command -v sha256sum >/dev/null 2>&1; then
  grep -F "  ${ARCHIVE}" SHA256SUMS | sha256sum -c -
else
  grep -F "  ${ARCHIVE}" SHA256SUMS | shasum -a 256 -c -
fi

mkdir unpack
tar -xzf "$ARCHIVE" -C unpack

install -d "$HOME/.local/bin"
install -m 0755 "unpack/runecode_${VERSION}_${OS}_${ARCH}"/bin/runecode-* "$HOME/.local/bin/"
```

This quick path verifies signed checksums and the signed archive before install. For Windows steps and full provenance verification with `gh attestation verify`, see `docs/install-verify.md`.

## Implemented in this repo today:
- A protocol/schema bundle in `protocol/schemas/` with an authoritative manifest at `protocol/schemas/manifest.json`
- Shared JSON Schema object families for manifests, identities, approvals, artifacts/provenance, audit events/receipts, audit segment files/seals, audit verification reports, policy decisions, model request/response/streaming, broker local API request/response/read-model/stream families, detached signature envelopes, and shared errors
- Shared machine-consumed code registries for `error.code`, `policy_reason_code`, `approval_trigger_code`, `audit_event_type`, `audit_receipt_kind`, and `audit_verification_reason_code`
- Shared fixtures in `protocol/fixtures/` validated in both Go and Node, including schema, stream-sequence, runtime-invariant, and canonicalization/hash cases
- CI guardrails for runner trust-boundary access and protocol parity
- Workflow/process planning schemas and fixtures, plus a trusted Go `RunPlan` compiler that merges executor bindings and deterministic gate definitions into one immutable execution contract
- Deterministic gate contracts and reporting families for gate planning, runner checkpoint/result reporting, gate checkpoint/result reporting, and gate evidence persistence
- A thin untrusted runner kernel foundation that loads broker-compiled `RunPlan` data from the shared schema bundle, persists plan-bound journal/snapshot durable state, replays approval waits and recovery state fail closed, schedules plan entries, and emits typed reports back to the broker
- A narrow internal runner runtime seam for local checkpoint, wait, and resume mechanics without making runner-local state, third-party runtimes, or framework checkpoints authoritative
- MVP artifact data classes and an `ArtifactPolicy` schema family anchoring flow-matrix, approval-promotion, quota, and retention/GC controls
- A trusted local artifact store with immutable hash-addressed artifact persistence, broker-facing flow checks, quota enforcement, retention/GC, backup/restore, approval records, persisted policy decisions, and audit event recording for artifact and approval actions
- Approval promotion, resolution, and revocation flows for `unapproved_file_excerpts` and `approved_file_excerpts`, including signed request/decision verification bound to canonical request bytes, promoted inputs, verifier owner identity, and durable policy-decision linkage
- Store-layer atomic persistence for canonical approval records plus runner-advisory approval mirrors, with rollback that restores durable runner journal/snapshot state consistently on failure
- A trusted local audit ledger with append/seal persistence, segment recovery, digest-addressed sidecar evidence, explicit audit anchoring over signed segment seals, readiness evaluation, audit verification reports, and broker/TUI-facing audit verification, record inspection, anchoring, and readiness surfaces
- A broker local API with fail-closed local auth, schema-validated typed operations for runs, approvals, artifacts, audit timeline and record inspection, audit anchor presence and action flows, readiness, version info, and backend posture, plus uniform log and artifact read streaming semantics
- A trusted full-screen TUI workbench that launches in alt-screen mode, keeps sidebar/main/inspector composition in the shell, supports multi-session workspace navigation and quick switching, exposes an object-aware palette plus Action Center, derives live activity and sync health from typed watch families, preserves ordinary terminal selection alongside explicit copy actions, and persists layout/theme/session convenience state locally without promoting it to control-plane authority
- A trusted local secrets daemon with durable secret import plus short-lived lease issue/renew/revoke/retrieve flows, fail-closed recovery, and secret-safe onboarding that avoids CLI-arg or environment-variable transport
- Broker run read models that keep authoritative trusted state distinct from runner-advisory projection, including durable approval-wait, lifecycle, checkpoint, result, and attempt hints
- Broker-projected subsystem readiness for secrets and model-gateway posture, plus model-gateway runtime enforcement for allowlisted destinations, canonical request binding, quota context, and audit-bound egress decisions
- Broker-projected backend posture state and approval-mediated instance posture changes, including the active launcher `instance_id`, selected `backend_kind`, reduced-assurance cues, per-backend availability, and policy/approval linkage for posture changes
- A trusted launcher daemon/service plus a Linux-first microVM/QEMU/KVM MVP vertical slice and a Linux-only explicit-opt-in container backend slice for offline `workspace` launches, including a deterministic `runecode-launcher serve --hello-world` path for end-to-end launcher->broker runtime reporting
- Durable launcher runtime evidence persistence and broker-derived authoritative runtime projection for `backend_kind`, `isolation_assurance_level`, `provisioning_posture`, lifecycle, and terminal state
- Broker-owned runtime audit emission for `isolate_session_started` and `isolate_session_bound`, with reference-heavy payloads bound to persisted launcher evidence digests
- Checked-in bounded TLA+ security-kernel artifacts plus deterministic TLC model-checking wired into `just model-check` and `just ci`

Still incremental / not implemented end-to-end yet:
- Secrets lifecycle foundations and broker-projected secrets/model-gateway posture now exist, but secure-storage posture projection and downstream provider/auth integrations remain incremental
- The primary secure path remains Linux-first microVM/QEMU/KVM MVP. Container backend support now exists as a Linux-only explicit-opt-in reduced-assurance MVP for offline `workspace` launches; broader role coverage, non-Linux runtime paths, and further hardening/verification remain future work
- The broker and artifact store now implement local runtime behavior, but the overall system is still early alpha and not production-ready

- Roadmap: `runecontext/project/roadmap.md`

## Protocol Foundation

`protocol/` is the current implemented foundation for cross-boundary contracts.

- Bundle ID: `runecode.protocol.v0`
- Source of truth: `protocol/schemas/manifest.json`
- Schema draft: JSON Schema `2020-12`
- Canonicalization profile: RFC 8785 JCS
- Current trusted wrapper root policy: top-level JSON object or array only
- Top-level posture: exact `schema_id` + `schema_version`; unknown fields and unknown schema versions fail closed
- Shared fixtures: `protocol/fixtures/manifest.json`
- Cross-language verification: Go tests in `internal/protocolschema/` and Node tests in `runner/scripts/protocol-fixtures.test.js`

Current MVP object families cover:
- manifests: `RoleManifest`, `CapabilityManifest`
- identity and content addressing: `PrincipalIdentity`, `Digest`, `ArtifactReference`, `ArtifactPolicy`, `ProvenanceReceipt`
- secrets custody and posture: `SecretLease`, `SecretStoragePosture`
- audit, approvals, and policy: `AuditEvent`, `AuditReceipt`, `AuditSegmentFile`, `AuditSegmentSeal`, `AuditVerificationReport`, `ApprovalRequest`, `ApprovalDecision`, `ApprovalBackendPostureSelection`, `PolicyDecision`, `PolicyRuleSet`, `PolicyAllowlist`
- workflow planning and deterministic gates: `WorkflowDefinition`, `ProcessDefinition`, `RunPlan`, `GateDefinition`, `GateContract`, `RunnerCheckpointReport`, `RunnerResultReport`, `GateCheckpointReport`, `GateResultReport`, `GateEvidence`
- stage summaries and sign-off payloads: `StageSummary`, `RunStageSummary`, `ActionPayloadStageSummarySignOff`
- runtime evidence and session lifecycle payloads: `RuntimeImageDescriptor`, `IsolateSessionStartedPayload`, `IsolateSessionBoundPayload`
- policy actions and destinations: `ActionRequest`, `ActionPayloadArtifactRead`, `ActionPayloadPromotion`, `ActionPayloadGatewayEgress`, `ActionPayloadSecretAccess`, `ActionPayloadWorkspaceWrite`, `ActionPayloadExecutorRun`, `ActionPayloadBackendPostureChange`, `ActionPayloadGateOverride`, `ActionPayloadStageSummarySignOff`, `DestinationDescriptor`, `GatewayScopeRule`
- model traffic: `LLMRequest`, `LLMResponse`, `LLMStreamEvent`, `LLMInvokeRequest`, `LLMInvokeResponse`, `LLMStreamRequest`, `LLMStreamEnvelope`
- broker local API requests/responses: `RunListRequest`, `RunGetRequest`, `ApprovalListRequest`, `ApprovalGetRequest`, `ApprovalResolveRequest`, `BackendPostureGetRequest`, `BackendPostureChangeRequest`, `ArtifactListRequest`, `ArtifactHeadRequest`, `ArtifactReadRequest`, `AuditTimelineRequest`, `AuditRecordGetRequest`, `AuditVerificationGetRequest`, `AuditAnchorPresenceGetRequest`, `AuditAnchorPreflightGetRequest`, `AuditAnchorPreflightGetResponse`, `AuditAnchorSegmentRequest`, `AuditFinalizeVerifyRequest`, `AuditFinalizeVerifyResponse`, `ReadinessGetRequest`, `VersionInfoGetRequest`
- broker local API read models: `RunSummary`, `RunDetail`, `RunStageSummary`, `RunRoleSummary`, `RunCoordinationSummary`, `ApprovalSummary`, `ApprovalBoundScope`, `BackendPostureState`, `BackendPostureAvailability`, `ArtifactSummary`, `BrokerReadiness`, `BrokerVersionInfo`
- broker local API streams and error envelopes: `LogStreamEvent`, `ArtifactStreamEvent`, `BrokerErrorResponse`
- wrappers and shared errors: `SignedObjectEnvelope`, `Error`

## Development

Canonical local workflow uses Nix + `just` (Nix `>= 2.18`):

```sh
nix develop -c just ci
```

Canonical release-builder commands:

```sh
nix build --no-link .#release-artifacts
nix eval --raw .#lib.release.tag
```

Common commands:

```sh
just fmt
just lint
just model-check
just test
just ci
```

Useful protocol-specific checks:

```sh
go test ./internal/brokerapi
go test ./internal/protocolschema
cd runner && node --test scripts/protocol-fixtures.test.js
cd runner && npm test
cd runner && npm run boundary-check
```

These checks are also covered by `just ci`.

Formal model checking entrypoint:

```sh
just model-check
```

Optional: enable automatic dev-shell entry with `direnv` + `nix-direnv`:

```sh
direnv allow
```

Non-Nix fallback (e.g., Windows): install Go 1.25.x, Node `>=22.22.1 <25` with npm, `just`, and either a `tlc` binary or Java 17+ plus `tla2tools.jar` (or set `TLA2TOOLS_JAR`), then run:

```sh
just ci
```

## Components

The Go binaries currently shipped by the release pipeline remain pre-production and intentionally do not expose the full production system surface.

Alongside that still-incremental surface, the repository already includes working foundations with:
- manifest-verified schemas and registries
- cross-language fixture validation
- canonicalization/hash golden tests
- runner trust-boundary static checks
- a trusted full-screen `runecode-tui` workbench with dashboard/chat/runs/approvals/Action Center/artifacts/audit/status routes, shell-owned pane composition, session quick switching, typed watch-backed live activity, selection-mode copy ergonomics, and local-only layout/theme persistence
- a trusted local artifact store and broker CLI for artifact put/get/head/list, flow checks, excerpt promotion and revocation, run-status updates, GC, and backup/restore
- a trusted local audit ledger plus broker/auditd CLI surfaces for audit readiness, audit verification inspection, audit record inspection, and explicit audit anchoring over signed segment seals
- a broker local IPC API and CLI read/action surfaces for run list/detail, approval list/detail/resolve, policy-backed artifact reads, audit timeline/record inspection, audit anchoring presence/action, audit verification/readiness, version inspection, structured log streaming, and broker-projected backend posture get/change operations
- a trusted local secrets daemon CLI for secret import and short-lived lease issue/renew/revoke/retrieve flows without passing secret values through CLI args or environment variables
- broker-projected secrets and model-gateway readiness surfaces plus model-gateway runtime enforcement for allowlisted destinations, canonical request binding, quota admission/stream checks, and audit-backed egress decisions
- a trusted launcher service with `serve`, `--once`, Linux-first `--hello-world` operator paths, and a Linux-only explicit-opt-in container backend posture for offline `workspace` launches
- launcher-produced runtime evidence persisted durably and projected into broker `RunSummary` / `RunDetail` authoritative state
- broker-emitted runtime lifecycle audit events referencing persisted launcher evidence rather than transient launcher-local state

You can inspect their help output:

```sh
go run ./cmd/runecode-tui --help
go run ./cmd/runecode-launcher --help
go run ./cmd/runecode-broker --help
go run ./cmd/runecode-secretsd --help
go run ./cmd/runecode-auditd --help
```

`runecode-tui` expects a local broker API listener in another terminal, typically `runecode-broker serve-local`, and also supports `--runtime-dir` / `--socket-name` for isolated local-dev IPC overrides.

The broker help surface currently includes local API and operator commands such as:

```sh
go run ./cmd/runecode-broker serve-local --help
go run ./cmd/runecode-broker run-list --help
go run ./cmd/runecode-broker run-get --help
go run ./cmd/runecode-broker approval-list --help
go run ./cmd/runecode-broker approval-get --help
go run ./cmd/runecode-broker promote-excerpt --help
go run ./cmd/runecode-broker revoke-approved-excerpt --help
go run ./cmd/runecode-broker audit-verification --help
go run ./cmd/runecode-broker audit-record-get --help
go run ./cmd/runecode-broker audit-anchor-segment --help
go run ./cmd/runecode-broker audit-readiness --help
go run ./cmd/runecode-broker version-info --help
go run ./cmd/runecode-broker stream-logs --help
```

## Docs

- Install and verify releases: `docs/install-verify.md`
- Maintainer release process: `docs/release-process.md`
- Nix release/dev layout: `nix/README.md`
- Mission: `runecontext/project/mission.md`
- Roadmap: `runecontext/project/roadmap.md`
- Tech stack: `runecontext/project/tech-stack.md`
- Trust boundaries: `docs/trust-boundaries.md`
- Protocol schemas: `protocol/schemas/README.md`
- Protocol/schema spec: `runecontext/specs/protocol-schema-bundle-v0.md`
- Formal security-kernel model: `formal/tla/security-kernel/README.md`
- Agent and AI contributor guidance: `AGENTS.md`

## Uninstall

Remove the installed binaries for the path used in the install docs.

Linux and macOS:

```sh
rm -f \
  "$HOME/.local/bin/runecode-auditd" \
  "$HOME/.local/bin/runecode-broker" \
  "$HOME/.local/bin/runecode-launcher" \
  "$HOME/.local/bin/runecode-secretsd" \
  "$HOME/.local/bin/runecode-tui"
```

Windows PowerShell:

```powershell
$InstallDir = Join-Path $env:LOCALAPPDATA "Programs\RuneCode\bin"
Remove-Item `
  "$InstallDir\runecode-auditd.exe", `
  "$InstallDir\runecode-broker.exe", `
  "$InstallDir\runecode-launcher.exe", `
  "$InstallDir\runecode-secretsd.exe", `
  "$InstallDir\runecode-tui.exe" `
  -Force -ErrorAction SilentlyContinue
```

## Contributing

See `CONTRIBUTING.md`. DCO sign-off is required (`git commit -s`).

## Security

Please do not open public issues for security vulnerabilities. See [SECURITY.md](SECURITY.md).

## License

Apache-2.0. See `LICENSE` and `NOTICE`.
