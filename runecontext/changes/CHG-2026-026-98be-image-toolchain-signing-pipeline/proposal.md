## Summary
Establish one signed, immutable runtime-asset pipeline for RuneCode images and toolchains across Linux, macOS, and Windows, and across microVM and container backends, with fail-closed launch enforcement, offline-capable verified local caches, and audit-ready runtime identity.

## Problem
The current launcher slice already models digest-addressed runtime image identity, but the actual launch path is still provisional in ways that are not suitable as the long-term trust foundation:

- the microVM path still resolves boot assets from ambient host state rather than from published immutable runtime assets
- the current Linux vertical slice still builds an initramfs during launch instead of consuming a reviewed signed artifact
- signing metadata are required in contracts, but the runtime path does not yet cryptographically verify signed descriptor and component identity before launch
- runtime image identity, concrete boot-component identity, toolchain provenance, launch denial evidence, and later attestation hooks are not yet tied together into one durable contract

Without this change, later attestation, verification, Windows/macOS runtime support, and container parity would grow on top of a Linux-first provisional boot path that would need another semantics rewrite.

## Proposed Change
- Make published immutable runtime-image artifacts mandatory for normal RuneCode launch paths.
- Reuse the trusted detached-signature and verifier-record model as the authoritative runtime signing contract rather than inventing a second runtime-specific trust format.
- Define closed boot-contract profiles so backend-neutral runtime identity can stay stable while platform- or backend-specific realization details remain private launcher evidence.
- Treat toolchain signing as reviewed runtime-asset build and publication provenance, and remove ambient host build steps from normal launch.
- Add a trusted admission step that verifies signed runtime descriptors, component artifacts, and toolchain provenance before assets are admitted into a launcher-private verified cache.
- Make launch operate only on verified local immutable assets addressed by descriptor digest, with fail-closed enforcement on signer, descriptor, component, and compatibility mismatches.
- Record launch-allow and launch-deny verification outcomes through persisted runtime evidence and broker-owned audit surfaces.
- Keep runtime image identity distinct from validated project-substrate identity while preserving an explicit future binding point for cases where later evidence needs both.

## Why Now
This work lands in `v0.1.0-alpha.9` because both isolate attestation and the first usable end-to-end release need a stable signed runtime-identity contract before measured provisioning can be treated as durable assurance.

Landing this foundation before the beta cut avoids a split architecture where constrained devices and scaled deployments follow different trust or caching models, and it prevents Windows, macOS, and container follow-on work from inheriting Linux-specific launch assumptions as public contract semantics.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode owns the user-facing command and operator UX while invoking RuneContext capabilities under the hood where project context or assurance are involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.
- Trusted Go services remain the authoritative verification boundary; runner-side or external parity verification may exist, but it is not the launch trust root.
- The same reviewed runtime-asset architecture should apply on Raspberry Pi-class systems and on larger vertical or horizontal deployments, with scale changing cache and prewarming behavior rather than the architecture itself.

## Out of Scope
- Shipping full production Windows, macOS, or container runtime implementations in this change.
- Re-introducing legacy Agent OS planning paths as canonical references.
- Making live network access part of the normal launch trust path.
- Treating platform hypervisors, OCI runtimes, or other ambient host dependencies as the primary runtime signing trust root unless they are later promoted into reviewed signed runtime bundles.

## Impact
This change gives RuneCode one durable runtime-asset trust foundation:

- one authoritative signed runtime identity model
- one backend-neutral launch contract for microVM and container backends
- one topology-neutral performance and cache posture for constrained and scaled environments
- one future-proof identity seam for attestation and verified project-substrate binding

That foundation reduces supply-chain risk without weakening fail-closed behavior and prevents future backend or platform support from fragmenting the core product architecture.
