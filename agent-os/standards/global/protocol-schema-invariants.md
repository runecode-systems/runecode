# Protocol Schema Invariants

All top-level protocol objects use the same fail-closed baseline.

- Require `schema_id` and `schema_version`
- Keep runtime `schema_id` separate from schema-document `$id`
- Set `additionalProperties: false`
- Add explicit structural bounds (`maxProperties`, `maxItems`, `maxLength`, numeric limits)
- Add field descriptions
- Add `x-data-class` on boundary-visible fields: `public | sensitive | secret`
- No top-level exceptions
