# Design

## Overview
Define one signed runtime-asset pipeline for isolate images and toolchains with fail-closed enforcement, aligned to the digest-addressed runtime image model established by `CHG-2026-009-1672-launcher-microvm-backend-v0` and suitable for future Linux, macOS, and Windows runtimes and for both microVM and container backends.

The intended architecture is:

`publish -> sign -> trusted admission -> verified local cache -> launch -> persisted evidence -> broker projection`

That flow should remain the same whether RuneCode is running on a constrained local device or on a larger scaled deployment. Scaling may add mirrors, prefetch, warm caches, or warm pools as private implementation details, but it must not introduce a second architecture or a weaker trust path.

## Key Decisions
- Image and toolchain signing keys are separate from manifest-signing keys and remain separate logical publisher authorities even if one automation system manages them.
- Published immutable runtime-image artifacts are mandatory for normal launch paths.
- Enforcement is fail-closed, including trusted admission, cache admission, launch admission, and launch execution.
- Signing and launch enforcement operate on a digest-addressed `RuntimeImageDescriptor` and typed signed runtime-asset payloads rather than mutable tags, loose file paths, or ad hoc per-platform boot references.
- The authoritative runtime signing contract should reuse the trusted detached-signature and verifier-record model already used by the control plane rather than introducing a second runtime-specific signing format.
- External release-signing and provenance systems such as Sigstore or cosign may remain additive publication evidence, but they are not the authoritative launch-time trust root.
- Verification and audit should record both the descriptor digest and the concrete boot component digests actually used or denied.
- `boot_contract_version` should become a closed boot-profile contract rather than a vague per-backend string. The first production microVM profile should be equivalent to `microvm-linux-kernel-initrd-v1`, and the first container profile should be equivalent to `container-oci-image-v1`.
- Guest compatibility should remain part of runtime-image identity, while host realization capabilities such as KVM, HVF, WHPX, or the selected OCI runtime remain launcher-private capability checks and implementation evidence.
- Toolchain signing should cover reviewed runtime-asset build and publication provenance; normal launch must not depend on ambient host build steps such as compiling guest boot artifacts during launch.
- Launch should work from verified local assets without requiring live network access.
- Runtime image identity remains distinct from validated project-substrate identity; later verification, audit, or attestation flows may bind both, but this change must not collapse runtime identity into project-context identity.

## Main Workstreams
- Signed Runtime-Asset Contract
- Signing Authority + Verifier Policy
- Boot-Profile Contract
- Build + Publication Pipeline
- Trusted Admission + Verified Local Cache
- Launcher Enforcement
- Audit + Verification Integration
- Cross-Platform and Cross-Backend Contract Stability
- Performance and Scaling Posture

## Governing Principles

### One Topology-Neutral Architecture
- Constrained local devices and scaled deployments must use the same logical runtime-asset architecture.
- Scaling may change cache population, warm-pool behavior, or artifact distribution mechanics, but not trust semantics, launch semantics, or audit semantics.

### Trusted-Domain Authority
- Trusted Go services are the authoritative sign/verify boundary.
- Runner-side or external tooling may provide parity checks or publication-time assurance, but they must not become the launch trust root.

### Backend-Neutral Public Contracts
- Public and shared control-plane contracts should stay stable around runtime-image identity, compatibility, launch evidence, hardening posture, lifecycle, and audit posture.
- Platform-specific hypervisor, IPC, package layout, or container-runtime mechanics remain private launcher realization details or backend-specific evidence.

### Offline-Capable Launch
- Normal launch should not depend on live registry, CDN, or verifier-service access.
- Trusted admission may happen at fetch, install, update, or staged-prewarm time, but launch must operate from verified local state.

## Signed Runtime-Asset Contract

### Authoritative Contract Model
- The change should define typed signed runtime-asset payloads whose canonical bytes are signed and verified using the same detached-signature and verifier-record model already used in the trusted control plane.
- The signed payload should represent runtime identity directly rather than relying on ambiguous rules such as "sign the object except its digest field".
- `RuntimeImageDescriptor` remains the reviewed runtime identity object family for launch and audit surfaces, but the signing model should operate on one explicit canonical payload contract so the signed bytes are obvious and stable.

### Recommended Artifact Families
The planned foundation should distinguish the following logical artifact families even if initial implementation names differ:

