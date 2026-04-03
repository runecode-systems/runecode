---
schema_version: 1
id: protocol-schema-bundle-v0
title: Protocol & Schema Bundle v0
originating_changes:
  - CHG-2026-001-57d6-agent-os-to-runecontext-migration-umbrella
revised_by_changes: []
---

# Protocol & Schema Bundle v0

## Summary

RuneCode uses a schema-validated, hash-addressable protocol bundle as the canonical trust-boundary contract for cross-component communication.

## Durable Current-State Outcomes

- `protocol/schemas/manifest.json` is the authoritative inventory for protocol schemas and registries.
- `protocol/fixtures/manifest.json` is the authoritative inventory for shared fixtures.
- Shared object families cover manifests, identities, approvals, policy decisions, artifacts/provenance, audit records, model request/response streaming, signed envelopes, and shared errors.
- Cross-language verification is implemented for Go and Node consumers against shared fixtures.
- Canonicalization and hashing posture is defined for deterministic verification workflows.

## Boundary Invariants

- Cross-boundary payloads are schema-driven and fail closed on unknown/unsupported structures.
- Shared registry discipline separates machine-consumed code families (error, policy reason, approval trigger, audit event type).
- Runner trust-boundary access remains limited to approved protocol schema/fixture surfaces.

## Related Standards

- `runecontext/standards/global/protocol-bundle-manifest.md`
- `runecontext/standards/global/protocol-schema-invariants.md`
- `runecontext/standards/global/protocol-registry-discipline.md`
- `runecontext/standards/global/protocol-canonicalization-profile.md`
- `runecontext/standards/testing/protocol-fixture-manifest-parity.md`
