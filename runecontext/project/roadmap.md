# Product Roadmap

This roadmap is a human-facing summary of planned and implemented product work.
Active lifecycle state lives in `runecontext/changes/*/status.yaml`, and durable completed outcomes live in `runecontext/specs/*.md` as completed work is promoted.

## Upcoming Features

### v0.1.0-alpha.4

- Audit Anchoring v0
  - Audit segment roots can be anchored with verifiable receipts to strengthen tamper-evidence beyond local verification.
  - Planned change: `runecontext/changes/CHG-2026-006-84f0-audit-anchoring-v0/`
- Alpha Implementation Callouts
  - `Minimal TUI v0` must remain a strict client of the brokered local API and real policy, audit, artifact, and approval surfaces.
  - The first end-to-end demo must use explicit artifact handoff, audit capture plus verify, signed policy decisions, and one real isolated backend with no trust-boundary shortcuts.
  - `Workflow Runner + Workspace Roles + Deterministic Gates v0` is sequenced as a thin MVP in `v0.1.0-alpha.3`; remaining hardening and scope complete in `v0.1.0-alpha.4`.
  - `Audit Anchoring v0` and `Container Backend v0 (Explicit Opt-In)` remain follow-on hardening work and must not displace the primary secure path.

### v0.1.0-beta.1

- Formal Spec v0 (TLA+ + CI Model Checking)
  - Critical separation and audit invariants are formally specified and model-checked in CI.
  - Planned change: `runecontext/changes/CHG-2026-015-cae6-formal-spec-v0-tla-ci-model-checking/`
- ZK Proof v0 (One Narrow Proof + Verify)
  - RuneCode can generate and verify at least one narrow zero-knowledge integrity proof.
  - Planned change: `runecontext/changes/CHG-2026-016-8cdb-zk-proof-v0-one-narrow-proof-verify/`
- TUI Multi-Session + Power Workspace v0
  - The terminal client grows into a multi-session, power-user workbench with richer live activity, deeper inspection, saved layouts, and theme presets while staying on the same secure brokered control-plane.
  - Planned change: `runecontext/changes/CHG-2026-037-91be-tui-multi-session-power-workspace-v0/`

### v0.2 (Post-MVP)

- Git Gateway (Commit/Push/PR)
  - Git operations are isolated behind a gateway with outbound patch verification.
  - Planned change: `runecontext/changes/CHG-2026-002-33c5-git-gateway-commit-push-pr/`
- Approval Profiles (Strict/Permissive)
  - Add selectable human-in-the-loop profiles beyond MVP moderate.
  - Planned change: `runecontext/changes/CHG-2026-014-0c5d-approval-profiles-strict-permissive/`
- Workflow Extensibility v0
  - Add schema-validated custom workflows and rebuildable shared-memory accelerators without changing the safety model.
  - Planned change: `runecontext/changes/CHG-2026-017-3d58-workflow-extensibility-v0/`
- Optional LangGraph Runner Runtime Evaluation
  - Optionally evaluate LangGraph as an internal runner runtime for checkpoint/wait/resume mechanics after the native runner foundation is hardened, implementing it only if it is still needed at that time.
  - Planned change: `runecontext/changes/CHG-2026-044-9f2a-optional-langgraph-runner-runtime-evaluation/`
- Auth Gateway Role v0
  - Provider login and refresh runs in an auth-only gateway role; long-lived tokens live only in secretsd.
  - Planned change: `runecontext/changes/CHG-2026-018-5900-auth-gateway-role-v0/`
- Bridge Runtime Protocol v0
  - Shared bridge contracts keep user-installed provider runtimes auditable and in explicit LLM-only mode.
  - Planned change: `runecontext/changes/CHG-2026-019-40c5-bridge-runtime-protocol-v0/`
- OpenAI ChatGPT Subscription Provider (OAuth + Codex Bridge)
  - Access GPT models via a ChatGPT subscription OAuth flow without expanding the trust boundary.
  - Planned change: `runecontext/changes/CHG-2026-020-4425-openai-chatgpt-subscription-provider-oauth-codex-bridge/`
- GitHub Copilot Subscription Provider (Official Runtime Bridge)
  - Access Copilot models via an official local runtime bridge in LLM-only mode.
  - Planned change: `runecontext/changes/CHG-2026-022-8051-github-copilot-subscription-provider-official-runtime-bridge/`
- Local IPC Protobuf Transport v0
  - Migrate local broker IPC to protobuf without changing the logical protocol or local-only posture.
  - Planned change: `runecontext/changes/CHG-2026-021-8d6d-local-ipc-protobuf-transport-v0/`
- Web Research Role
  - Controlled web research runs with strict egress allowlists and citation artifacts.
  - Planned change: `runecontext/changes/CHG-2026-023-59ac-web-research-role/`
- Deps Fetch + Offline Cache
  - Dependencies can be fetched without giving workspace roles internet access.
  - Planned change: `runecontext/changes/CHG-2026-024-acde-deps-fetch-offline-cache/`
- External Audit Anchoring v0
  - Optionally anchor audit roots to external targets with explicit egress and typed receipts.
  - Planned change: `runecontext/changes/CHG-2026-025-5679-external-audit-anchoring-v0/`
