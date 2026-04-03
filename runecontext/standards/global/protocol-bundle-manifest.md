---
schema_version: 1
id: global/protocol-bundle-manifest
title: Protocol Bundle Manifest
status: active
suggested_context_bundles:
    - protocol-foundation
aliases:
    - agent-os/standards/global/protocol-bundle-manifest
---

# Protocol Bundle Manifest

Use `protocol/schemas/manifest.json` as the only inventory of checked-in protocol schemas and registries.

- Declare every checked-in schema and registry in the manifest
- Fail if the manifest references a missing file
- Fail if a checked-in schema/registry is missing from the manifest
- Fail if a manifest path escapes `protocol/schemas/`
- Keep Go/JS tooling keyed off the manifest; do not maintain side lists
