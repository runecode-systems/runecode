# Protocol Schemas

- `protocol/schemas/manifest.json` is the authoritative bundle manifest for protocol object families and shared registries.
- `protocol/schemas/meta/manifest.schema.json` validates `protocol/schemas/manifest.json`.
- `protocol/schemas/meta/registry.schema.json` validates `protocol/schemas/registries/*.registry.json`.

## Status Semantics

- `mvp` means the object family is in MVP bundle scope. Some `mvp` families are intentionally minimal anchors until their owning spec task lands; those entries include a manifest `note` describing the pending task.
- `reserved` means the family is reserved for post-MVP extension work and must not expand capabilities without a later schema/task update.

## Schema Document IDs

- Object-schema `$id` values under `https://runecode.dev/protocol/schemas/...` are canonical schema identifiers for tooling and reference resolution.
- These `$id` values are not a network fetch contract. Validation and CI use the checked-in schema bundle as the source of truth.