- signed runtime-image descriptor payload
- signed runtime-toolchain descriptor or manifest payload
- immutable boot component artifacts addressed by digest
- immutable toolchain bundle artifacts addressed by digest
- publication bundle or signature bundle metadata that maps signed identity to immutable published artifacts and provenance material

The core contract is that runtime identity is defined by signed canonical payload bytes and immutable component digests, not by mutable registry tags, mutable release channels, or host-local file references.

## Signing Authority + Verifier Policy

### Separate Logical Authorities
- Runtime-image signing and runtime-toolchain signing must remain separate logical publisher authorities even if they share the same release automation.
- Manifest-signing authorities remain distinct from both.
- The implementation may extend current authority vocabulary or otherwise enforce this separation through key namespace, verifier policy, and audit identity, but the separation itself is a fixed design requirement.

### Verifier Distribution
- The trusted product should ship with a reviewed built-in verifier set for runtime-image and runtime-toolchain signing.
- Additional verifier records or rotations may be imported through trusted reviewed mechanisms, but launch must not depend on online verifier discovery.
- Rotation, revocation, and signer compromise handling must be first-class. New launches fail closed when the signer or verifier state is no longer admissible.

### External Publication Provenance
- Sigstore, cosign, GitHub Actions provenance, or equivalent external publication evidence may remain part of release and operator verification workflows.
- Those external systems should be treated as additive distribution and publication provenance, not as the primary launch-time authorization path.

## Boot-Profile Contract

### Closed Boot Profiles
- `boot_contract_version` should become a closed boot-profile contract that drives component requirements, compatibility checks, and launch realization rules.
- Component requirements must be profile-driven, not inferred only from `backend_kind`.

### Initial Profiles
- The first production microVM profile should be equivalent to `microvm-linux-kernel-initrd-v1`.
- The first production container profile should be equivalent to `container-oci-image-v1`.
- Future profiles such as disk-backed microVM or different guest boot chains should be added as new closed profiles rather than silently redefining existing ones.

### Host Versus Guest Compatibility
- Guest OS, guest architecture, and other runtime-image compatibility claims belong in the runtime-image descriptor.
- Host-specific realization capabilities such as KVM, HVF, WHPX, or container-runtime implementation choice remain private capability checks and evidence, not part of the published runtime identity.

## Toolchain Contract

### Toolchain Scope
- Toolchain signing should cover the reviewed build, assembly, and publication toolchain that materially determines the signed runtime-image bytes or runtime-asset identity.
- Ambient host tooling or host-provided launch prerequisites should not become the primary signed toolchain trust root unless later reviewed work explicitly promotes them into signed runtime bundles.

### Launch-Path Consequence
- Normal launch must not synthesize guest boot artifacts from ambient host compilers or other ad hoc build steps.
- The current provisional pattern of constructing guest boot artifacts during launch is a migration target to remove, not a reviewed long-term runtime architecture.

## Build + Publication Pipeline

### Mandatory Published Immutable Runtime Assets
- Normal RuneCode launch should consume only published immutable runtime-image artifacts and toolchain artifacts.
- Publication should emit immutable artifact bytes addressed by digest together with the signed runtime-image and toolchain identity payloads.

### Publication Outputs
The pipeline should emit at least:

- a signed runtime-image descriptor payload
- immutable component artifacts referenced by digest
- a signed runtime-toolchain descriptor or manifest when toolchain provenance is part of the reviewed launch chain
- typed publication metadata or signature-bundle material sufficient for trusted admission and operator verification

### No Mutable Trust Inputs
- Mutable tags, latest pointers, or host paths may be used only as convenience discovery inputs outside the trust root.
- The trusted path resolves them to immutable signed and digest-addressed identities before admission.

## Trusted Admission + Verified Local Cache

### Admission Stage
- Before launchable assets are accepted into local trusted use, the trusted domain should verify signed descriptor and toolchain identity, signer admissibility, boot-profile compatibility, and declared component digests.
- Assets that fail verification are never admitted to the verified runtime cache.

### Verified Local Cache
- The launcher should maintain a launcher-private verified runtime-asset cache or equivalent immutable store for admitted runtime assets.
- Cache keys should bind at least descriptor digest, boot profile, backend kind, and guest platform compatibility.
- Launch should resolve assets from this verified local cache rather than reinterpreting mutable publication state on every run.

### Scale Without Architecture Forks
- Small devices may rely on node-local verified cache only.
- Larger deployments may add mirrored fetch, staged prewarming, or warm pools.
- All of those remain private optimizations beneath the same trusted admission and verified-cache model.

