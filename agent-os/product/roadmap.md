# Product Roadmap

Phased development plan with prioritized features.

This is the canonical view of what is planned next (as specs) and what has shipped (as releases). Move items from "Upcoming Features" to "Completed Features" when the corresponding version is released.

## Upcoming Features

### v0.1.0-alpha.1

- [x] Dev Environment + CI Bootstrap (Nix Flakes) (`agent-os/specs/2026-03-08-1128-dev-env-ci-nix-flakes/`)
  - Standard dev shell via Nix + direnv + just; CI runs equivalent checks across OSes.
- [x] Monorepo Scaffold + Package Boundaries (`agent-os/specs/2026-03-08-1039-scaffold-ci-matrix/`)
  - Clear Go/TS package boundaries with a consistent local build/test/lint loop.
- [x] Source Quality Guardrails v0 (`agent-os/specs/2026-03-13-1415-source-quality-guardrails-v0/`)
  - Keep security-sensitive Go and runner code maintainable with language-aware docs, complexity limits, and a repo-specific source-quality gate.
- [x] Protocol & Schema Bundle v0 (`agent-os/specs/2026-03-08-1039-protocol-schemas-v0/`)
  - Cross-boundary messages/manifests are schema-validated and hash-addressable.

### v0.1.0-alpha.2

- [ ] Artifact Store + Data Classes v0 (`agent-os/specs/2026-03-08-1039-artifact-store-data-classes-v0/`)
  - Data moves via explicit hash-addressed artifacts with enforced data-class flows.
- [ ] Crypto / Key Management v0 (`agent-os/specs/2026-03-08-1039-crypto-key-mgmt-v0/`)
  - Manifests and audit events are signed and verifiable with recorded key posture.
- [ ] Audit Log v0 + Verify (`agent-os/specs/2026-03-08-1039-audit-log-verify-v0/`)
  - Runs produce a tamper-evident audit trail with local verification.
- [ ] Audit Anchoring v0 (`agent-os/specs/2026-03-08-1039-audit-anchoring/`)
  - Audit segment roots can be anchored with verifiable receipts to strengthen tamper-evidence beyond local verification.

### v0.1.0-alpha.3

- [ ] Broker + Local API v0 (`agent-os/specs/2026-03-08-1039-broker-local-api-v0/`)
  - Components/isolates communicate via a local brokered API with schema validation.
- [ ] Policy Engine v0 (`agent-os/specs/2026-03-08-1039-policy-engine-v0/`)
  - Actions are deterministically allowed/denied by signed policy with explicit approvals.
- [ ] Launcher MicroVM Backend v0 (`agent-os/specs/2026-03-08-1039-launcher-microvm-backend-v0/`)
  - Roles can run in microVM isolates (Linux-first) with no host filesystem mounts.
- [ ] Container Backend v0 (Explicit Opt-In) (`agent-os/specs/2026-03-08-1039-container-backend-opt-in-v0/`)
  - Container isolation is available only via explicit opt-in with reduced-assurance UX.

### v0.1.0-alpha.4

- [ ] Secretsd + Model-Gateway v0 (`agent-os/specs/2026-03-08-1039-secretsd-model-gateway-v0/`)
  - Model egress is centralized behind a gateway with scoped secret leases and auditing.
- [ ] Workflow Runner + Workspace Roles + Deterministic Gates v0 (`agent-os/specs/2026-03-08-1039-workflow-workspace-roles-gates-v0/`)
  - End-to-end runs execute offline workspace roles with deterministic evidence gates.
- [ ] Minimal TUI v0 (`agent-os/specs/2026-03-08-1039-minimal-tui-v0/`)
  - Users can approve actions and inspect diffs/artifacts/audit timelines locally.

### v0.1.0-beta.1

- [ ] Formal Spec v0 (TLA+ + CI Model Checking) (`agent-os/specs/2026-03-08-1039-formal-spec-tla-v0/`)
  - Critical separation and audit invariants are formally specified and model-checked in CI.
