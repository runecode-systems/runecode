---
schema_version: 1
id: testing/protocol-fixture-manifest-parity
title: Protocol Fixture Manifest Parity
status: active
suggested_context_bundles:
    - protocol-foundation
aliases:
    - agent-os/standards/testing/protocol-fixture-manifest-parity
---

# Protocol Fixture Manifest Parity

Use `protocol/fixtures/manifest.json` as the shared contract for protocol fixtures.

- Put every shared schema, stream, runtime-invariant, and canonicalization fixture set in the manifest
- Make Go and JS iterate the same manifest-defined fixture IDs
- Keep positive and fail-closed cases in the same checked-in fixture set
- Capture runtime-only rules in fixtures when JSON Schema cannot express them
- CI verifies fixtures; it must not regenerate or mutate them implicitly
- No shared-fixture exceptions outside the manifest
