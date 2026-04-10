# Tasks

## Backend Interface + Ownership Model

- [ ] Make the launcher a long-lived trusted runtime daemon/service under broker control rather than leaving runtime orchestration implicit in helper commands.
- [ ] Freeze the broker/launcher ownership split:
  - broker owns policy evaluation, approvals, artifact authorization, authoritative run state, and operator-facing read models
  - launcher owns backend realization, hardening, attachment materialization, session establishment, watchdogs, and fail-closed termination
- [ ] Define the private trusted broker-to-launcher control contract around service-oriented operations such as launch, terminate, get-state, and runtime update delivery while keeping the broker local API as the only public/untrusted interface.
- [ ] Define a small backend-neutral trusted interface for runtime backends around typed objects such as:
  - `BackendLaunchSpec`
  - `LaunchContext`
  - `AttachmentPlan`
  - `RuntimeImageDescriptor`
  - `BackendLaunchReceipt`
  - `AppliedHardeningPosture`
  - `BackendTerminalReport`
- [ ] Split immutable launch/session/hardening/terminal evidence from mutable lifecycle snapshots so durable state, audit refs, and broker projections do not depend on mutating one overloaded receipt object.
- [ ] Persist launcher-produced runtime evidence in trusted storage and derive broker-facing runtime state from those records rather than from in-memory snapshots or runner-local inference.
- [ ] Keep hypervisor flags, host-local paths, device numbering, guest mount paths, and transport allocation details out of the logical contract.
- [ ] Keep the logical trust-boundary contract as the broker local API; launcher-managed transport/session plumbing must not become a second ad hoc runtime API.

Parallelization: can be implemented in parallel with broker/policy/schema work; agree on ownership and object seams before QEMU-specific code expands.

## Runtime Posture Vocabulary + Authoritative State

- [ ] Define closed vocabulary for operator-visible backend posture:
  - `backend_kind = microvm | container`
  - `assurance_level` means runtime isolation assurance only, or rename it to `isolation_assurance_level` if the schema can change cleanly
- [ ] Keep runtime isolation assurance separate from:
  - provisioning/binding posture (`tofu`, later attested variants)
  - audit verification posture
  - implementation evidence (`qemu`, `kvm`, `hvf`, `whpx`)
- [ ] Keep `backend_kind` operator-facing and topology-neutral (`microvm`, not `qemu`).
- [ ] Define the authoritative runtime state facts the launcher emits and the broker projects into run surfaces, including at least:
  - `backend_kind`
  - isolation assurance
  - `isolate_id`
  - `session_id`
  - provisioning posture
  - runtime image descriptor digest
  - concrete component digests used at boot
  - applied hardening posture summary
  - launch/terminal failure reason codes
- [ ] Ensure broker `RunSummary` / `RunDetail` surfaces project launcher facts rather than inferring backend posture indirectly from audit-only or runner-only data.
- [ ] Ensure broker runtime posture is derived from persisted launcher evidence objects and remains durable across broker or launcher restarts.

Parallelization: can be implemented in parallel with broker local API work; coordinate early on schema vocabulary and run-state projection.

## MicroVM Backend Architecture

- [ ] Standardize on QEMU microVMs for MVP.
- [ ] Implement the first real Linux/QEMU/KVM vertical slice through the broker->launcher->isolate->broker path before expanding additional contract-only work.
- [ ] Pin and record the QEMU version/build provenance (reproducibility + patch posture) and emit it into audit metadata.
- [ ] Define a cross-platform abstraction for acceleration:
  - Linux: KVM (MVP runtime)
  - later macOS runtime work lives in `runecontext/changes/CHG-2026-029-5e5e-macos-virtualization-polish/`
  - later Windows runtime work lives in `runecontext/changes/CHG-2026-028-647e-windows-microvm-runtime-support/`
- [ ] One isolate maps to one `role_instance`, one `role_kind`, and one `role_family` in MVP.
- [ ] Standardize an isolate <-> host/broker transport that works everywhere:
  - MVP (Linux): vsock (AF_VSOCK) is the preferred transport
  - fallback: `virtio-serial` for portability and non-vsock environments
  - security must not depend on transport choice
- [ ] Keep transport selection backend-private and platform-specific while preserving one transport-neutral secure-session model across Linux, macOS, and Windows follow-on work.

## Session Handshake + Binding Contract

- [ ] Define a small typed session-establishment family (e.g. `HostHello`, `IsolateHello`, `SessionReady`) that runs before ordinary broker API traffic.
- [ ] Define `LaunchContext` as immutable read-only guest-visible session input that is digest-bound into launch/session identity.
- [ ] Transport security (MVP requirement):
  - establish a mutually authenticated + encrypted session using a standard secure-channel design rather than custom crypto
  - isolate generates a per-session signing key inside the isolate boundary
  - bind isolate identity to the isolate signing public key announced at session start
  - require cryptographically verified proof-of-possession for that key during the handshake
  - treat transport/session keys as channel keys only; they are not the durable isolate identity
  - launcher generates a unique `session_nonce`
  - record a canonical `handshake_transcript_hash`
  - enforce replay protection, message framing, and strict size limits
