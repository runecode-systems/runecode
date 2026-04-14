# Design

## Overview
Define the explicit reduced-assurance container backend, including instance-scoped opt-in semantics, hardened defaults, artifact movement, policy integration, audit/read-model projection, and reuse of the shared runtime contracts established by `CHG-2026-009-1672-launcher-microvm-backend-v0`.

## Key Decisions
- Containers are never a silent fallback; they require explicit opt-in and acknowledgment.
- The selected runtime instance is a first-class trust-boundary identity. Backend posture actions and approvals must bind to the current launcher `instance_id`, not merely to a `run_id`.
- The active backend kind, runtime isolation assurance, provisioning/binding posture where applicable, and audit posture are treated as first-class operator/audit data and must not be collapsed into one generic “assurance” string.
- Container networking is isolated by default (no egress); any allowed egress is enforced via explicit network namespace + firewall/proxy rules, not convention.
- The container backend should reuse the same backend-neutral logical seams as the microVM backend where applicable, including launch intent, attachment planning, hardening posture recording, and terminal reporting, rather than inventing container-only control-plane contracts.
- `backend_kind` remains operator-facing and topology-neutral (`container`, not runtime implementation names such as Docker/Podman/runc).
- Container-specific runtime details remain implementation evidence, not public run identity.
- Reduced-assurance container opt-in should remain an exact explicit approval bound to canonical policy/action identity rather than a generic UI toggle or ambient workflow setting.
- The first container v0 implementation should cover offline workspace-role launches only; gateway-role container runtime support is a later scoped follow-on, not part of the initial container foundation.
- For this v0 change, `container mode` is selected for the active running RuneCode instance rather than per run, per stage, or per role instance.
- Restart should return the instance to the preferred primary microVM posture unless a later explicitly planned feature introduces reviewed durable operator policy for backend preference.
- The TUI/operator experience should remain the same across backends. Backend choice is surfaced through shared posture and approval/read-model cues, not through a second backend-specific UX flow.
- Container mode should feel operationally seamless to the user, but the reduced-assurance posture must remain unmissable in policy, audit, and operator surfaces.
- The trust root for container selection is the signed exact-action approval bound to the canonical backend posture action identity, not a client-local flag, sticky UI setting, or freeform acknowledgment string.
- The reviewed reduced-assurance step is the selection of `backend_kind=container` itself. Missing baseline container hardening controls are admission failures, not an acceptable second layer of degradation.
- Shared backend/runtime vocabulary should distinguish `degraded`, `unknown`, and `not_applicable` semantics where backend-specific mechanics differ, so container posture does not look degraded solely because it does not use microVM-specific transport or acceleration fields.
- Backend-neutral contracts should remain small and stable; backend-specific evidence should extend them through typed implementation evidence or evidence refs rather than by overloading microVM-only fields.
- The broker local API remains the only public/untrusted boundary. Backend posture selection must not introduce a direct TUI-to-launcher control path.
- The broker must own approval issuance and approval consumption side effects for backend posture changes. Launcher only applies trusted posture changes through the private broker-to-launcher contract.

## Foundation Corrections

The current branch audit shows the remaining closure work is not primarily “add a container runtime.” The stronger foundation requires the following corrections first:

- The backend posture model must become truly instance-scoped instead of run-scoped in policy, approval, and evidence paths.
- The broker must gain a real production approval-issuance path for backend posture approvals rather than relying on seeded test records.
- `ApprovalResolve` must become action-generic in request shape so backend posture approvals do not inherit promotion-specific fields.
- Backend posture application must become a typed broker-to-launcher side effect with compare-and-set binding to the current `instance_id`.

These corrections are part of this change, not follow-on polish. Without them, a container backend implementation would sit on an incorrect trust root.

## Alpha.4 Callout Alignment
- Container Backend v0 remains follow-on hardening work and must not displace the primary microVM-backed secure path.
- Minimal TUI v0 remains a strict broker client and must not invent a container-only interaction model, approval truth, or audit truth.
- The first honest secure slice remains centered on explicit artifact handoff, audit capture + verify, signed policy decisions, and one real isolated backend with no trust-boundary shortcuts; container mode extends that same foundation rather than replacing it.

