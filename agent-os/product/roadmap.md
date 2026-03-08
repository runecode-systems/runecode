# Product Roadmap

Phased development plan with prioritized features.

This is the canonical view of what is planned next (as specs) and what has shipped (as releases). Move items from "Upcoming Features" to "Completed Features" when the corresponding version is released.

## Upcoming Features

### v0.1.0-alpha.1

- [ ] Dev Environment + CI Bootstrap (Nix Flakes) (`agent-os/specs/2026-03-08-1128-dev-env-ci-nix-flakes/`)
  - Standard dev shell via Nix + direnv + just; CI runs equivalent checks across OSes.
- [ ] Monorepo Scaffold + Package Boundaries (`agent-os/specs/2026-03-08-1039-scaffold-ci-matrix/`)
  - Clear Go/TS package boundaries with a consistent local build/test/lint loop.
- [ ] Protocol & Schema Bundle v0 (`agent-os/specs/2026-03-08-1039-protocol-schemas-v0/`)
  - Cross-boundary messages/manifests are schema-validated and hash-addressable.

### v0.1.0-alpha.2

- [ ] Artifact Store + Data Classes v0 (`agent-os/specs/2026-03-08-1039-artifact-store-data-classes-v0/`)
  - Data moves via explicit hash-addressed artifacts with enforced data-class flows.
- [ ] Crypto / Key Management v0 (`agent-os/specs/2026-03-08-1039-crypto-key-mgmt-v0/`)
  - Manifests and audit events are signed and verifiable with recorded key posture.
- [ ] Audit Log v0 + Verify (`agent-os/specs/2026-03-08-1039-audit-log-verify-v0/`)
  - Runs produce a tamper-evident audit trail with local verification.

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
- [ ] Web Research Role (`agent-os/specs/2026-03-08-1039-web-research-role/`)
  - Controlled web research runs with strict egress allowlists and citation artifacts.
- [ ] Deps Fetch + Offline Cache (`agent-os/specs/2026-03-08-1039-deps-fetch-cache/`)
  - Dependencies can be fetched without giving workspace roles internet access.
- [ ] Audit Anchoring (`agent-os/specs/2026-03-08-1039-audit-anchoring/`)
  - Audit roots can be optionally anchored externally with verifiable receipts.
- [ ] Image/Toolchain Signing Pipeline (`agent-os/specs/2026-03-08-1039-image-toolchain-signing/`)
  - Isolate images/toolchains are signed and enforced at boot to reduce supply chain risk.
- [ ] Windows MicroVM Runtime Support (`agent-os/specs/2026-03-08-1039-windows-microvm-runtime/`)
  - MicroVM-backed roles run on Windows with consistent policy and audit semantics.
- [ ] macOS Virtualization Polish (`agent-os/specs/2026-03-08-1039-macos-virtualization-polish/`)
  - macOS microVM reliability and UX are improved without changing the security model.

## Unscheduled (Needs Specs)

## Completed Features
