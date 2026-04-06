# References

## Product Context

- **Mission:** `runecontext/project/mission.md`
- **Tech stack:** `runecontext/project/tech-stack.md`
- **Roadmap:** `runecontext/project/roadmap.md`
- **Trust boundaries:** `docs/trust-boundaries.md`

## Related Specs And Changes

- `runecontext/specs/protocol-schema-bundle-v0.md`
- `runecontext/changes/CHG-2026-004-acdb-artifact-store-data-classes-v0/`
- `runecontext/changes/CHG-2026-005-cfd0-crypto-key-management-v0/`
- `runecontext/changes/CHG-2026-006-84f0-audit-anchoring-v0/`
- `runecontext/changes/CHG-2026-007-2315-policy-engine-v0/`
- `runecontext/changes/CHG-2026-008-62e1-broker-local-api-v0/`
- `runecontext/changes/CHG-2026-013-d2c9-minimal-tui-v0/`
- `runecontext/changes/CHG-2026-018-5900-auth-gateway-role-v0/`
- `runecontext/changes/CHG-2026-023-59ac-web-research-role/`
- `runecontext/changes/CHG-2026-024-acde-deps-fetch-offline-cache/`
- `runecontext/changes/CHG-2026-025-5679-external-audit-anchoring-v0/`
- `runecontext/changes/CHG-2026-027-71ed-workflow-concurrency-v0/`
- `runecontext/changes/CHG-2026-030-98b8-isolate-attestation-v0/`
- `runecontext/changes/CHG-2026-031-7a3c-secretsd-core-v0/`

## Applicable Standards

- `runecontext/standards/global/local-first-future-optionality.md`
- `runecontext/standards/global/protocol-canonicalization-profile.md`
- `runecontext/standards/global/protocol-registry-discipline.md`
- `runecontext/standards/security/trust-boundary-layered-enforcement.md`
- `runecontext/standards/security/trust-boundary-interfaces.md`
- `runecontext/standards/security/trusted-local-artifact-persistence.md`

## Similar Implementations

- Certificate Transparency style Merkle-sealed append-only logs are a useful structural analogue for segment sealing and inclusion proofs, but RuneCode keeps local-first, typed protocol objects and explicit trust-boundary rules rather than copying external log ecosystems directly.
