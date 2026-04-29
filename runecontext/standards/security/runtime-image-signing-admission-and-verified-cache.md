---
schema_version: 1
id: security/runtime-image-signing-admission-and-verified-cache
title: Runtime Image Signing Admission And Verified Cache
status: active
suggested_context_bundles:
    - go-control-plane
    - protocol-foundation
---

# Runtime Image Signing Admission And Verified Cache

When trusted launcher and broker paths handle launchable runtime assets:

- Treat signed runtime-image identity, signed runtime-toolchain identity, and admitted local runtime assets as one reviewed trust chain rather than separate optional features
- Keep runtime-image signing and runtime-toolchain signing as separate logical authorities with distinct verifier-policy kinds; do not collapse them into one interchangeable verifier namespace
- Reuse the canonical detached-signature and verifier-record model for runtime signing; do not introduce ad hoc runtime-only signing formats or unsigned side channels
- Keep runtime-image and toolchain payload identity canonical, typed, and digest-addressed; mutable host paths, mutable tags, and package-manager state must not become trusted runtime identity
- Import, persist, and validate verifier-authority state as trusted local control-plane state, with kind-specific admissibility checks and fail-closed handling for malformed or stale authority data
- Perform trusted admission before launch: verify signer admissibility, canonical payload identity, boot-profile compatibility, declared component digests, and required toolchain provenance before assets are treated as launchable
- Resolve normal launches from launcher-private verified local assets after admission; do not re-interpret mutable publication state or ambient host paths on each launch
- Keep the verified runtime cache private to the trusted launcher domain and keyed by immutable digests; cache hits may optimize launch cost but must not weaken the same admission semantics as cold paths
- Hello-world, demo, and operator-validation flows must stay on the same signed-admission and verified-cache architecture unless an explicit reviewed exception says otherwise
- Launch must fail closed when signature material, verifier authority, admitted assets, or required toolchain evidence is missing, revoked, malformed, or inconsistent with the requested runtime identity
- Keep runtime-image identity distinct from project-substrate identity even when later evidence binds both in audit or attestation surfaces
- Tests should cover authority import and normalization, kind-specific verifier admissibility, admitted-cache warm paths, launch-denied behavior, and cross-platform compile parity without introducing weaker trust semantics on non-Linux platforms