## Instance-Scoped Selection Model
- `container mode` names the backend posture for the active running RuneCode instance.
- For MVP one-user/one-machine operation, that instance-scoped posture is the narrowest reviewed selection model that still avoids per-run or per-role UX drift.
- The launcher service mints a stable `instance_id` for the lifetime of the running instance.
- The instance posture is selected through a canonical backend posture change action evaluated by policy and consumed through an exact-action approval.
- The canonical backend posture action hash must include `target_instance_id` so approvals for one instance cannot be replayed against a later restarted instance.
- Approval review surfaces may display run/workspace context when helpful, but the trusted posture target is the runtime instance.
- Existing active isolates are not retroactively migrated when the instance posture changes.
- New launches performed while the instance is in `container mode` use the container backend when compatible with the role and policy.
- Operator/client UX should treat this as an instance runtime posture change, not as a workspace preference or a run-local hidden flag.
- Future multi-instance scheduling or scaling may map this same logical selection model onto scheduler placement or instance pools, but the canonical posture semantics should remain topology-neutral.

### Instance Identity Recommendation

- Add `target_instance_id` to the backend posture action payload and to typed approval detail for backend posture selection.
- Extend `ApprovalBoundScope` with optional `instance_id` as explanatory UX metadata only; the exact action hash remains the trust root.
- Record the selected `instance_id` in posture-selection evidence and in runtime launch evidence for runs started under that posture.
- Restart must mint a new `instance_id`; old backend-posture approvals become unusable against the restarted instance.

## Instance-Control Policy Model

- Backend posture changes should be evaluated by the shared policy engine, but not by pretending they are ordinary run-scoped runtime actions.
- The broker should add a dedicated trusted instance-control evaluation path built on the same canonical `ActionRequest` and signed trusted-context model used elsewhere.
- If a dedicated instance-control trusted context artifact is needed, it should remain generic control-plane context rather than container-specific configuration.
- Policy should still be able to explain denials, explicit opt-in requirements, and fallback-attempt denials through shared `PolicyDecision` semantics.

This preserves one policy engine and one action model while fixing the current run-scoped mismatch.

## Uniform UX And Approval Model
- The TUI should continue to use the same runs, approvals, artifacts, audit, and status routes regardless of backend.
- Backend choice should surface only through shared run posture, approval detail, and audit detail models.
- The operator should see:
  - `container mode` / `backend_kind=container`
  - reduced runtime isolation assurance
  - bound approval and policy references
  - effective hardening posture and degraded reasons
- The TUI should not expose a durable “always use containers” setting.
- The UI action that enables `container mode` should materialize as the same backend posture action evaluated by the shared policy engine and reviewed through the same approval inspection surfaces used elsewhere.
- The broker/TUI approval flow must remain generic exact-action approval review and consumption, not a container-specific approval pathway.

### Recommended Operator Flow

- The operator inspects current runtime posture through a broker-backed posture/status surface that shows `instance_id`, active backend, and reduced-assurance warnings.
- The operator chooses “switch active instance to container mode.”
- The TUI requires explicit reduced-assurance acknowledgment and submits a broker local API backend-posture change request.
- The broker either denies immediately, applies immediately when reverting to the safer path, or creates a pending approval through the shared approval model.
- The operator reviews and resolves that approval through the existing approvals route.
- On successful approval resolution, the broker applies the posture through the private launcher contract and refreshes the same shared read models.
- Returning to microVM should use the same operation family and the same shared UX surfaces.

## Backend Posture Action Contract
- The backend posture action should remain one canonical typed `ActionRequest` family, but its payload should be tightened from the current freeform posture string model to closed semantics that future backends can reuse.
- The reviewed foundation should move toward explicit fields for concepts such as:
  - target instance id
  - target backend kind
  - selection mode (explicit selection vs automatic fallback attempt)
  - assurance change kind
  - opt-in kind / approval posture
- `requires_opt_in` should not be treated as the trust root; the authoritative approval binding is the signed exact-action approval for the canonical action hash.
- Automatic fallback attempts should remain representable so policy can deny them explicitly and audit the attempted behavior without relying on string matching.

## Approval Issuance And Consumption Model

- Backend posture opt-in must use the same shared approval system as other exact-action approvals.
- The broker must convert a backend-posture `require_human_approval` policy decision into a canonical signed `ApprovalRequest` and persist a pending approval record atomically.
- Approval detail should explicitly describe:
  - `target_instance_id`
  - target backend kind
  - selection mode
  - assurance change kind
  - explicit reduced-assurance effect
  - “future launches only; existing isolates are unaffected” semantics
