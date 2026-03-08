# Launcher MicroVM Backend v0

User-visible outcome: RuneCode can launch and manage isolated microVM-based roles on Linux (KVM) with a clear, auditable isolation boundary and no host filesystem mounts.

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-08-1039-launcher-microvm-backend-v0/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

## Task 2: MicroVM Backend Architecture

- Standardize on QEMU microVMs for MVP.
- Pin and record the QEMU version/build provenance (reproducibility + patch posture) and emit it into audit metadata.
- Define a cross-platform abstraction for acceleration:
  - Linux: KVM (MVP runtime)
  - macOS: HVF (only if it does not materially slow the Linux/KVM MVP; otherwise post-MVP)
  - Windows: WHPX/Hyper-V (post-MVP runtime)
- Standardize an isolate <-> host/broker transport that works everywhere:
  - MVP (Linux): vsock (AF_VSOCK) is the preferred transport
  - fallback: `virtio-serial` for portability and non-vsock environments
  - security must not depend on transport choice
- Transport security (MVP requirement):
  - establish a mutually authenticated + encrypted session (e.g., Noise-style handshake)
  - bind isolate identity to the isolate signing public key announced at session start
  - enforce replay protection, message framing, and strict size limits

## Task 3: QEMU Hardening / Host Sandbox (MVP)

- Treat the QEMU process as part of the attack surface; harden and confine it:
  - run as an unprivileged user
  - drop Linux capabilities; set `no_new_privs` where possible
  - apply a restrictive seccomp policy
  - restrict filesystem access to only required paths
  - use an allowlist of QEMU devices and command-line flags
- Record the applied hardening posture in audit (so "we forgot to sandbox" is detectable).

## Task 4: Guest Image + Boot Contract (Minimal)

- Define a minimal Linux guest image contract for role execution.
- Ensure roles are image-pinned by digest/signature in manifests (enforcement can start as “required fields” even if image signature verification is staged).

## Task 5: Disk + Artifact Attachment Model

- Attach encrypted workspace disks as virtual block devices.
- Workspace disk encryption (MVP):
  - provide at-rest protection via host-managed volume encryption (e.g., LUKS2/dm-crypt)
  - key protection posture is recorded (hardware-backed / OS keystore / explicit dev opt-in)
  - fail closed by default if required encryption/key protection is unavailable
- Attach read-only artifacts as explicit virtual disks or read-only channels.
- Enforce “no host filesystem mounts into isolates”.

## Task 6: Resource Limits + Lifecycle

- Define MVP resource controls (vCPU/memory/disk/timeouts) and a watchdog that terminates misbehaving isolates.
- Ensure isolate termination between steps is the default.

## Task 7: Failure Handling

- If microVM launch fails, fail closed and surface a clear error.
- If a running microVM crashes or becomes unresponsive:
  - terminate it
  - mark the step as failed (non-partial)
  - record a clear audit event
- Do not automatically fall back to containers.
- Provide an explicit, separate user flow to opt into container mode (handled by the container backend spec).

## Acceptance Criteria

- On Linux, the launcher can start/stop a microVM role and run at least one deterministic “hello world” role action.
- Isolation backend + assurance level are recorded in the audit log.
- Isolate <-> host transport is mutually authenticated and encrypted; unauthenticated messages are rejected.
- QEMU hardening is enabled by default for MVP.
- No host filesystem mounts are used.