- [ ] Treat `SessionReady` as a verified summary of an already-established secure session rather than as a declarative trust signal.
- [ ] Hosting node identity remains audit metadata, not isolate identity; the isolate key binding model must stay topology-neutral so future multi-node scheduling does not change object semantics.

Key provisioning posture alignment (MVP):
- [ ] Treat isolate key provisioning as TOFU for MVP and record binding context:
  - image digest
  - active manifest hash
  - `handshake_transcript_hash`
  - `provisioning_mode=tofu`
  - `session_nonce`
- [ ] Pin the isolate public key to `{run_id, isolate_id, session_id}` as a topology-neutral identity binding; treat hosting node identity as non-authoritative metadata.
- [ ] Fail closed on key mismatch, replayed handshakes, transcript mismatches, unsupported posture, or mid-session identity changes.
- [ ] Later attestation work in `runecontext/changes/CHG-2026-030-98b8-isolate-attestation-v0/` must bind evidence to the same per-session isolate key model.
- [ ] Surface this posture in audit metadata and TUI (degraded posture, not silent).

CI/non-KVM environments (MVP):
- [ ] Runtime behavior remains fail-closed (no auto container fallback).
- [ ] CI should keep broad coverage for backend-agnostic components without requiring KVM.
- [ ] MicroVM end-to-end tests may require a dedicated CI lane (self-hosted runners) where KVM is available.

Parallelization: can be implemented in parallel with broker/policy/schema work; coordinate early on the session handshake family, metadata schema, and error taxonomy.

## QEMU Hardening / Host Sandbox (MVP)

- [ ] Treat the QEMU process as part of the attack surface; harden and confine it:
  - run as an unprivileged user
  - drop Linux capabilities; set `no_new_privs` where possible
  - apply a restrictive seccomp policy
  - restrict filesystem access to only required paths
  - use an allowlist of QEMU devices and command-line flags
- [ ] Build QEMU invocation through one launcher-owned allowlisted argv/device policy builder rather than distributing flag assembly across the codebase.
- [ ] Define typed `AppliedHardeningPosture` with:
  - requested posture
  - effective posture
  - degraded reasons
- [ ] Keep common posture fields backend-neutral and cover at least:
  - execution identity posture
  - filesystem exposure posture
  - network exposure posture
  - syscall filtering posture
  - device surface posture
  - control-channel kind
  - acceleration kind
- [ ] Allow backend-specific evidence (QEMU provenance, sandbox/profile identifiers, policy IDs) without leaking host-local paths.
- [ ] Record the applied hardening posture in audit and expose a summary through broker run detail so "we forgot to sandbox" is detectable.
- [ ] Ensure the recorded hardening posture captures what was actually enforced, not only the requested policy target.

Parallelization: can be implemented in parallel with the microVM launch path; it depends on a stable hardening posture recording format.

## Guest Image + Boot Contract (Minimal)

- [ ] Define a digest-addressed `RuntimeImageDescriptor` object for guest image identity rather than loose kernel/rootfs paths or mutable tags.
- [ ] `RuntimeImageDescriptor` should carry at least:
  - backend/platform compatibility
  - boot-contract version
  - kernel/rootfs/initrd or equivalent component digests
  - optional signing metadata / signer identity hooks
  - future attestation/measurement hooks
- [ ] Ensure roles are descriptor-pinned by digest/signature in manifests (enforcement can start as “required fields” even if image signature verification is staged).
- [ ] Record both the descriptor digest and the concrete component digests actually used at boot in launch/audit receipts.

Parallelization: can be implemented in parallel with image/toolchain signing pipeline design; MVP can start with digest pinning while preserving later signature enforcement.

## Attachment Plan + Artifact Movement

- [ ] Define a typed `AttachmentPlan` keyed by logical attachment role rather than hypervisor-specific disk numbering.
- [ ] MVP attachment roles should include:
  - `launch_context` (read-only)
  - `workspace` (run-scoped read-write encrypted volume)
  - `input_artifacts` (read-only immutable inputs)
  - `scratch` (ephemeral read-write)
- [ ] Attach encrypted workspace disks as virtual block devices.
- [ ] Workspace disk encryption (MVP):
  - provide at-rest protection via host-managed volume encryption (e.g., LUKS2/dm-crypt)
  - key protection posture is recorded (hardware-backed / OS keystore / explicit dev opt-in)
  - fail closed by default if required encryption/key protection is unavailable
