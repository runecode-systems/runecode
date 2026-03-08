# Image/Toolchain Signing Pipeline — Post-MVP

User-visible outcome: isolate images and toolchains are reproducibly built and signed, and the launcher refuses to start roles whose images do not match pinned digests/signatures.

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-08-1039-image-toolchain-signing/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

## Task 2: Reproducible Build Definitions

- Define build inputs for guest images/toolchains.
- Ensure outputs are deterministic enough to support signature verification.

## Task 3: Signing + Pinning

- Define an image signing key separate from manifest signing.
- Pin expected digests/signatures in role manifests.

## Task 4: Boot-Time Enforcement

- Make the launcher verify image digest/signature before starting an isolate.
- Record image digests and signer identity in the audit log.

## Task 5: Update Strategy

- Define how security updates to images/toolchains are introduced and audited.

## Acceptance Criteria

- The system refuses to start isolates with unverified images.
- Image/toolchain updates are explicit and auditable.