- [ ] ZK Proof v0 (One Narrow Proof + Verify) (`agent-os/specs/2026-03-08-1039-zk-proof-v0/`)
  - RuneCode can generate and verify at least one narrow ZK integrity proof.

### v0.2 (Post-MVP)

- [ ] Git Gateway (Commit/Push/PR) (`agent-os/specs/2026-03-08-1039-git-gateway/`)
  - Git operations are isolated behind a gateway with outbound patch verification.
- [ ] Approval Profiles (Strict/Permissive) (`agent-os/specs/2026-03-10-1530-approval-profiles-v0/`)
  - Add selectable human-in-the-loop profiles beyond MVP moderate.
- [ ] Workflow Extensibility v0 (`agent-os/specs/2026-03-13-1600-workflow-extensibility-v0/`)
  - Add schema-validated custom workflows and rebuildable shared-memory accelerators without changing the safety model.
- [ ] Auth Gateway Role v0 (`agent-os/specs/2026-03-12-1030-auth-gateway-role-v0/`)
  - Provider login/refresh runs in an auth-only gateway role; long-lived tokens live only in secretsd.
- [ ] Bridge Runtime Protocol v0 (`agent-os/specs/2026-03-13-1601-bridge-runtime-protocol-v0/`)
  - Shared bridge contracts keep user-installed provider runtimes auditable and in explicit LLM-only mode.
- [ ] OpenAI ChatGPT Subscription Provider (OAuth + Codex Bridge) (`agent-os/specs/2026-03-11-1920-openai-chatgpt-subscription-provider-v0/`)
  - Access GPT models via a ChatGPT subscription OAuth flow without expanding the trust boundary.
- [ ] GitHub Copilot Subscription Provider (Official Runtime Bridge) (`agent-os/specs/2026-03-11-1921-github-copilot-subscription-provider-v0/`)
  - Access Copilot models via an official local runtime bridge in LLM-only mode.
- [ ] Local IPC Protobuf Transport v0 (`agent-os/specs/2026-03-13-1602-local-ipc-protobuf-transport-v0/`)
  - Migrate local broker IPC to protobuf without changing the logical protocol or local-only posture.
- [ ] Web Research Role (`agent-os/specs/2026-03-08-1039-web-research-role/`)
  - Controlled web research runs with strict egress allowlists and citation artifacts.
- [ ] Deps Fetch + Offline Cache (`agent-os/specs/2026-03-08-1039-deps-fetch-cache/`)
  - Dependencies can be fetched without giving workspace roles internet access.
- [ ] External Audit Anchoring v0 (`agent-os/specs/2026-03-13-1603-external-audit-anchoring-v0/`)
  - Optionally anchor audit roots to external targets with explicit egress and typed receipts.
- [ ] Image/Toolchain Signing Pipeline (`agent-os/specs/2026-03-08-1039-image-toolchain-signing/`)
  - Isolate images/toolchains are signed and enforced at boot to reduce supply chain risk.
- [ ] Windows MicroVM Runtime Support (`agent-os/specs/2026-03-08-1039-windows-microvm-runtime/`)
  - MicroVM-backed roles run on Windows with consistent policy and audit semantics.
- [ ] macOS Virtualization Polish (`agent-os/specs/2026-03-08-1039-macos-virtualization-polish/`)
  - macOS microVM reliability and UX are improved without changing the security model.

### vNext (Planned)

- [ ] Workflow Concurrency v0 (`agent-os/specs/2026-03-13-1730-workflow-concurrency-v0/`)
  - Add explicit, auditable shared-workspace concurrency instead of relying on one-run-per-workspace forever.
- [ ] Isolate Attestation v0 (`agent-os/specs/2026-03-13-1731-isolate-attestation-v0/`)
  - Upgrade MVP TOFU isolate binding to measured, attestable provisioning without changing the core audit contract.

## Unscheduled (Needs Specs)

## Completed Features