- `ApprovalResolve` should remain one logical operation, but action-specific resolution detail must move into typed sub-objects so backend posture changes do not inherit promotion-specific fields such as excerpt digests or repo paths.
- For backend posture approvals, successful resolution must verify the signed request and decision, then apply the posture through the private launcher contract, and only then mark the approval consumed.
- If posture application fails because the runtime instance changed, the approval must fail closed and become stale or superseded rather than being reused silently.

This keeps exact-action approval semantics shared while allowing action-specific side effects to remain typed and explicit.

## Public Broker Local API Shape

- Add a broker-owned backend posture API family rather than overloading run or readiness endpoints.
- `BackendPostureGet` should return at least:
  - active `instance_id`
  - launcher service state
  - active backend kind
  - preferred backend kind
  - whether reduced-assurance posture is active
  - whether a backend-posture approval is currently pending
  - latest applied posture-selection reference
  - whether container backend is currently available on the host
- `BackendPostureChange` should accept a typed backend posture request and return one of:
  - denied
  - pending approval created
  - applied
- The TUI should use only these broker local API operations and the existing approval APIs; it must not call launcher directly.

## Private Broker-To-Launcher Contract

- Backend posture application should be a typed private trusted operation between broker and launcher, not a public API.
- Add a private launcher operation equivalent to `ApplyInstanceBackendPosture` with compare-and-set binding to `instance_id`.
- The broker should pass the selected backend kind plus stable references to the exact approved action/evidence so launcher-applied state can be traced back to approval and policy records.
- Add a private query equivalent to `GetInstancePosture` so broker restart can reconstruct authoritative posture while the same launcher instance remains running.
- Transport realization may remain simple initially, but the logical service contract must stay stable and private to the trusted domain.

## Shared Runtime Contract Cleanup
- Reuse `BackendLaunchSpec`, `LaunchContext`, `AttachmentPlan`, `RuntimeImageDescriptor`, `BackendLaunchReceipt`, `AppliedHardeningPosture`, and `BackendTerminalReport` as the shared logical seams.
- Tighten the shared vocabulary where it still carries microVM-colored assumptions that would weaken container semantics if copied forward unchanged.
- Distinguish:
  - backend-neutral operator posture
  - backend-neutral runtime evidence
  - backend-specific implementation evidence
- Keep backend-specific implementation evidence behind typed evidence objects or evidence refs, not public run identity.

### Attachment Model Guidance
- Preserve logical attachment roles:
  - `launch_context`
  - `workspace`
  - `input_artifacts`
  - `scratch`
- Preserve the “no host filesystem mounts into isolates” invariant for containers as well as microVMs.
- Keep broker authorization at logical role + digest level only.
- Launcher realization remains backend-private and must not expose host-local paths, mount identities, or device numbering through public or shared operator contracts.
- Shared attachment vocabulary should be reviewed so container realization does not get forced into misleading microVM-only channel semantics.
- If a current shared field is truly backend-specific, it should be moved into backend-specific evidence rather than treated as universally meaningful.

### Attachment Realization Recommendation

- Preserve the current broker authorization boundary of logical role plus digest identity only.
- Realize `launch_context` and `input_artifacts` as digest-pinned read-only content.
- Realize `workspace` as isolated writable storage and `scratch` as ephemeral writable state.
- Keep no host bind mounts as a hard invariant for container mode as well as microVM mode.
- Prefer runtime-managed named volumes, image layers, tmpfs, or equivalent launcher-private realization mechanisms rather than host-path projections.
- Host-local paths, mount identities, and runtime-private storage layout must not appear in public schemas, run surfaces, audit payloads, or degraded-reason strings.

### Transport And Session Guidance
- The secure session and identity model should remain backend-neutral in purpose.
- Container realization must not be marked degraded solely because it does not use microVM-specific transport or acceleration mechanisms.
- Shared posture fields should allow `not_applicable` where appropriate, rather than overloading `degraded` or `unknown`.
- Backend-private transport realization must not become a second public runtime API.

### Evidence And Error Taxonomy Guidance
- Keep public/operator identity stable around `backend_kind`, runtime isolation assurance, provisioning posture, and audit posture.
- Keep shared runtime evidence stable around launch/session/hardening/terminal evidence.
- Backend-specific provenance such as QEMU or container runtime details should live in backend-specific evidence or evidence refs.
- Shared runtime error taxonomy should stay semantics-first and implementation-neutral where possible.
- Backend-specific detail may extend error evidence without leaking runtime implementation names into public run identity.

## Container Runtime Implementation

