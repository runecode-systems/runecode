# Tasks

## Signed Runtime-Asset Contract

- [ ] Define one authoritative signed runtime-asset contract that reuses the trusted detached-signature and verifier-record model rather than introducing a second runtime-specific signing format.
- [ ] Define the canonical signed payload bytes for runtime-image identity explicitly so the signing input does not depend on ambiguous omit-a-field rules.
- [ ] Keep `RuntimeImageDescriptor` as the reviewed runtime-image identity family while clarifying how its signed payload, descriptor digest, and publication metadata relate.
- [ ] Define typed runtime-image and toolchain signature-bundle or publication-bundle metadata sufficient for trusted admission and offline-capable launch.

## Signing Authority + Verifier Policy

- [ ] Keep runtime-image signing and runtime-toolchain signing separate from manifest-signing keys and authority semantics.
- [ ] Keep runtime-image signing and runtime-toolchain signing separate from each other as distinct logical publisher authorities even if one automation system manages them.
- [ ] Define the trusted built-in verifier set and the reviewed import or rotation path for runtime signing authorities.
- [ ] Define signer admissibility, rotation, revocation, and compromise handling so new launches fail closed when runtime signing authorities are no longer valid.

## Boot-Profile Contract

- [ ] Turn `boot_contract_version` into a closed boot-profile contract rather than a vague backend string.
- [ ] Define the first production microVM boot profile as equivalent to `microvm-linux-kernel-initrd-v1`.
- [ ] Define the first production container boot profile as equivalent to `container-oci-image-v1`.
- [ ] Make component requirements and compatibility checks profile-driven rather than inferred only from `backend_kind`.
- [ ] Keep guest compatibility in runtime-image identity while leaving host realization capabilities such as KVM, HVF, WHPX, or OCI runtime selection as private launcher evidence.

## Toolchain Contract

- [ ] Define toolchain signing as reviewed runtime-asset build, assembly, and publication provenance for artifacts that materially determine launchable runtime bytes or enforcement decisions.
- [ ] Define when launcher-consumed helper artifacts must be promoted into the signed runtime-asset set instead of remaining ambient host dependencies.
- [ ] Remove ambient launch-time build steps from the normal reviewed launch path.

## Build + Publication Pipeline

- [ ] Make published immutable runtime-image artifacts mandatory for normal RuneCode launch paths.
- [ ] Emit signed runtime-image descriptor payloads, immutable component artifacts, and signed toolchain provenance where applicable.
- [ ] Ensure mutable tags, channels, or discovery helpers resolve to immutable signed identities before they can influence trusted launch.

## Trusted Admission + Verified Local Cache

- [ ] Define a trusted admission step that verifies descriptor identity, signer admissibility, boot-profile compatibility, component digests, and required toolchain provenance before assets are admitted into trusted local use.
- [ ] Add a launcher-private verified runtime-asset cache or equivalent immutable local store for admitted runtime assets.
- [ ] Define cache keys and cache evidence so warm launch paths reuse admitted verified assets without changing trust semantics.
- [ ] Keep launch offline-capable once assets are admitted locally.

## Launcher Enforcement

- [ ] Enforce runtime-image descriptor, signer, verifier, boot-profile, component-digest, and required toolchain-verification checks fail closed at launch time.
- [ ] Ensure launch operates only on verified local immutable assets rather than ambient host paths or mutable publication state.
- [ ] Fail closed if resolved boot components do not match the signed descriptor identity expected by launch.
- [ ] Remove the provisional ambient host-kernel and launch-time synthesized guest-boot path from normal operation once the reviewed signed asset path exists.
- [ ] Keep microVM launch failure fail-closed and never treat it as an implicit container fallback.

## Audit + Verification Integration

- [ ] Record requested and resolved descriptor digests, concrete component digests, signer identity, verifier identity, boot profile, cache posture, and enforcement outcome in persisted runtime evidence.
- [ ] Add broker-owned launch-admission or launch-denied audit surfaces for failures that occur before session establishment.
- [ ] Keep runtime image identity distinct from validated project-substrate identity while reserving an explicit binding seam for later evidence that needs both.
- [ ] Preserve hooks for later attestation or measurement evidence without requiring another runtime-image identity rewrite.

## Cross-Platform And Cross-Backend Stability

- [ ] Keep Linux, macOS, and Windows runtime support on the same logical runtime-asset pipeline and runtime-identity contract.
- [ ] Keep microVM and container backends on the same logical runtime-asset, evidence, and audit model.
- [ ] Keep platform-specific hypervisor, service-management, and OCI-runtime mechanics in private launcher realization layers or backend-specific evidence rather than public contract forks.

## Performance And Scaling Posture

- [ ] Keep one topology-neutral runtime-asset architecture for constrained devices and scaled deployments rather than introducing separate small-device and large-deployment architectures.
- [ ] Ensure verified-cache warm paths improve startup cost without weakening trust-boundary or fail-closed behavior.
- [ ] Align cold and warm runtime-asset behavior with the future launcher startup and warm-cache performance gates planned in `CHG-2026-053-9d2b-performance-baselines-verification-gates-v0`.

## Acceptance Criteria

- [ ] Normal RuneCode launch paths consume only published immutable signed runtime-image artifacts and reviewed toolchain provenance where applicable.
- [ ] Runtime-image signing, runtime-toolchain signing, and manifest signing remain separate logical authorities with explicit verifier-policy rules.
- [ ] Boot-profile compatibility and component requirements are defined through closed boot profiles rather than ambient backend heuristics.
- [ ] Launch works from verified local immutable assets without requiring live network access.
- [ ] Successful and denied launch admissions both produce persisted evidence suitable for audit and verification.
- [ ] Runtime image identity remains distinct from validated project-substrate identity even when later evidence binds both.
- [ ] Linux, macOS, and Windows can all build on the same runtime-asset contract without platform-specific public identity forks.
- [ ] MicroVM and container backends can both build on the same runtime-asset contract without backend-specific public identity forks.
- [ ] The runtime-asset architecture supports both constrained local devices and scaled deployments without introducing separate trust or launch architectures.
- [ ] Signed runtime-image descriptors and toolchain artifacts reduce supply-chain risk without weakening fail-closed behavior.
