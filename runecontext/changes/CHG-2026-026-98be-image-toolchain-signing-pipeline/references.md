# References

## Product Context

- **Mission:** `runecontext/project/mission.md`
- **Tech stack:** `runecontext/project/tech-stack.md`
- **Trust boundaries:** `docs/trust-boundaries.md`
- **Release signing and publication:** `docs/release-process.md`
- **Install-time release verification:** `docs/install-verify.md`

## Related Specs

- `runecontext/changes/CHG-2026-009-1672-launcher-microvm-backend-v0/`
- `runecontext/changes/CHG-2026-010-54b7-container-backend-v0-explicit-opt-in/`
- `runecontext/changes/CHG-2026-005-cfd0-crypto-key-management-v0/`
- `runecontext/changes/CHG-2026-004-acdb-artifact-store-data-classes-v0/`
- `runecontext/changes/CHG-2026-030-98b8-isolate-attestation-v0/`
- `runecontext/changes/CHG-2026-046-a91d-runecontext-verified-project-substrate-compatibility-lifecycle-v0/`
- `runecontext/changes/CHG-2026-028-647e-windows-microvm-runtime-support/`
- `runecontext/changes/CHG-2026-029-5e5e-macos-virtualization-polish/`
- `runecontext/changes/CHG-2026-053-9d2b-performance-baselines-verification-gates-v0/`

## Similar Implementations

- Existing release publication and additive provenance path: `.github/workflows/release.yml`
- Existing trusted signing and verifier foundations: `internal/trustpolicy/`, `internal/secretsd/`
- Existing launcher runtime-identity and evidence contracts: `internal/launcherbackend/`
