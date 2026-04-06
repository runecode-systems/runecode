# Tasks

## MicroVM Backend Architecture

- [ ] Standardize on QEMU microVMs for MVP.
- [ ] Pin and record the QEMU version/build provenance (reproducibility + patch posture) and emit it into audit metadata.
- [ ] Define a cross-platform abstraction for acceleration:
  - Linux: KVM (MVP runtime)
  - later macOS runtime work lives in `runecontext/changes/CHG-2026-029-5e5e-macos-virtualization-polish/`
  - later Windows runtime work lives in `runecontext/changes/CHG-2026-028-647e-windows-microvm-runtime-support/`
- [ ] Standardize an isolate <-> host/broker transport that works everywhere:
  - MVP (Linux): vsock (AF_VSOCK) is the preferred transport
  - fallback: `virtio-serial` for portability and non-vsock environments
  - security must not depend on transport choice
- [ ] Transport security (MVP requirement):
  - establish a mutually authenticated + encrypted session (e.g., Noise-style handshake)
  - isolate generates a per-session signing key inside the isolate boundary
  - bind isolate identity to the isolate signing public key announced at session start
  - require proof-of-possession for that key during the handshake
  - enforce replay protection, message framing, and strict size limits

Key provisioning posture alignment (MVP):
- [ ] Treat isolate key provisioning as TOFU for MVP and record binding context:
  - image digest
  - active manifest hash
  - `handshake_transcript_hash`
  - `provisioning_mode=tofu`
  - `session_nonce`
- [ ] Pin the isolate public key to `{run_id, isolate_id, session_id}` as a topology-neutral identity binding; treat hosting node identity as non-authoritative metadata.
- [ ] Fail closed on key mismatch, replayed handshakes, or mid-session identity changes.
- [ ] Later attestation work in `runecontext/changes/CHG-2026-030-98b8-isolate-attestation-v0/` must bind evidence to the same per-session isolate key model.
- [ ] Surface this posture in audit metadata and TUI (degraded posture, not silent).

CI/non-KVM environments (MVP):
- [ ] Runtime behavior remains fail-closed (no auto container fallback).
- [ ] CI should keep broad coverage for backend-agnostic components without requiring KVM.
- [ ] MicroVM end-to-end tests may require a dedicated CI lane (self-hosted runners) where KVM is available.

Parallelization: can be implemented in parallel with broker/policy/schema work; coordinate early on the handshake/session metadata schema and error taxonomy.

## QEMU Hardening / Host Sandbox (MVP)

- [ ] Treat the QEMU process as part of the attack surface; harden and confine it:
  - run as an unprivileged user
  - drop Linux capabilities; set `no_new_privs` where possible
  - apply a restrictive seccomp policy
  - restrict filesystem access to only required paths
  - use an allowlist of QEMU devices and command-line flags
- [ ] Record the applied hardening posture in audit (so "we forgot to sandbox" is detectable).

Parallelization: can be implemented in parallel with the microVM launch path; it depends on a stable hardening posture recording format.

## Guest Image + Boot Contract (Minimal)

- [ ] Define a minimal Linux guest image contract for role execution.
- [ ] Ensure roles are image-pinned by digest/signature in manifests (enforcement can start as “required fields” even if image signature verification is staged).

Parallelization: can be implemented in parallel with image/toolchain signing pipeline design; MVP can start with digest pinning.

## Disk + Artifact Attachment Model

- [ ] Attach encrypted workspace disks as virtual block devices.
- [ ] Workspace disk encryption (MVP):
  - provide at-rest protection via host-managed volume encryption (e.g., LUKS2/dm-crypt)
  - key protection posture is recorded (hardware-backed / OS keystore / explicit dev opt-in)
  - fail closed by default if required encryption/key protection is unavailable
- [ ] Attach read-only artifacts as explicit virtual disks or read-only channels.
- [ ] Enforce “no host filesystem mounts into isolates”.

Parallelization: can be implemented in parallel with artifact store work; it depends on stable artifact attachment and data-class flow rules.

## Resource Limits + Lifecycle

- [ ] Define MVP resource controls (vCPU/memory/disk/timeouts) and a watchdog that terminates misbehaving isolates.
- [ ] Ensure isolate termination between steps is the default.
- [ ] Performance (without weakening the boundary):
  - keep guest images minimal and role-specific to reduce boot latency
  - allow (optional) launcher-managed warm pools or boot caching as an implementation detail, while preserving the same capability model and audit semantics
  - prevent cross-run state bleed:
    - pooled VMs must be reset to a known-clean state (or destroyed) before reuse
    - no reuse of guest disk/memory state across distinct runs/stages unless explicitly designed and audited
    - no reuse of prior session isolate identity private keys across distinct runs/sessions
  - caches must be verifiable:
    - cached images/boot artifacts must remain pinned by digest/signature as specified by the signed manifest
    - cache hits/misses and the resulting image digests are recorded in audit metadata

Parallelization: can be implemented in parallel with workflow runner work; watchdog/timeouts should align with broker deadlines.

## Failure Handling

- [ ] If microVM launch fails, fail closed and surface a clear error.
- [ ] If a running microVM crashes or becomes unresponsive:
  - terminate it
  - mark the step as failed (non-partial)
  - record a clear audit event
- [ ] Do not automatically fall back to containers.
- [ ] Provide an explicit, separate user flow to opt into container mode (handled by the container backend spec).

Parallelization: can be implemented in parallel with container backend work; behavior must remain “no automatic fallback”.

## Acceptance Criteria

- [ ] On Linux, the launcher can start/stop a microVM role and run at least one deterministic “hello world” role action.
- [ ] Isolation backend + assurance level are recorded in the audit log.
- [ ] Isolate <-> host transport is mutually authenticated and encrypted; unauthenticated messages are rejected.
- [ ] Isolate session identity keys are per-session ephemeral and verified with proof-of-possession.
- [ ] QEMU hardening is enabled by default for MVP.
- [ ] No host filesystem mounts are used.
- [ ] Backend kind and assurance posture are available for broker operator-facing run summaries and detail views.
