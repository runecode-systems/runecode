# Design

## Overview
Implement the Linux-first microVM isolation backend, including launcher hardening, guest-image contracts, artifact attachment, fail-closed lifecycle handling, and the small set of typed contracts that later container, attestation, macOS, and Windows work should reuse without changing core runtime semantics.

## Key Decisions
- MicroVMs are the preferred/primary boundary.
- The launcher becomes a long-lived trusted runtime daemon/service under broker control.
- The broker remains the control-plane authority for policy, approvals, artifact authorization, and authoritative run/read-model state.
- The launcher remains the runtime authority for backend realization, hardening, attachment materialization, session establishment, watchdogs, and fail-closed termination.
- The logical trust-boundary contract remains the broker local API. Launcher-managed vsock/virtio-serial plumbing and session establishment are implementation details for carrying the same broker-mediated semantics, not a second runtime API.
- The broker<->launcher integration should be modeled as a private trusted control contract for launch, termination, state query, and runtime update delivery rather than as ad hoc process helpers or a second public API.
- Define a backend-neutral trusted interface early and keep it small.
- MVP uses vsock-first on Linux with a virtio-serial fallback, with mandatory message-level authentication+encryption (do not rely on transport properties).
- The secure session should be implemented with a standard transport-neutral secure-channel design rather than custom crypto; typed handshake objects remain semantic contract summaries rather than the full cryptographic protocol.
- Isolate session identity keys are per-session ephemeral, generated inside the isolate boundary, and must prove possession during the handshake.
- Isolate key provisioning is TOFU for MVP; binding context (image digest + handshake transcript hash) is recorded and surfaced as a degraded posture.
- Hosting node identity is audit metadata, not isolate identity; the isolate key binding model must stay topology-neutral so future multi-node scheduling does not change object semantics.
- Later attestation work upgrades the same session-key model rather than replacing it with a different isolate identity contract.
- Keep hypervisor flags, host-local filesystem paths, device numbering, guest mount paths, and transport allocation details out of boundary-visible or cross-backend logical contracts.
- Operator-visible runtime posture is split into separate axes:
  - `backend_kind` identifies the selected backend class (`microvm`, later `container`)
  - `assurance_level` should refer only to runtime isolation assurance, or be renamed to `isolation_assurance_level` when schemas can evolve cleanly
  - provisioning/binding posture remains separate (`tofu` in MVP, later attested variants)
  - audit verification posture remains separate
- Keep backend kind operator-facing and topology-neutral (`microvm`, not `qemu`). Hypervisor implementation details such as `qemu` and acceleration details such as `kvm`, `hvf`, or `whpx` belong in detailed runtime/hardening evidence rather than public run identity.
- The broker should expose authoritative runtime state as a projection of launcher-produced facts rather than inferring backend posture indirectly from audit-only or runner-only data.
- Launcher-produced facts should be persisted as immutable evidence objects and broker-visible runtime state should be derived from those records rather than from mutable in-memory snapshots alone.
- One isolate maps to one `role_instance`, one `role_kind`, and one `role_family` in MVP.
- MicroVM failure must not auto-enable container mode.
- QEMU hardening/sandboxing is part of the MVP security boundary (not a later polish item).
- Performance work (boot latency, warm pools, caching) must not relax isolation semantics or bypass audit/policy.
- Warm pools/caches must not introduce cross-run state bleed; reuse requires reset-to-clean (or destroy) semantics and verifiable, manifest-pinned artifacts.
- CI may not always have KVM; backend-agnostic components must be testable without KVM, while microVM e2e runs can use a dedicated KVM-capable lane.
- Backend kind and runtime isolation assurance are first-class operator-visible outputs that should align with shared broker run-summary/run-detail surfaces rather than existing only as audit side notes.
- The first implementation step should be one thin but real Linux/QEMU/KVM vertical slice that proves the control contract, secure session, evidence persistence, broker projection, and audit flow before further public contract growth.

## Control-Plane Ownership
- Broker responsibilities:
  - policy evaluation and approval handling
  - artifact authorization and data-class enforcement
  - authoritative run state and operator-facing read models
  - projecting launcher-produced facts into `RunSummary` and `RunDetail`
  - persisting launcher-produced immutable evidence and emitting audit events that reference those stored objects
- Launcher responsibilities:
  - backend realization
  - hardening and confinement
  - attachment materialization
  - session establishment and binding validation
  - watchdogs, timeouts, and fail-closed termination
  - producing immutable runtime evidence objects and incremental lifecycle updates for broker projection
