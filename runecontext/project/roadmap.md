# Product Roadmap

This roadmap is a human-facing summary of planned and implemented product work.
Active lifecycle state lives in `runecontext/changes/*/status.yaml`, and durable completed outcomes live in `runecontext/specs/*.md` as completed work is promoted.

## Upcoming Features

### v0.1.0-alpha.2

- Artifact Store + Data Classes v0
  - Data moves via explicit hash-addressed artifacts with enforced data-class flows.
- Crypto / Key Management v0
  - Manifests and audit events are signed and verifiable with recorded key posture.
- Audit Log v0 + Verify
  - Runs produce a tamper-evident audit trail with local verification.
- Audit Anchoring v0
  - Audit segment roots can be anchored with verifiable receipts to strengthen tamper-evidence beyond local verification.

### v0.1.0-alpha.3

- Broker + Local API v0
  - Components and isolates communicate via a local brokered API with schema validation.
- Policy Engine v0
  - Actions are deterministically allowed or denied by signed policy with explicit approvals.
- Launcher MicroVM Backend v0
  - Roles can run in microVM isolates (Linux-first) with no host filesystem mounts.
- Container Backend v0 (Explicit Opt-In)
  - Container isolation is available only via explicit opt-in with reduced-assurance UX.

### v0.1.0-alpha.4

- Secretsd + Model Gateway v0
  - Model egress is centralized behind a gateway with scoped secret leases and auditing.
- Workflow Runner + Workspace Roles + Deterministic Gates v0
  - End-to-end runs execute offline workspace roles with deterministic evidence gates.
- Minimal TUI v0
  - Users can approve actions and inspect diffs, artifacts, and audit timelines locally.

### v0.1.0-beta.1

- Formal Spec v0 (TLA+ + CI Model Checking)
  - Critical separation and audit invariants are formally specified and model-checked in CI.
- ZK Proof v0 (One Narrow Proof + Verify)
  - RuneCode can generate and verify at least one narrow zero-knowledge integrity proof.

### v0.2 (Post-MVP)

- Git Gateway (Commit/Push/PR)
  - Git operations are isolated behind a gateway with outbound patch verification.
- Approval Profiles (Strict/Permissive)
  - Add selectable human-in-the-loop profiles beyond MVP moderate.
- Workflow Extensibility v0
  - Add schema-validated custom workflows and rebuildable shared-memory accelerators without changing the safety model.
- Auth Gateway Role v0
  - Provider login and refresh runs in an auth-only gateway role; long-lived tokens live only in secretsd.
- Bridge Runtime Protocol v0
  - Shared bridge contracts keep user-installed provider runtimes auditable and in explicit LLM-only mode.
- OpenAI ChatGPT Subscription Provider (OAuth + Codex Bridge)
  - Access GPT models via a ChatGPT subscription OAuth flow without expanding the trust boundary.
- GitHub Copilot Subscription Provider (Official Runtime Bridge)
  - Access Copilot models via an official local runtime bridge in LLM-only mode.
- Local IPC Protobuf Transport v0
  - Migrate local broker IPC to protobuf without changing the logical protocol or local-only posture.
- Web Research Role
  - Controlled web research runs with strict egress allowlists and citation artifacts.
- Deps Fetch + Offline Cache
  - Dependencies can be fetched without giving workspace roles internet access.
- External Audit Anchoring v0
  - Optionally anchor audit roots to external targets with explicit egress and typed receipts.
- Image/Toolchain Signing Pipeline
  - Isolate images and toolchains are signed and enforced at boot to reduce supply-chain risk.
- Windows MicroVM Runtime Support
  - MicroVM-backed roles run on Windows with consistent policy and audit semantics.
- macOS Virtualization Polish
  - macOS microVM reliability and UX are improved without changing the security model.

### vNext (Planned)

- Workflow Concurrency v0
  - Add explicit, auditable shared-workspace concurrency instead of relying on one-run-per-workspace indefinitely.
- Isolate Attestation v0
  - Upgrade MVP TOFU isolate binding to measured, attestable provisioning without changing the core audit contract.

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
