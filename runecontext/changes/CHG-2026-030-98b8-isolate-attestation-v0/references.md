# References

## Product Context

- **Mission:** `runecontext/project/mission.md`
- **Roadmap:** `runecontext/project/roadmap.md`
- **Tech stack:** `runecontext/project/tech-stack.md`
- **Trust boundaries:** `docs/trust-boundaries.md`

## Related Specs

- `runecontext/changes/CHG-2026-005-cfd0-crypto-key-management-v0/`
- `runecontext/changes/CHG-2026-009-1672-launcher-microvm-backend-v0/`
- `runecontext/changes/CHG-2026-003-b567-audit-log-v0-verify/`
- `runecontext/changes/CHG-2026-026-98be-image-toolchain-signing-pipeline/`
- `runecontext/changes/CHG-2026-046-a91d-runecontext-verified-project-substrate-compatibility-lifecycle-v0/`
- `runecontext/changes/CHG-2026-053-9d2b-performance-baselines-verification-gates-v0/`

## Governing Standards

- `runecontext/standards/security/trust-boundary-interfaces.md`
- `runecontext/standards/security/trust-boundary-layered-enforcement.md`
- `runecontext/standards/security/trusted-runtime-evidence-and-broker-projection.md`
- `runecontext/standards/security/audit-verification-scope-and-evidence-binding.md`
- `runecontext/standards/security/trusted-local-artifact-persistence.md`
- `runecontext/standards/security/runtime-image-signing-admission-and-verified-cache.md`
- `runecontext/standards/global/control-plane-api-contract-shape.md`
- `runecontext/standards/global/local-first-future-optionality.md`
- `runecontext/standards/global/project-substrate-contract-and-lifecycle.md`

## Current Code Seams

- `internal/launcherbackend/contract_launch_session.go`
- `internal/launcherbackend/contract_handshake.go`
- `internal/launcherbackend/contract_runtime_evidence.go`
- `internal/launcherbackend/contract_runtime_image.go`
- `internal/launcherbackend/contract_runtime_admission_record.go`
- `internal/launcherdaemon/runtime_asset_cache.go`
- `internal/trustpolicy/foundation_runtime.go`
- `internal/brokerapi/service_runtime_facts_audit.go`
- `internal/brokerapi/service_runtime_facts.go`
- `internal/artifacts/store_runtime_facts.go`

## Similar Implementations

None yet. This change defines the first reviewed isolate-attestation contract for RuneCode.