## Launcher Enforcement

### Launch Admission State Machine
Launch should follow one strict state machine:

1. resolve the requested runtime-image descriptor by immutable identity
2. verify signer identity and verifier admissibility
3. verify boot-profile compatibility and declared component set
4. resolve immutable component artifacts from the verified cache
5. compare concrete component digests against the signed descriptor
6. verify required signed toolchain provenance when it materially affects launchable bytes or enforcement decisions
7. launch only when all checks succeed

### Launch Failure Policy
- Missing signature material, missing verifier state, revoked signers, component mismatches, incompatible boot profiles, incompatible platform claims, or unverifiable toolchain state fail closed.
- MicroVM launch failure must never imply container fallback.
- Container launch remains an explicit backend posture selection governed by its own reviewed posture and approval path.

### Normal Runtime Path Cleanup
- The long-term normal launcher path should no longer resolve host kernels from ambient `/boot` state or build guest boot artifacts during launch.
- Those provisional mechanics may remain temporarily for migration or tests, but they are not the reviewed target architecture.

## Audit + Verification Integration

### Persisted Runtime Evidence
- Persisted runtime evidence should record:
  - requested descriptor digest
  - resolved descriptor digest
  - declared and concrete component digests
  - signer identity and verifier identity
  - boot profile
  - toolchain identity when applicable
  - cache posture and cache result
  - enforcement outcome and fail-closed reason code

### Launch-Allow And Launch-Deny Surfaces
- The design should include evidence and audit for both successful launch admission and denied launch admission.
- Session-oriented runtime audit events are not enough for signature or component failures that happen before session establishment.
- Broker-owned audit surfaces should therefore include a reviewed launch-admission or launch-denied event family, backed by persisted evidence rather than logs or stderr scraping.

### Verification Semantics
- Verification flows should be able to recompute and confirm descriptor identity, component identity, signer admissibility, and enforcement outcome from persisted evidence.
- Audit payloads should stay reference-heavy and must not expose host-local paths, hypervisor argv, runtime-private cache paths, or mutable package-manager metadata as trust identity.

## Runtime Identity, Attestation, And Project Identity

### Runtime Identity Stack
Runtime identity should mean:

- runtime-image descriptor identity
- concrete boot-component identity
- signer and verifier identity
- later attestation or measurement evidence where applicable

### Project Identity Stack
Validated project-substrate identity remains a separate stack defined by the verified RuneContext project substrate.

### Binding Rule
- The design should reserve one explicit binding point for cases where later evidence needs both runtime identity and validated project identity.
- It must not overload runtime-image fields with project-substrate meaning or vice versa.

## Cross-Platform And Cross-Backend Stability

### Shared Public Semantics
- Linux, macOS, and Windows runtimes should share the same logical runtime-image descriptor, boot-profile, admission, launch-evidence, and audit semantics.
- MicroVM and container backends should share the same logical runtime-asset pipeline and runtime-identity model.

### Private Realization Differences
- QEMU plus KVM, HVF-backed QEMU, WHPX or Hyper-V realization, and launcher-private OCI runtime realization remain backend-private or platform-private implementation mechanics.
- Those differences may appear in backend-specific evidence, but not as forks in the public runtime identity contract.

## Performance And Scaling Posture

### Performance Goals
- Launch should avoid repeated expensive recomputation when verified local assets are already admitted.
- Verified-cache hits should be the normal warm path.
- The architecture must support deterministic offline operation and bounded resource use on constrained nodes while still enabling prewarming and warm-pool optimizations on larger deployments.

### Performance Constraints
- Performance work must not weaken trust-boundary or fail-closed behavior.
- No optimization may bypass signature verification, verifier admissibility, component-digest checks, or audit evidence generation.
- The same reviewed runtime-asset architecture should underlie future CHG-053 performance gates for cold and warm launcher startup.

## Recommended Implementation Order

1. Define the signed runtime-asset payload contract and verifier-policy rules.
2. Define closed boot profiles and toolchain-scope rules.
3. Define trusted admission and launcher-private verified cache semantics.
4. Replace provisional ambient launch inputs with published immutable runtime assets in the normal path.
5. Add launch fail-closed enforcement and denied-launch evidence.
6. Bind runtime identity cleanly into later attestation and verified project-substrate surfaces without collapsing identities.
7. Add cold and warm performance checks once the normal trusted path is stable.

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
