# Tasks

## Handshake Gating

- [ ] Wire reviewed secure-session validation into the launcher runtime lifecycle rather than leaving it as an unused trusted contract.
- [ ] Define the launcher state transition where secure session is considered validated for a live runtime.
- [ ] Prevent `ProvisioningPostureAttested` from being awarded before secure-session validation completes.

## Runtime-Side Proof Collection

- [ ] Define the backend-neutral post-handshake seam where runtime-side proof or attestation material is collected.
- [ ] Ensure collected proof binds to the validated live session tuple and not only to launcher-generated identifiers.
- [ ] Keep backend-specific raw proof details implementation-private while preserving a stable trusted verifier contract.

## Trusted Verification Ordering

- [ ] Move supported attestation success behind post-handshake trusted verification rather than receipt construction.
- [ ] Ensure trusted verification still binds to admitted signed runtime-image identity, boot profile, boot component digests, measurement profile, and freshness claims.
- [ ] Fail closed when handshake validation is absent, proof is missing, or verification fails.

## Persistence And Reconstruction

- [ ] Persist the post-handshake evidence and verification outputs needed for restart-safe reconstruction.
- [ ] Ensure broker authoritative posture continues to reconstruct from persisted evidence rather than launcher transient memory.
- [ ] Keep `attestation_evidence_digest` as the additive linkage seam for audit and projection.

## Audit And Operator Surfaces

- [ ] Ensure audit timing reflects that attestation is earned after secure-session validation and trusted verification, not at initial receipt construction.
- [ ] Keep operator surfaces able to distinguish:
  - session binding exists
  - runtime-side evidence exists
  - trusted attestation verification succeeded or failed
- [ ] Preserve clear distinction between isolation posture, coarse provisioning posture, and detailed attestation posture.

## Backends And Topology Neutrality

- [ ] Make microVM and container backends satisfy the same reviewed trust ordering.
- [ ] Preserve one architecture for constrained local devices and scaled deployments; only private optimizations may differ.
- [ ] Ensure any later caching or prewarming work does not bypass the required live handshake and post-handshake proof ordering.

## Verification

- [ ] Add tests proving launcher-side synthetic receipt fields alone cannot produce supported attested posture.
- [ ] Add tests proving secure-session validation is required before attestation success.
- [ ] Add tests proving post-handshake runtime-side evidence is required before attestation success.
- [ ] Add restart and broker-projection tests proving authoritative posture still reconstructs from persisted evidence.

## Acceptance Criteria

- [ ] Supported `attested` posture is impossible before secure-session validation.
- [ ] Supported `attested` posture is impossible before post-handshake runtime-side proof or attestation evidence is collected and trusted verification succeeds.
- [ ] Launcher-generated receipt fields alone are insufficient to claim supported attestation.
- [ ] Persisted evidence and verification outputs remain the authoritative restart-safe source for broker projection and audit linkage.
- [ ] MicroVM and container backends follow the same trust ordering without deployment-size-specific trust-path forks.
