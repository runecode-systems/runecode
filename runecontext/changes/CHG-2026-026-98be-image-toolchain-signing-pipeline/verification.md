# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `go test ./internal/launcherbackend ./internal/launcherdaemon ./internal/brokerapi ./internal/trustpolicy`
- `just test`

## Verification Notes
- Confirm the change defines one topology-neutral runtime-asset architecture across constrained local devices and larger deployments rather than implying separate small-device and scaled-deployment trust paths.
- Confirm normal launch paths are planned to consume only published immutable runtime-image artifacts rather than ambient host kernels, mutable tags, or launch-time synthesized guest boot artifacts.
- Confirm the authoritative runtime signing contract reuses the trusted detached-signature and verifier-record model and does not introduce a second runtime-specific signing format.
- Confirm external release-signing or provenance systems remain additive publication evidence rather than the primary launch-time trust root.
- Confirm runtime-image signing, runtime-toolchain signing, and manifest signing remain separate logical authorities with explicit verifier-policy expectations.
- Confirm boot compatibility is defined through closed boot profiles rather than inferred only from backend kind.
- Confirm the first production microVM profile is planned around a kernel plus initrd boot contract and the first container profile around an OCI-image contract.
- Confirm launch is planned to operate from verified local immutable assets without requiring live network access.
- Confirm the design records trusted admission, launcher-private verified cache semantics, cache evidence, and fail-closed launch enforcement.
- Confirm the design includes persisted evidence and broker-owned audit for both launch-allow and launch-deny outcomes, including failures before session establishment.
- Confirm runtime image identity remains distinct from validated project-substrate identity even when later attestation, verification, or audit flows bind both.
- Confirm Linux, macOS, and Windows are all expected to build on the same runtime-asset contract without platform-specific public identity forks.
- Confirm microVM and container backends are both expected to build on the same runtime-asset contract without backend-specific public identity forks.
- Confirm the roadmap and change text both keep this work in `v0.1.0-alpha.9`.

## Close Gate
Use the repository's standard verification flow before closing this change.
