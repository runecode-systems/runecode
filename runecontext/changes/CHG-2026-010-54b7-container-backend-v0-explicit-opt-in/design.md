# Design

## Overview
Define the explicit reduced-assurance container backend, including instance-scoped opt-in semantics, hardened defaults, artifact movement, policy integration, audit/read-model projection, and reuse of the shared runtime contracts established by `CHG-2026-009-1672-launcher-microvm-backend-v0`.

## Key Decisions
- Containers are never a silent fallback; they require explicit opt-in and acknowledgment.
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
- Shared backend/runtime vocabulary should distinguish `degraded`, `unknown`, and `not_applicable` semantics where backend-specific mechanics differ, so container posture does not look degraded solely because it does not use microVM-specific transport or acceleration fields.
- Backend-neutral contracts should remain small and stable; backend-specific evidence should extend them through typed implementation evidence or evidence refs rather than by overloading microVM-only fields.

## Alpha.4 Callout Alignment
- Container Backend v0 remains follow-on hardening work and must not displace the primary microVM-backed secure path.
- Minimal TUI v0 remains a strict broker client and must not invent a container-only interaction model, approval truth, or audit truth.
- The first honest secure slice remains centered on explicit artifact handoff, audit capture + verify, signed policy decisions, and one real isolated backend with no trust-boundary shortcuts; container mode extends that same foundation rather than replacing it.

## Instance-Scoped Selection Model
- `container mode` names the backend posture for the active running RuneCode instance.
- For MVP one-user/one-machine operation, that instance-scoped posture is the narrowest reviewed selection model that still avoids per-run or per-role UX drift.
- The instance posture is selected through a canonical backend posture change action evaluated by policy and consumed through an exact-action approval.
- Existing active isolates are not retroactively migrated when the instance posture changes.
- New launches performed while the instance is in `container mode` use the container backend when compatible with the role and policy.
- Operator/client UX should treat this as an instance runtime posture change, not as a workspace preference or a run-local hidden flag.
- Future multi-instance scheduling or scaling may map this same logical selection model onto scheduler placement or instance pools, but the canonical posture semantics should remain topology-neutral.

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

## Backend Posture Action Contract
- The backend posture action should remain one canonical typed `ActionRequest` family, but its payload should be tightened from the current freeform posture string model to closed semantics that future backends can reuse.
- The reviewed foundation should move toward explicit fields for concepts such as:
  - target backend kind
  - selection mode (explicit selection vs automatic fallback attempt)
  - assurance change kind
  - opt-in kind / approval posture
- `requires_opt_in` should not be treated as the trust root; the authoritative approval binding is the signed exact-action approval for the canonical action hash.
- Automatic fallback attempts should remain representable so policy can deny them explicitly and audit the attempted behavior without relying on string matching.

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

## Main Workstreams
- Instance-Scoped Backend Selection Model
- Shared Backend Contract Alignment
- Generic Exact-Action Approval Consumption
- Opt-In UX + Audit
- Hardened Container Baseline
- No Host Mounts + Artifact Movement
- Policy Integration

## Scope And Sequencing
- Sequence container work after the primary microVM-backed secure path is in place.
- Start by tightening shared contracts, approval consumption, and posture vocabulary before growing container-runtime implementation details.
- Implement the first container controller only after the shared launcher/broker/policy/TUI foundation is explicit enough that the container path does not require special-case public behavior.
- Keep container v0 limited to offline workspace-role launches in the initial slice.
- Future gateway-role container runtime support should build as a follow-on change on top of the same reviewed backend-neutral contracts.

## Policy Integration Model
- Policy blocks container backend usage by default.
- Explicit instance-scoped container mode selection is represented as a canonical backend posture action and evaluated by the shared policy engine.
- Explicit opt-in should lead to an exact-action approval requirement under the shared approval model.
- Approval review should make the reduced-assurance effect explicit through structured broker-projected detail rather than through client-local prose.
- MicroVM launch failure must fail closed; it must never imply or trigger container mode.

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

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