- The broker may request a backend kind; the launcher may realize it or fail closed. The launcher must never silently substitute another backend.

## Trusted Runtime Control Contract
- The broker local API remains the only public/untrusted boundary.
- Broker-to-launcher communication is a private trusted contract inside the trusted domain and should remain service-oriented even if the initial process bootstrap is simple.
- The control contract should cover at least:
  - `Launch(BackendLaunchSpec)`
  - `Terminate(run/stage/role/isolate tuple)`
  - `GetState(...)`
  - runtime update delivery or subscription for lifecycle/evidence/terminal events
- This contract should use the backend-neutral `launcherbackend` types and must not expose hypervisor argv, host paths, transport allocation details, or guest mount conventions.
- The launcher should be free to realize the contract via a long-lived daemon, a supervised local service, or a broker-managed trusted process, provided the logical service contract stays stable.

## Foundational Contracts
The change should define a small trusted backend contract around typed objects such as:
- `BackendLaunchSpec`: broker-authored, policy-authorized launch intent for one role instance.
- `LaunchContext`: immutable read-only guest-visible context bound into launch/session identity.
- `AttachmentPlan`: authorized runtime attachments keyed by logical role instead of hypervisor-specific disk numbering.
- `RuntimeImageDescriptor`: digest-addressed guest image descriptor pinned in manifests.
- `BackendLaunchReceipt`: launcher-produced record of what backend was realized and what image inputs were used.
- `IsolateSessionBinding`: per-session isolate identity binding pinned to `{run_id, isolate_id, session_id}` with TOFU/attestation posture.
- `AppliedHardeningPosture`: requested posture, effective posture, and degraded reasons for runtime sandboxing.
- `BackendTerminalReport`: typed terminal outcome/failure report for lifecycle, audit, and operator surfaces.

These names may ship as protocol objects, trusted internal structs, or both, but the logical seams should be frozen early and remain backend-neutral.

### Evidence Model Recommendation
- Treat launch/session/hardening/terminal outputs as immutable launcher-produced evidence objects first and as read-model inputs second.
- `BackendLaunchReceipt` should represent immutable launch/realization evidence rather than absorbing mutable lifecycle state over time.
- Backend lifecycle progression should be tracked as distinct lifecycle updates or snapshots rather than by mutating the original realization receipt.
- `BackendTerminalReport` remains a separate terminal evidence object for failure/completion outcomes.
- Persisted evidence should be content-addressed where practical so audit payloads and broker state can reference stable digests.

### Protocol Surface Recommendation
- Keep `RuntimeImageDescriptor` as an explicit protocol object.
- Keep typed handshake messages internal to the trusted implementation unless a cross-language/public requirement emerges.
- Promote persisted digest-referenced evidence objects to protocol schemas only when audit, verification, or cross-process durability requires a stable typed public format.

## Runtime Posture Model
- Keep operator-visible posture as separate axes:
  - `backend_kind` identifies the selected backend class (`microvm`, later `container`)
  - `assurance_level` should refer only to runtime isolation assurance, or be renamed to `isolation_assurance_level` when schemas can evolve cleanly
  - provisioning/binding posture remains separate (`tofu` in MVP, later attested variants)
  - audit verification posture remains separate
- Keep `backend_kind` operator-facing and topology-neutral (`microvm`, not `qemu`).
- Hypervisor implementation details such as `qemu` and acceleration details such as `kvm`, `hvf`, or `whpx` belong in detailed runtime/hardening evidence rather than public run identity.
- The broker should expose authoritative runtime state as a projection of launcher-produced facts rather than inferring backend posture indirectly from audit-only or runner-only data.
- Broker-facing runtime posture should be derived from persisted launcher evidence rather than copied from runner state or lossy local status fields.

## Session Establishment Model
- MVP uses vsock-first on Linux with a `virtio-serial` fallback, but transport choice must not change message or identity semantics.
- Session establishment should use a small typed handshake family such as:
  - `HostHello`
  - `IsolateHello`
  - `SessionReady`
- Ordinary broker API traffic flows only after this secure session is established.
- The durable isolate identity is the per-session isolate signing key generated inside the isolate boundary.
- Transport/session keys are channel keys only and must not become the durable identity model.
- The launcher generates `session_nonce`.
- The handshake records canonical transcript bytes and a `handshake_transcript_hash`.
- Replay detection, message framing, and strict size limits are required.
- TOFU is the MVP provisioning mode; later attestation work upgrades the same session-key model rather than replacing it with a different identity contract.
- The cryptographic session should be implemented using a standard secure-channel pattern that can export transcript material, channel binding data, and replay-resistant session state across `vsock`, `virtio-serial`, Windows named pipes, or future transports without changing the logical handshake semantics.
- `SessionReady` should be treated as a verified summary emitted after the secure channel and isolate key proof are established, not as the mechanism that itself establishes trust.

