# RuneCode — Security-first AI coding: isolated execution, signed, auditable

[![CI](https://github.com/runecode-ai/runecode/actions/workflows/ci.yml/badge.svg)](https://github.com/runecode-ai/runecode/actions/workflows/ci.yml)
[![Status: alpha.9 in progress](https://img.shields.io/badge/status-alpha.9%20in%20progress-orange)](runecontext/project/roadmap.md)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

RuneCode is a security-first agentic automation platform for software engineering.
It treats isolation and cryptographic provenance as co-equal pillars: work runs in tightly scoped isolates with deny-by-default capabilities, explicit artifact-based data movement, and a tamper-evident audit trail.

## Status

The latest published release is `v0.1.0-alpha.7`, and the repository mainline already includes `v0.1.0-alpha.9` work in progress.
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
install -m 0755 "unpack/runecode_${VERSION}_${OS}_${ARCH}"/bin/runecode* "$HOME/.local/bin/"
```

This quick path verifies signed checksums and the signed archive before install. For Windows steps and full provenance verification with `gh attestation verify`, see `docs/install-verify.md`.

## Implemented in this repo today:
- A protocol/schema bundle in `protocol/schemas/` with an authoritative manifest at `protocol/schemas/manifest.json`
- Shared JSON Schema object families for manifests, identities, approvals, artifacts/provenance, audit events/receipts, audit segment files/seals, audit verification reports, policy decisions, model request/response/streaming, provider profile/auth-material/setup/validation/credential-lease families, broker local API request/response/read-model/stream families, detached signature envelopes, and shared errors
- Shared machine-consumed code registries for `error.code`, `policy_reason_code`, `approval_trigger_code`, `audit_event_type`, `audit_receipt_kind`, and `audit_verification_reason_code`
- Shared fixtures in `protocol/fixtures/` validated in both Go and Node, including schema, stream-sequence, runtime-invariant, and canonicalization/hash cases
- CI guardrails for runner trust-boundary access and protocol parity
- Workflow/process planning schemas and fixtures, plus trusted Go compilation, persistence, and selection of immutable `RunPlan` authority that binds reviewed workflow selection, authoritative process DAG shape, executor bindings, deterministic gate definitions, dependency edges, and compiled runtime entries into one broker-owned execution contract
- A first-party RuneContext workflow pack with broker-owned routing for `change_draft`, `spec_draft`, `draft_promote_apply`, and `approved_change_implementation`, where drafting remains artifact-first, approved implementation binds one exact reviewed `implementation_input_set`, and shared-workspace execution stays at one active mutation-bearing run per authoritative repository root in `v0`
- Deterministic gate contracts and reporting families for gate planning, runner checkpoint/result reporting, gate checkpoint/result reporting, and gate evidence persistence, with stored evidence bound back to the active plan, workflow/process definition hashes, policy context hash, and validated project context digest
- A thin untrusted runner kernel foundation that loads broker-compiled `RunPlan` data from the shared schema bundle, persists plan-bound journal/snapshot durable state, replays approval waits and recovery state fail closed, schedules plan entries, and emits typed reports back to the broker
- A narrow internal runner runtime seam for local checkpoint, wait, and resume mechanics without making runner-local state, third-party runtimes, or framework checkpoints authoritative
- MVP artifact data classes and an `ArtifactPolicy` schema family anchoring flow-matrix, approval-promotion, quota, and retention/GC controls
- A trusted local artifact store with immutable hash-addressed artifact persistence, broker-facing flow checks, quota enforcement, retention/GC, self-contained signed backup bundle export and fail-closed restore, approval records, persisted policy decisions, and audit event recording for artifact and approval actions
- Approval promotion, resolution, and revocation flows for `unapproved_file_excerpts` and `approved_file_excerpts`, including signed request/decision verification bound to canonical request bytes, promoted inputs, verifier owner identity, and durable policy-decision linkage
- Store-layer atomic persistence for canonical approval records plus runner-advisory approval mirrors, with rollback that restores durable runner journal/snapshot state consistently on failure
- A trusted local audit ledger with append/seal persistence, segment recovery, digest-addressed sidecar evidence, explicit audit anchoring over signed segment seals, external-anchor evidence and sidecar persistence, readiness evaluation, audit verification reports, rebuildable record-inclusion lookup, evidence-preservation snapshots, verifier-friendly bundle manifests, streaming evidence-bundle export, offline bundle verification, and broker/TUI-facing audit verification, record inspection, anchoring, evidence review, and readiness surfaces
- A broker local API with fail-closed local auth, schema-validated typed operations for runs, sessions, approvals, artifacts, audit timeline and record inspection, audit record inclusion lookup, audit evidence snapshot and retention review, audit evidence-bundle manifest/export/offline-verify flows, audit anchor presence and action flows, external-anchor mutation prepare/get/issue-execute-lease/execute flows, readiness, version info, and backend posture, including transcript append, session execution trigger, session list/detail, session watch, and turn-execution watch semantics plus uniform log and artifact read streaming semantics
- A broker-owned project-substrate lifecycle for canonical RuneContext repositories, including discovery and validation of repo-root `runecontext.yaml`, canonical `runecontext/` anchors, and `runecontext/assurance/baseline.yaml`, runtime-derived compatibility posture evaluation from local `runectx metadata` when available (with release fallback), read-only adoption of existing compatible substrate, explicit init and upgrade preview/apply flows, preview-digest-bound upgrade apply, auditable apply results, and validated snapshot digests for later planning, audit, and verification binding
- A canonical top-level `runecode` product command that resolves authoritative repo scope, ensures the repo-scoped local broker lifecycle exists, and exposes attach/start/status/stop/restart flows without making local bootstrap artifacts authoritative
- A trusted repo-scoped local bootstrap/resolver layer that derives one product instance per authoritative repository root, recovers stale pid/socket artifacts safely, and validates that a reachable broker matches the expected repo-scoped product instance before attach
- A broker-owned typed product lifecycle posture surface that keeps attachability, lifecycle generation, repo-scoped product identity, degraded/blocked reason codes, and normal-operation permission explicit instead of inferring lifecycle truth from readiness, version, or socket reachability
- A trusted full-screen TUI workbench that launches in alt-screen mode, keeps sidebar/main/inspector composition in the shell, supports multi-session workspace navigation and quick switching, exposes an object-aware palette plus Action Center, uses a configurable `space`-by-default leader system plus bottom-left `:` command mode backed by one authoritative shell action graph, keeps a visible beginner-friendly quit action and double-press `ctrl+c` emergency escape hatch without colliding with ordinary typing or secret entry, derives live activity and sync health from typed watch families, keeps chat execution progress derived from broker-owned session execution trigger and turn-execution watch state rather than chat-local truth, preserves ordinary terminal selection alongside explicit copy actions, and persists layout/theme/session convenience state locally without promoting it to control-plane authority
- Durable broker-owned session summaries and details that keep session object lifecycle distinct from projected `work_posture`, transcript lifecycle, and client attachment state, while exposing `current_turn_execution`, `latest_turn_execution`, and `pending_turn_executions` so sessions and linked runs remain inspectable across TUI close and later reconnect
- A trusted local secrets daemon with durable secret import plus short-lived lease issue/renew/revoke/retrieve flows, fail-closed recovery, and secret-safe onboarding that avoids CLI-arg or environment-variable transport
- A shared provider substrate with durable broker-owned provider profiles, explicit auth-material separation, stable provider-profile identity across credential rotation and validation retries, and broker-projected readiness plus compatibility posture
- Broker-owned direct-credential provider setup flows with typed setup sessions, one-time secret-ingress handles, validation lifecycle surfaces, and provider credential lease issuance without carrying raw secret values in ordinary typed broker request or response bodies
- Direct-credential model access for OpenAI-compatible Chat Completions and Anthropic-compatible Messages beneath the canonical typed `LLMRequest`, `LLMResponse`, and `LLMStreamEvent` boundary, with manual allowlisted model IDs remaining canonical
- Broker run read models that keep authoritative trusted state distinct from runner-advisory projection, including durable approval-wait, lifecycle, checkpoint, result, and attempt hints
- Broker-projected subsystem readiness for secrets and model-gateway posture, plus model-gateway runtime enforcement for allowlisted destinations, canonical request binding, quota context, and audit-bound egress decisions
- Broker-projected backend posture state and approval-mediated instance posture changes, including the active launcher `instance_id`, selected `backend_kind`, reduced-assurance cues, per-backend availability, and policy/approval linkage for posture changes
- A trusted launcher daemon/service plus a Linux-first microVM/QEMU/KVM MVP vertical slice and a Linux-only explicit-opt-in container backend slice for offline `workspace` launches, including a deterministic `runecode-launcher serve --hello-world` path for end-to-end launcher->broker runtime reporting
- Signed runtime-image and runtime-toolchain identity contracts, typed verifier-authority state, trusted admission into a launcher-private verified runtime cache, and fail-closed launch from verified local assets rather than mutable host paths or ad hoc launch-time synthesis
- Durable launcher runtime evidence persistence and broker-derived authoritative runtime projection for `backend_kind`, `isolation_assurance_level`, `provisioning_posture`, lifecycle, terminal state, and runtime attestation support or verification posture from persisted evidence rather than transient launcher state
- Broker-owned runtime audit emission for `runtime_launch_admission`, `runtime_launch_denied`, `isolate_session_started`, and `isolate_session_bound`, with reference-heavy payloads bound to persisted launcher evidence digests
- Checked-in bounded TLA+ security-kernel artifacts plus deterministic TLC model-checking wired into `just model-check` and `just ci`

Still incremental / not implemented end-to-end yet:
- Secure-storage posture projection and broader provider auth modes remain incremental, but direct-credential provider setup and execution now exist for OpenAI-compatible and Anthropic-compatible endpoints on the shared provider substrate
- The primary secure path now includes signed runtime-image and toolchain admission into a verified local cache for Linux-first launcher operation. Container backend support still exists as a Linux-only explicit-opt-in reduced-assurance MVP for offline `workspace` launches; broader role coverage, non-Linux runtime paths, and further hardening/verification remain future work
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
- provider substrate and setup lifecycle: `ProviderProfile`, `ProviderAuthMaterial`, `ProviderModelCatalogPosture`, `ProviderReadinessPosture`, `ProviderSetupSession`, `ProviderSetupSessionBeginRequest`, `ProviderSetupSessionBeginResponse`, `ProviderSetupSecretIngressPrepareRequest`, `ProviderSetupSecretIngressPrepareResponse`, `ProviderSetupSecretIngressSubmitRequest`, `ProviderSetupSecretIngressSubmitResponse`, `ProviderValidationBeginRequest`, `ProviderValidationBeginResponse`, `ProviderValidationCommitRequest`, `ProviderValidationCommitResponse`, `ProviderCredentialLeaseIssueRequest`, `ProviderCredentialLeaseIssueResponse`
- project substrate contract and lifecycle: `ProjectSubstrateContractState`, `ProjectSubstrateValidationSnapshot`, `ProjectSubstratePostureSummary`, `ProjectLifecycleOperatorDecisionPath`, `ProjectSubstrateAdoptionResult`, `ProjectSubstrateInitPreview`, `ProjectSubstrateInitApplyResult`, `ProjectSubstrateUpgradePreview`, `ProjectSubstrateUpgradeApplyResult`
- audit, approvals, and policy: `AuditEvent`, `AuditReceipt`, `AuditSegmentFile`, `AuditSegmentSeal`, `AuditVerificationReport`, `AuditRecordInclusion`, `AuditEvidenceSnapshot`, `AuditEvidenceRetentionReviewRequest`, `AuditEvidenceRetentionReviewResponse`, `AuditEvidenceBundleManifest`, `AuditEvidenceBundleExportRequest`, `AuditEvidenceBundleExportEvent`, `AuditEvidenceBundleOfflineVerification`, `ExternalAnchorEvidence`, `ApprovalRequest`, `ApprovalDecision`, `ApprovalBackendPostureSelection`, `PolicyDecision`, `PolicyRuleSet`, `PolicyAllowlist`, `TrustedContractImportRequest`
- workflow planning and deterministic gates: `WorkflowDefinition`, `ProcessDefinition`, `RunPlan`, `GateDefinition`, `GateContract`, `RunnerCheckpointReport`, `RunnerResultReport`, `GateCheckpointReport`, `GateResultReport`, `GateEvidence`
- stage summaries and sign-off payloads: `StageSummary`, `RunStageSummary`, `ActionPayloadStageSummarySignOff`
- runtime evidence and session lifecycle payloads: `RuntimeImageDescriptor`, `RuntimeImageSignedPayload`, `RuntimeToolchainDescriptor`, `RuntimeLaunchAdmissionPayload`, `RuntimeLaunchDeniedPayload`, `IsolateSessionStartedPayload`, `IsolateSessionBoundPayload`
- policy actions and destinations: `ActionRequest`, `ActionPayloadArtifactRead`, `ActionPayloadPromotion`, `ActionPayloadGatewayEgress`, `ActionPayloadSecretAccess`, `ActionPayloadWorkspaceWrite`, `ActionPayloadExecutorRun`, `ActionPayloadBackendPostureChange`, `ActionPayloadGateOverride`, `ActionPayloadStageSummarySignOff`, `DestinationDescriptor`, `GatewayScopeRule`
- model traffic: `LLMRequest`, `LLMResponse`, `LLMStreamEvent`, `LLMInvokeRequest`, `LLMInvokeResponse`, `LLMStreamRequest`, `LLMStreamEnvelope`
- broker local API requests/responses: `RunListRequest`, `RunGetRequest`, `SessionListRequest`, `SessionListResponse`, `SessionGetRequest`, `SessionGetResponse`, `SessionSendMessageRequest`, `SessionSendMessageResponse`, `SessionExecutionTriggerRequest`, `SessionExecutionTriggerResponse`, `SessionWatchRequest`, `SessionTurnExecutionWatchRequest`, `ApprovalListRequest`, `ApprovalGetRequest`, `ApprovalResolveRequest`, `BackendPostureGetRequest`, `BackendPostureChangeRequest`, `ArtifactListRequest`, `ArtifactHeadRequest`, `ArtifactReadRequest`, `AuditTimelineRequest`, `AuditRecordGetRequest`, `AuditRecordInclusionGetRequest`, `AuditRecordInclusionGetResponse`, `AuditEvidenceSnapshotGetRequest`, `AuditEvidenceSnapshotGetResponse`, `AuditEvidenceRetentionReviewRequest`, `AuditEvidenceRetentionReviewResponse`, `AuditEvidenceBundleManifestGetRequest`, `AuditEvidenceBundleManifestGetResponse`, `AuditEvidenceBundleOfflineVerifyRequest`, `AuditEvidenceBundleOfflineVerifyResponse`, `AuditVerificationGetRequest`, `AuditAnchorPresenceGetRequest`, `AuditAnchorPreflightGetRequest`, `AuditAnchorPreflightGetResponse`, `AuditAnchorSegmentRequest`, `AuditFinalizeVerifyRequest`, `AuditFinalizeVerifyResponse`, `ExternalAnchorMutationPrepareRequest`, `ExternalAnchorMutationPrepareResponse`, `ExternalAnchorMutationGetRequest`, `ExternalAnchorMutationGetResponse`, `ExternalAnchorMutationIssueExecuteLeaseRequest`, `ExternalAnchorMutationIssueExecuteLeaseResponse`, `ExternalAnchorMutationExecuteRequest`, `ExternalAnchorMutationExecuteResponse`, `ExternalAnchorMutationPreparedState`, `ProjectSubstrateGetRequest`, `ProjectSubstrateGetResponse`, `ProjectSubstratePostureGetRequest`, `ProjectSubstratePostureGetResponse`, `ProjectSubstrateAdoptRequest`, `ProjectSubstrateAdoptResponse`, `ProjectSubstrateInitPreviewRequest`, `ProjectSubstrateInitPreviewResponse`, `ProjectSubstrateInitApplyRequest`, `ProjectSubstrateInitApplyResponse`, `ProjectSubstrateUpgradePreviewRequest`, `ProjectSubstrateUpgradePreviewResponse`, `ProjectSubstrateUpgradeApplyRequest`, `ProjectSubstrateUpgradeApplyResponse`, `ProductLifecyclePostureGetRequest`, `ProductLifecyclePostureGetResponse`, `ReadinessGetRequest`, `VersionInfoGetRequest`
- broker local API read models: `RunSummary`, `RunDetail`, `RunStageSummary`, `RunRoleSummary`, `RunCoordinationSummary`, `ApprovalSummary`, `ApprovalBoundScope`, `BackendPostureState`, `BackendPostureAvailability`, `ArtifactSummary`, `BrokerReadiness`, `BrokerVersionInfo`, `BrokerProductLifecyclePosture`, `SessionSummary`, `SessionDetail`, `SessionTurnExecution`
- broker local API streams and error envelopes: `LogStreamEvent`, `ArtifactStreamEvent`, `SessionWatchEvent`, `SessionTurnExecutionWatchEvent`, `BrokerErrorResponse`
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

If `nix build .#release-artifacts` reports a replacement `vendorHash`, refresh it explicitly:

```sh
just refresh-release-vendor-hash
go run ./tools/releasebuilder refresh-vendor-hash
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

Non-Nix fallback (e.g., Windows): install Go 1.25.x, Node `>=22.22.1 <25` with npm, and `just`. For full formal-model parity outside Nix, also install either a `tlc` binary or Java 17+ plus `tla2tools.jar` (or set `TLA2TOOLS_JAR`). Then run:

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
- a trusted full-screen `runecode-tui` workbench with dashboard/chat/runs/approvals/Action Center/artifacts/audit/status/model-providers/git-setup/git-remote routes, shell-owned pane composition, session quick switching, a configurable `space`-default leader surface, bottom-left `:` command mode, one unified action graph for help/discovery/leader/command aliases, a visible quit action plus double-press `ctrl+c` emergency escape hatch, typed watch-backed live activity, chat execution progress derived from broker-owned session execution trigger plus turn-execution watch state, selection-mode copy ergonomics, broker-owned direct-credential provider setup with masked secret entry, and local-only layout/theme persistence
- the TUI status route now surfaces broker-owned project-substrate posture plus adopt, init, and upgrade actions without making the TUI itself authoritative
- a trusted local artifact store and broker CLI for artifact put/get/head/list, flow checks, excerpt promotion and revocation, run-status updates, GC, and self-contained signed backup bundle export or fail-closed restore that preserves runtime evidence, lifecycle state, and related durable attestation state
- a trusted local audit ledger plus broker/auditd CLI surfaces for audit readiness, audit verification inspection, audit record inspection, audit record inclusion lookup, evidence snapshots and retention review, verifier-friendly evidence-bundle manifest generation, streaming bundle export, offline bundle verification, explicit audit anchoring over signed segment seals, and external-anchor evidence plus sidecar persistence used by verification and projections
- a broker local IPC API and CLI read/action surfaces for run list/detail, session list/detail/message append/execution trigger/session watch, approval list/detail/resolve, policy-backed artifact reads, audit timeline/record inspection, audit record inclusion lookup, audit evidence snapshot/retention review/bundle manifest/bundle export/offline verify, audit anchoring presence/action, audit verification/readiness, external-anchor mutation prepare/get/issue-execute-lease/execute, trusted-contract import, version inspection, structured log streaming, broker-projected backend posture get/change operations, project-substrate posture/get/adopt/init/upgrade operations with preview-digest-bound upgrade apply, provider profile list/get, provider setup session and secret-ingress flows, provider validation lifecycle operations, provider credential lease issuance, and broker-owned session-turn-execution watch streams for in-flight execution state
- a trusted local secrets daemon CLI for secret import and short-lived lease issue/renew/revoke/retrieve flows without passing secret values through CLI args or environment variables
- broker-projected secrets and model-gateway readiness surfaces plus model-gateway runtime enforcement for allowlisted destinations, canonical request binding, quota admission/stream checks, and audit-backed egress decisions
- a trusted launcher service with `serve`, `--once`, Linux-first `--hello-world` operator paths, and a Linux-only explicit-opt-in container backend posture for offline `workspace` launches
- signed runtime-image and runtime-toolchain admission into a launcher-private verified cache, plus typed verifier-authority import and fail-closed launch from admitted local assets
- launcher-produced runtime evidence persisted durably and projected into broker `RunSummary` / `RunDetail` authoritative state, including authoritative runtime lifecycle and attestation-support or verification detail derived from persisted evidence rather than client-local inference
- broker-emitted runtime launch/session audit events referencing persisted launcher evidence rather than transient launcher-local state

You can inspect their help output:

```sh
go run ./cmd/runecode --help
go run ./cmd/runecode-tui --help
go run ./cmd/runecode-launcher --help
go run ./cmd/runecode-broker --help
go run ./cmd/runecode-secretsd --help
go run ./cmd/runecode-auditd --help
```

Normal user lifecycle flow now goes through `runecode`:

```sh
go run ./cmd/runecode
go run ./cmd/runecode status
go run ./cmd/runecode stop
```

Bare `runecode` is the canonical `attach` path: it resolves the authoritative repository root, ensures the repo-scoped local broker lifecycle exists, and opens the TUI against that broker-owned product instance. `runecode status` is intentionally non-starting and reports either broker-owned lifecycle plus project-substrate posture or only the bootstrap-local fact that no live product instance is reachable.

`runecode-tui` remains a low-level/dev entrypoint for attaching to an already running broker listener and still supports `--runtime-dir` / `--socket-name` for isolated local-dev IPC overrides. `runecode-broker` now also accepts those as broker-global options for live-IPC command surfaces such as session, approval, and external-anchor mutation commands.

Low-level broker help still covers plumbing/admin surfaces such as:

```sh
go run ./cmd/runecode-broker serve-local --help
go run ./cmd/runecode-broker run-list --help
go run ./cmd/runecode-broker run-get --help
go run ./cmd/runecode-broker session-list --help
go run ./cmd/runecode-broker session-get --help
go run ./cmd/runecode-broker session-send-message --help
go run ./cmd/runecode-broker session-execution-trigger --help
go run ./cmd/runecode-broker session-watch --help
go run ./cmd/runecode-broker approval-list --help
go run ./cmd/runecode-broker approval-get --help
go run ./cmd/runecode-broker import-trusted-contract --help
go run ./cmd/runecode-broker promote-excerpt --help
go run ./cmd/runecode-broker revoke-approved-excerpt --help
go run ./cmd/runecode-broker audit-verification --help
go run ./cmd/runecode-broker audit-record-get --help
go run ./cmd/runecode-broker audit-record-inclusion-get --help
go run ./cmd/runecode-broker audit-evidence-snapshot-get --help
go run ./cmd/runecode-broker audit-evidence-retention-review --help
go run ./cmd/runecode-broker audit-evidence-bundle-manifest-get --help
go run ./cmd/runecode-broker audit-evidence-bundle-export --help
go run ./cmd/runecode-broker audit-evidence-bundle-offline-verify --help
go run ./cmd/runecode-broker audit-anchor-segment --help
go run ./cmd/runecode-broker audit-readiness --help
go run ./cmd/runecode-broker external-anchor-mutation-prepare --help
go run ./cmd/runecode-broker external-anchor-mutation-get --help
go run ./cmd/runecode-broker external-anchor-mutation-issue-execute-lease --help
go run ./cmd/runecode-broker external-anchor-mutation-execute --help
go run ./cmd/runecode-broker provider-setup-direct --help
go run ./cmd/runecode-broker provider-profile-list --help
go run ./cmd/runecode-broker provider-profile-get --help
go run ./cmd/runecode-broker project-substrate-get --help
go run ./cmd/runecode-broker project-substrate-posture-get --help
go run ./cmd/runecode-broker project-substrate-adopt --help
go run ./cmd/runecode-broker project-substrate-init-preview --help
go run ./cmd/runecode-broker project-substrate-init-apply --help
go run ./cmd/runecode-broker project-substrate-upgrade-preview --help
go run ./cmd/runecode-broker project-substrate-upgrade-apply --help
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
  "$HOME/.local/bin/runecode" \
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
  "$InstallDir\runecode.exe", `
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
