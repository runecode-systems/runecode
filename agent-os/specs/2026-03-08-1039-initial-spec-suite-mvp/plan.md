# Initial Spec Suite (MVP + Post-MVP)

Goal: create an initial set of small, implementation-ready specs and split them into MVP vs post-MVP.

Constraints (MVP-defining):
- Security-first; isolation and cryptographic provenance are first-class.
- Deny-by-default capabilities; explicit opt-ins and approvals for risk.
- Single-user, single-machine operation for MVP (no multi-user daemon or remote access).
- MicroVMs are the preferred/primary isolation boundary.
- Container isolation is supported but is explicit opt-in only; it must never be an automatic fallback.
- No host filesystem mounts into isolates; data moves via explicit, hash-addressed artifacts.

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-08-1039-initial-spec-suite-mvp/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

## Task 2: Create MVP + Post-MVP Spec Folders

Create the following spec folders and their documentation files (no direct references to the source discovery doc filename/path).

MVP (v0.1):
- Implement first: Dev Environment + CI Bootstrap (Nix Flakes): `agent-os/specs/2026-03-08-1128-dev-env-ci-nix-flakes/`
- Monorepo Scaffold + Package Boundaries: `agent-os/specs/2026-03-08-1039-scaffold-ci-matrix/`
- Protocol & schema bundle v0: `agent-os/specs/2026-03-08-1039-protocol-schemas-v0/`
- Crypto/key management v0: `agent-os/specs/2026-03-08-1039-crypto-key-mgmt-v0/`
- Artifact store + data classes v0: `agent-os/specs/2026-03-08-1039-artifact-store-data-classes-v0/`
- Audit log v0 + verify: `agent-os/specs/2026-03-08-1039-audit-log-verify-v0/`
- Policy engine v0: `agent-os/specs/2026-03-08-1039-policy-engine-v0/`
- Launcher microVM backend v0: `agent-os/specs/2026-03-08-1039-launcher-microvm-backend-v0/`
- Container backend v0 (explicit opt-in): `agent-os/specs/2026-03-08-1039-container-backend-opt-in-v0/`
- Broker + local API v0: `agent-os/specs/2026-03-08-1039-broker-local-api-v0/`
- Secretsd + model-gateway v0: `agent-os/specs/2026-03-08-1039-secretsd-model-gateway-v0/`
- Workflow runner + workspace roles + deterministic gates v0: `agent-os/specs/2026-03-08-1039-workflow-workspace-roles-gates-v0/`
- Minimal TUI v0: `agent-os/specs/2026-03-08-1039-minimal-tui-v0/`
- Formal spec v0 (TLA+ + CI model checking): `agent-os/specs/2026-03-08-1039-formal-spec-tla-v0/`
- ZK proof v0 (one narrow proof + verify): `agent-os/specs/2026-03-08-1039-zk-proof-v0/`

Post-MVP (v0.2+):
- Git gateway (commit/push/PR): `agent-os/specs/2026-03-08-1039-git-gateway/`
- Web research role: `agent-os/specs/2026-03-08-1039-web-research-role/`
- Deps fetch + offline cache: `agent-os/specs/2026-03-08-1039-deps-fetch-cache/`
- Audit anchoring: `agent-os/specs/2026-03-08-1039-audit-anchoring/`
- Image/toolchain signing pipeline: `agent-os/specs/2026-03-08-1039-image-toolchain-signing/`
- Windows microVM runtime support: `agent-os/specs/2026-03-08-1039-windows-microvm-runtime/`
- macOS virtualization polish: `agent-os/specs/2026-03-08-1039-macos-virtualization-polish/`

## Task 3: Update Product Roadmap

Update `agent-os/product/roadmap.md` to:
- Add spec entries under `### v0.1 (MVP)` and `### v0.2 (Post-MVP)`.
- Follow `product/roadmap-conventions` (checkbox spec entries, outcome-focused descriptions).
- Remove the duplicated unscheduled checkboxes that are now represented by spec entries.

## MVP Cut (What We Intentionally Leave Out)

To keep MVP essential and focused, the following are explicitly post-MVP unless pulled in by a must-have integration need:
- GitHub PR/push automation via a dedicated git gateway.
- Web research crawling + domain bundles/domain-expansion workflows.
- Dependency fetching role and offline dependency caches.
- External audit anchoring (TPM PCR / RFC3161 / witness services) beyond local verification.

## Notes / Open Questions (To Resolve Inside Individual Specs)

- Local durable state: SQLite (WAL) for MVP (runs/approvals/artifact metadata/audit indexing), with append-only files for large immutable blobs.
  - Pin SQLite to a version that includes known WAL integrity fixes (e.g., the WAL-reset fix in SQLite >= 3.52.0 or an equivalent backport) when WAL is enabled.
- Protocol encoding: MVP uses JSON (schema-validated) for both on-wire and on-disk objects, while keeping the logical object model encoding-agnostic so on-wire can migrate post-MVP to protobuf over local IPC (gRPC optional and local-only).
- Transport: vsock-first on Linux, virtio-serial fallback for portability; always run a mandatory message-level authenticated+encrypted session (do not rely on transport properties).
- MicroVM support by OS: MVP runtime targets Linux + KVM; macOS HVF is optional only if it does not materially slow MVP; Windows runtime support is post-MVP.
- Local auth: local IPC only; require OS peer credentials on platforms that support them (fail closed for MVP when unavailable).