- [ ] Attach read-only artifacts as explicit virtual disks or read-only channels.
- [ ] Enforce “no host filesystem mounts into isolates”.
- [ ] Keep host-local paths out of boundary-visible attachment contracts and audit payloads.
- [ ] Guest mount locations remain backend/guest convention rather than public contract identity.
- [ ] Keep broker authorization at logical attachment role + digest level only; launcher resolves and materializes backend-private storage/layout without exposing host path identity through the trusted control contract.

Parallelization: can be implemented in parallel with artifact store work; it depends on stable artifact attachment and data-class flow rules.

## Resource Limits + Lifecycle

- [ ] Define MVP resource controls (vCPU/memory/disk/timeouts) and a watchdog that terminates misbehaving isolates.
- [ ] Define an explicit backend lifecycle progression for launch/session establishment (e.g. `planned`, `launching`, `started`, `binding`, `active`, `terminating`, `terminated`) without overloading shared operator run lifecycle state.
- [ ] Persist lifecycle transitions separately from immutable launch evidence so later durable-state and restart recovery work can reuse one clear model.
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

## Failure Handling + Error Taxonomy

- [ ] Define stable backend error codes for at least:
  - acceleration unavailable
  - hypervisor launch failed
  - image descriptor/signature mismatch
  - attachment plan invalid
  - handshake failed
  - replay detected
  - session binding mismatch
  - guest unresponsive
  - watchdog timeout
  - required hardening unavailable
  - required disk encryption unavailable
- [ ] If microVM launch fails, fail closed and surface a clear error.
- [ ] If a running microVM crashes or becomes unresponsive:
  - terminate it
  - mark the step as failed (non-partial)
  - record a clear audit event
  - emit a typed `BackendTerminalReport`
- [ ] Do not automatically fall back to containers.
- [ ] Provide an explicit, separate user flow to opt into container mode (handled by the container backend spec).
- [ ] Add a backend conformance suite that asserts no automatic fallback, correct posture separation, attachment semantics, hardening/error reporting, and broker projection behavior across backend implementations.

Parallelization: can be implemented in parallel with container backend work; behavior must remain “no automatic fallback” and error vocabulary should stay reusable across backends.

## Audit Payloads + Run-Surface Alignment

- [ ] Define explicit payload schemas for `isolate_session_started` and `isolate_session_bound`.
- [ ] Keep those payloads small, reference-heavy, and topology-neutral; reference launch/session/image/hardening objects rather than duplicating host-local implementation details.
- [ ] Ensure later attestation work can upgrade TOFU posture without breaking these event families.
- [ ] Keep backend kind, runtime isolation assurance, provisioning posture, and audit posture visible through shared broker run-summary/run-detail surfaces rather than platform-specific side channels.
- [ ] Make the broker the owner of audit emission for launcher-visible runtime events while the launcher supplies referenced evidence objects and digests.

## Implementation Sequencing

- [ ] Finalize the trusted broker-to-launcher control contract, secure-session design, and immutable evidence model before expanding implementation-specific runtime code.
- [ ] Wire a fake backend through the real broker->launcher->broker path to prove lifecycle updates, evidence persistence, broker projection, and audit emission.
- [ ] Replace the fake backend with the real Linux/QEMU/KVM implementation behind the same control contract.
- [ ] Add backend-conformance and KVM-backed end-to-end verification before treating the change as complete.

Parallelization: can be implemented in parallel with audit and broker work; it depends on stable payload schemas, reference roles, and posture vocabulary.

## Acceptance Criteria

- [ ] On Linux, the launcher can start/stop a microVM role and run at least one deterministic “hello world” role action.
- [ ] Launcher/broker ownership split and the backend-neutral launch/session/attachment contracts are explicit enough that later backends do not need to redefine the core model.
- [ ] Broker<->launcher control is service-oriented and private to the trusted domain while the broker local API remains the only public/untrusted boundary.
- [ ] Isolation backend + runtime isolation assurance are recorded in audit and projected through broker operator-facing run surfaces.
- [ ] Broker-facing runtime state remains derivable from persisted launcher evidence after restart and does not depend on runner-local inference.
- [ ] Runtime isolation assurance, provisioning/binding posture, and audit posture are not silently collapsed into one ambiguous field.
- [ ] Isolate <-> host transport is mutually authenticated and encrypted; unauthenticated messages are rejected.
- [ ] Isolate session identity keys are per-session ephemeral and verified with proof-of-possession.
- [ ] Guest image identity is descriptor-pinned by digest and the concrete boot inputs are recorded.
- [ ] Attachment planning is typed, topology-neutral, and uses no host filesystem mounts.
- [ ] QEMU hardening is enabled by default for MVP.
- [ ] Backend launch/runtime failures surface through stable error codes and terminal reports rather than only logs or exit status.
- [ ] Audit payload schemas for isolate-session events are explicit enough for later attestation work to reuse without a format break.
- [ ] The same trusted control contract and conformance checks are reusable by container, macOS, Windows, attestation, and durable-state follow-on work without redefining runtime posture semantics.