- Implement a real container controller behind the shared launcher contract instead of stopping at policy/read-model scaffolding.
- Backend routing should select microVM or container controller based on the requested backend kind without allowing silent substitution.
- Container v0 remains Linux-only, offline, and limited to workspace-role launches.
- Use a launcher-private OCI runtime adapter that supports strong rootless execution and reviewed hardening controls; runtime implementation names remain backend-private evidence.
- Container backend implementation must not create a second public runtime API or bypass broker artifact authorization.

## Hardened Container Baseline

- The v0 container backend must treat the following as admission requirements:
  - rootless execution
  - `no_new_privs`
  - seccomp filtering
  - dropped Linux capabilities
  - read-only root filesystem
  - ephemeral writable layers
  - no host bind mounts
  - default no network egress for workspace-role launches
- If these controls cannot be enforced, container launch fails closed.
- `AppliedHardeningPosture` should record what was actually enforced, but missing required baseline controls should normally produce launch failure rather than a weaker successful launch.
- The backend-specific evidence may record private runtime policy identifiers, sandbox profile identifiers, or runtime provenance so long as host-local paths and runtime implementation names do not leak into public run identity.

### Networking Recommendation

- Workspace-role container v0 should default to no network connectivity or loopback-only semantics.
- If future reviewed work grants egress, enforcement must occur through explicit host-level or namespace-level controls rather than in-container convention.
- Gateway-role container networking is explicitly out of scope for this change.

## Main Workstreams
- Instance Identity + Selection Binding
- Instance-Control Policy Path
- Generic Approval Issuance + Generic Approval Resolve Shape
- Backend Posture Broker Local API
- Private Broker-To-Launcher Posture Contract
- Shared Backend Contract Alignment
- Container Runtime Controller
- Hardened Container Admission Baseline
- No Host Mounts + Explicit Artifact Movement
- Opt-In UX + Audit + Run-Surface Alignment
- Conformance + End-To-End Verification

## Scope And Sequencing
- Sequence container work after the primary microVM-backed secure path is in place.
- Start by fixing the instance-identity trust root, instance-control policy path, approval issuance, and approval resolution shape before growing container-runtime implementation details.
- Implement the first container controller only after the shared launcher/broker/policy/TUI foundation is explicit enough that the container path does not require special-case public behavior.
- Keep container v0 limited to offline workspace-role launches in the initial slice.
- Future gateway-role container runtime support should build as a follow-on change on top of the same reviewed backend-neutral contracts.

### Recommended Execution Order

1. Fix instance-scoped trust root and action binding.
2. Add instance-control policy evaluation and backend-posture approval issuance.
3. Repair generic `ApprovalResolve` request shape and backend-posture side effects.
4. Add broker local API posture operations and private launcher posture application.
5. Implement container controller and shared backend routing.
6. Enforce hardened container baseline and host-mount-free attachment realization.
7. Complete TUI flow, audit/event alignment, and run-surface linkage.
8. Add conformance and end-to-end verification before closure.

## Policy Integration Model
- Policy blocks container backend usage by default.
- Explicit instance-scoped container mode selection is represented as a canonical backend posture action and evaluated by the shared policy engine.
- Explicit opt-in should lead to an exact-action approval requirement under the shared approval model.
- Approval review should make the reduced-assurance effect explicit through structured broker-projected detail rather than through client-local prose.
- MicroVM launch failure must fail closed; it must never imply or trigger container mode.
- Instance-control policy evaluation must not depend on synthetic run identity. It should use a dedicated trusted instance-control context while preserving the same shared policy engine and canonical action hash semantics.

## Audit And Operator Surface Model
- Shared broker run surfaces should project at least:
  - `backend_kind`
  - runtime isolation assurance
  - provisioning/binding posture where applicable
  - audit posture
  - bound approval/policy references for reduced-assurance selection
  - hardening posture summary and degraded reasons
- The reduced-assurance posture should be machine-visible and operator-visible, not just described in logs.
- Audit payloads should stay small and reference-heavy, pointing to launcher-produced evidence and approval/policy identities.
- A container-backed run should still read as “same system, reduced runtime-isolation assurance,” not as a separate execution product.

### Backend Posture Audit Recommendation

- Add a broker-owned audit event for backend posture application itself, distinct from later run launches.
- That event should reference at least:
  - `instance_id`
  - target backend kind
  - exact action hash
  - policy decision hash
  - approval request digest
  - approval decision digest
  - posture-selection record or evidence ref
- Runtime launch evidence for a run launched under container mode should carry the applied posture-selection reference so later audit and run-detail views can link a run back to the approved instance posture.

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