- Image/Toolchain Signing Pipeline
  - Isolate images and toolchains are signed and enforced at boot to reduce supply-chain risk.
  - Planned change: `runecontext/changes/CHG-2026-026-98be-image-toolchain-signing-pipeline/`
- Windows MicroVM Runtime Support
  - MicroVM-backed roles run on Windows with consistent policy and audit semantics.
  - Planned change: `runecontext/changes/CHG-2026-028-647e-windows-microvm-runtime-support/`
- macOS Virtualization Polish
  - macOS microVM reliability and UX are improved without changing the security model.
  - Planned change: `runecontext/changes/CHG-2026-029-5e5e-macos-virtualization-polish/`

### vNext (Planned)

- Workflow Concurrency v0
  - Add explicit, auditable shared-workspace concurrency instead of relying on one-run-per-workspace indefinitely.
  - Planned change: `runecontext/changes/CHG-2026-027-71ed-workflow-concurrency-v0/`
- Isolate Attestation v0
  - Upgrade MVP TOFU isolate binding to measured, attestable provisioning without changing the core audit contract.
  - Planned change: `runecontext/changes/CHG-2026-030-98b8-isolate-attestation-v0/`

## Unscheduled (Needs Specs)

No unscheduled items are currently tracked outside the planned work listed above.

## Completed Features

### Current Implemented Foundation

- Dev Environment + CI Bootstrap (Nix Flakes)
  - Standard dev shell via Nix, direnv, and just; CI runs equivalent checks across OSes.
  - Durable spec: `runecontext/specs/dev-environment-ci-bootstrap-nix-flakes.md`
- Monorepo Scaffold + Package Boundaries
  - Clear Go and TypeScript package boundaries with a consistent local build, test, and lint loop.
  - Durable spec: `runecontext/specs/monorepo-scaffold-package-boundaries-v0.md`
- Source Quality Guardrails v0
  - Security-sensitive Go and runner code remain maintainable with language-aware docs, complexity limits, and a repo-specific source-quality gate.
  - Durable spec: `runecontext/specs/source-quality-guardrails-v0.md`
- Protocol & Schema Bundle v0
  - Cross-boundary messages and manifests are schema-validated and hash-addressable.
  - Durable spec: `runecontext/specs/protocol-schema-bundle-v0.md`

- Artifact Store + Data Classes v0
  - Data moves via explicit hash-addressed artifacts with enforced data-class flows.
  - Planned change: `runecontext/changes/CHG-2026-004-acdb-artifact-store-data-classes-v0/`
- Crypto / Key Management v0
  - Manifests and audit events are signed and verifiable with recorded key posture.
  - Planned change: `runecontext/changes/CHG-2026-005-cfd0-crypto-key-management-v0/`
- Audit Log v0 + Verify
  - Runs produce a tamper-evident audit trail with local verification.
  - Planned change: `runecontext/changes/CHG-2026-003-b567-audit-log-v0-verify/`
- Broker + Local API v0
  - Components and isolates communicate via a local brokered API with schema validation.
  - Planned change: `runecontext/changes/CHG-2026-008-62e1-broker-local-api-v0/`

- Policy Engine v0
  - Actions are deterministically allowed or denied by signed policy with explicit approvals.
  - Planned change: `runecontext/changes/CHG-2026-007-2315-policy-engine-v0/`
- Launcher MicroVM Backend v0
  - Roles can run in microVM isolates (Linux-first) with no host filesystem mounts.
  - Planned change: `runecontext/changes/CHG-2026-009-1672-launcher-microvm-backend-v0/`
- Workflow Runner + Workspace Roles + Deterministic Gates v0
  - A first honest end-to-end slice executes offline workspace roles through the secure policy and evidence loop, delivered through scoped child features.
  - Project change: `runecontext/changes/CHG-2026-012-f1ef-workflow-runner-workspace-roles-deterministic-gates-v0/`
  - Feature changes: `runecontext/changes/CHG-2026-033-6e7b-workflow-runner-durable-state-v0/`, `runecontext/changes/CHG-2026-034-b2d4-workspace-roles-v0/`, `runecontext/changes/CHG-2026-035-c8e1-deterministic-gates-v0/`
- Minimal TUI v0
  - Users land in a dashboard-first terminal client, can enter a first-class chat route, approve actions, and inspect runs, artifacts, and audit timelines through the real secure local API.
  - Planned change: `runecontext/changes/CHG-2026-013-d2c9-minimal-tui-v0/`
  - Callout: TUI delivery was tracked under `CHG-2026-038-5a1d-runecode-tui-experience-v0/`, and the alpha implementation was sequenced after the prerequisite contract lane in `CHG-2026-039-7c2e-interactive-control-plane-ux-contracts-v0/`.

- Secretsd + Model Gateway v0
  - Model egress is centralized behind a gateway with scoped secret leases and auditing, delivered through scoped child features.
  - Project change: `runecontext/changes/CHG-2026-011-7240-secretsd-model-gateway-v0/`
  - Feature changes: `runecontext/changes/CHG-2026-031-7a3c-secretsd-core-v0/`, `runecontext/changes/CHG-2026-032-4d1f-model-gateway-v0/`
- Container Backend v0 (Explicit Opt-In)
  - Container isolation is available only via explicit opt-in with reduced-assurance UX after the primary high-assurance path exists.
  - Planned change: `runecontext/changes/CHG-2026-010-54b7-container-backend-v0-explicit-opt-in/`

