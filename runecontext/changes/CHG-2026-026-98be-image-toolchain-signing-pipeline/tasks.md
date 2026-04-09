# Tasks

## Signing Key Hierarchy

- [ ] Keep image and toolchain signing keys separate from manifest-signing keys.

## Runtime Image Descriptor Alignment

- [ ] Align image identity and signing targets with the digest-addressed `RuntimeImageDescriptor` model defined for the microVM backend.
- [ ] Ensure the descriptor can bind backend/platform compatibility, boot-contract version, concrete component digests, and future attestation/measurement hooks.

## Build + Publication Pipeline

- [ ] Define the build and publication pipeline for signed images and toolchains that emits a signed, digest-addressed runtime image descriptor.

## Launcher Enforcement

- [ ] Enforce runtime image descriptor and component signatures fail closed at launch time.
- [ ] Fail closed if the resolved boot components do not match the descriptor digest/signing identity expected by launch.

## Audit + Verification Integration

- [ ] Record descriptor digest, concrete component digests, signing posture, and enforcement outcomes in audit and verification surfaces.

## Acceptance Criteria

- [ ] Signed runtime image descriptors and toolchain artifacts reduce supply-chain risk without weakening fail-closed behavior.