## Image and Attachment Model
- Manifests should identify guest images through one digest-addressed `RuntimeImageDescriptor` rather than mutable tags, host paths, or loose file references.
- The descriptor should carry:
  - backend/platform compatibility
  - boot-contract version
  - kernel/rootfs/initrd or equivalent component digests
  - optional signing metadata
  - future attestation/measurement hooks
- Launch/audit receipts should record both the descriptor digest and the concrete component digests actually used at boot.
- Runtime data movement should use a typed `AttachmentPlan` keyed by logical attachment role rather than hypervisor-specific device numbering.
- MVP attachment roles should include:
  - `launch_context` read-only
  - `workspace` run-scoped read-write encrypted volume
  - `input_artifacts` read-only immutable inputs
  - `scratch` ephemeral read-write working state
- No host filesystem mounts are permitted into the isolate.
- Guest mount locations remain backend/guest convention rather than public contract identity.
- The broker should authorize logical attachments by role and digest; the launcher should resolve and materialize those inputs inside launcher-private state without receiving boundary-visible host path identity from the broker.
- Attachment materialization should remain backend-private so later container, macOS, and Windows work can reuse the same logical attachment semantics without inheriting Linux path/layout assumptions.

## Hardening and Failure Model
- Treat the QEMU process as part of the attack surface and harden/confine it by default.
- Record hardening posture as a typed object with requested posture, effective posture, and degraded reasons.
- Common posture fields should remain backend-neutral and cover:
  - execution identity posture
  - filesystem exposure posture
  - network exposure posture
  - syscall filtering posture
  - device surface posture
  - control-channel kind
  - acceleration kind
- Backend-specific evidence may add QEMU provenance, sandbox/profile identifiers, and similar implementation details without leaking host-local paths.
- Error handling should use a stable backend error taxonomy; launch/runtime failures must not be represented only through logs, exit codes, or hypervisor stderr parsing.
- Applied hardening posture should describe what was actually enforced, not only what was requested.
- QEMU argv/device policy should be constructed through an allowlisted builder owned by the launcher implementation rather than by assembling ad hoc command flags across call sites.

## Audit and Operator Surfaces
- `isolate_session_started` and `isolate_session_bound` should have explicit payload schemas rather than only registry entries or test-local shapes.
- Audit payloads should remain small and reference-heavy, pointing to launch/session/image/hardening objects instead of duplicating host-local implementation details.
- Later attestation work should upgrade TOFU posture without requiring a format break for these audit event families.
- Backend kind, runtime isolation assurance, provisioning posture, and audit posture should all remain visible through shared broker run-summary/run-detail surfaces.
- Broker should own audit event emission for operator-facing runtime events; launcher should supply the evidence objects and digests that broker persists and references.

## Lifecycle
- Backend lifecycle should be explicit enough for launch/session debugging and durable state without overloading operator run lifecycle state. A minimal internal progression is:
  - `planned`
  - `launching`
  - `started`
  - `binding`
  - `active`
  - `terminating`
  - `terminated`

## Main Workstreams
- Backend Interface + Ownership Model
- Runtime Posture Vocabulary + Authoritative State
- MicroVM Backend Architecture
- Session Handshake + Binding Contract
- QEMU Hardening / Host Sandbox (MVP)
- Runtime Image Descriptor + Boot Contract
- Attachment Plan + Artifact Movement
- Resource Limits + Lifecycle
- Failure Handling + Error Taxonomy
- Audit Payloads + Run-Surface Alignment

## Implementation Strategy
- Phase 1: finalize the private trusted broker-to-launcher control contract, the immutable evidence model, and the secure-session design before expanding implementation-specific code.
- Phase 2: wire a thin fake backend through the broker->launcher->broker path to prove lifecycle updates, evidence persistence, broker projection, and audit emission without QEMU complexity.
- Phase 3: implement the real Linux/QEMU/KVM backend behind the same contract, including secure session, attachment materialization, hardening enforcement, and fail-closed lifecycle handling.
- Phase 4: add a backend conformance suite plus KVM-backed end-to-end verification so later container, macOS, Windows, and alternative microVM implementations can reuse one reviewed contract and test matrix.

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
