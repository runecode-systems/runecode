# References

## Product Context

- **Mission:** `runecontext/project/mission.md`
- **Roadmap:** `runecontext/project/roadmap.md`
- **Tech stack:** `runecontext/project/tech-stack.md`
- **Trust boundaries:** `docs/trust-boundaries.md`

## Related Changes

- `runecontext/changes/CHG-2026-009-1672-launcher-microvm-backend-v0/`
- `runecontext/changes/CHG-2026-026-98be-image-toolchain-signing-pipeline/`
- `runecontext/changes/CHG-2026-030-98b8-isolate-attestation-v0/`
- `runecontext/changes/CHG-2026-047-c3e2-local-control-plane-bootstrap-persistent-session-lifecycle-v0/`

## Governing Standards

- `runecontext/standards/product/roadmap-conventions.md`
- `runecontext/standards/security/trust-boundary-interfaces.md`
- `runecontext/standards/security/trust-boundary-layered-enforcement.md`
- `runecontext/standards/security/trusted-runtime-evidence-and-broker-projection.md`
- `runecontext/standards/security/audit-verification-scope-and-evidence-binding.md`
- `runecontext/standards/security/runtime-image-signing-admission-and-verified-cache.md`

## Current Code Seams

- `internal/launcherbackend/contract_handshake.go`
- `internal/launcherbackend/contract_secure_session.go`
- `internal/launcherbackend/contract_runtime_attestation_evidence.go`
- `internal/launcherbackend/contract_runtime_evidence.go`
- `internal/launcherdaemon/runtime_attestation_support.go`
- `internal/launcherdaemon/service.go`
- `internal/artifacts/store_runtime_facts.go`
- `internal/brokerapi/service_runtime_facts.go`

## Similar Implementations

- `runecontext/changes/CHG-2026-030-98b8-isolate-attestation-v0/` defines the attestation architecture this change finishes wiring into the live launch sequence.
