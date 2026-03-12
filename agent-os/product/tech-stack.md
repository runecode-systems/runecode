# Tech Stack

## Frontend

- Go (TUI-first dashboard for runs, approvals, diffs, artifacts, and audit timeline)

## Backend

- Go (local control plane / security kernel: launcher/PEP, broker, secrets daemon, audit daemon)
- TypeScript + Node.js (LangGraph-based workflow runner/scheduler; treated as untrusted at runtime)

## Database

- SQLite (WAL) for MVP durable local state and indexing (runs, approvals, artifact metadata, audit indexing)
- Append-only files for large immutable blobs (CAS objects, audit log segments), with SQLite holding indexes/pointers
- SQLite version is pinned when WAL is enabled (include known WAL integrity fixes such as the WAL-reset fix in SQLite >= 3.52.0 or an equivalent backport; record SQLite library version in audit metadata)
- SQLite operational guidance (MVP): avoid cross-process write contention.
  - Prefer separate SQLite databases per subsystem (or strict transaction discipline) if contention becomes an issue; record the chosen layout and WAL mode in audit metadata.

## Other

- Isolation: microVMs preferred (QEMU with KVM/HVF/WHPX); hardened containers fallback with explicit reduced-assurance UX
- MVP runtime target: Linux + KVM on a single-user local machine; macOS HVF and Windows WHPX/Hyper-V are post-MVP unless proven low-effort
- Isolate transport (MVP): vsock-first on Linux; virtio-serial fallback; message-level authenticated+encrypted session required
- Local APIs/protocols: JSON + JSON Schema for MVP; post-MVP on-wire RPC may migrate to protobuf over local IPC (gRPC optional and must remain local-only)
- Security/provenance: signed capability manifests, per-isolate identity keys, hash-chained audit events, content-addressed artifacts
- Storage: encrypted workspace disks/snapshots (LUKS2/dm-crypt baseline)
- Key management (where available): TPM/Secure Enclave; user-presence approvals via OS-confirmation and/or FIDO2/YubiKey
- CI: GitHub Actions (Linux/macOS/Windows)
- Workflow/docs: AgentOS (`agent-os/**`) as git-native source of truth; OpenCode runs inside offline workspace roles
