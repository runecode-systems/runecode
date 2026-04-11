---
schema_version: 1
id: global/protocol-schema-invariants
title: Protocol Schema Invariants
status: active
suggested_context_bundles:
    - protocol-foundation
---

# Protocol Schema Invariants

All top-level protocol objects use the same fail-closed baseline.

- Require `schema_id` and `schema_version`
- Keep runtime `schema_id` separate from schema-document `$id`
- Set `additionalProperties: false`
- Add explicit structural bounds (`maxProperties`, `maxItems`, `maxLength`, numeric limits)
- Add field descriptions
- Add `x-data-class` on boundary-visible fields: `public | sensitive | secret`
- No top-level exceptions
- Make policy-critical fields schema-required rather than optional when trusted evaluation depends on them; for example, gateway action payloads must require an explicit `operation` instead of relying on implementation-side defaulting
- Keep compiled planning and deterministic gate identity explicit in schema families; do not collapse gate identity to `gate_id` alone when `gate_kind` and `gate_version` are part of the executable contract
- For workflow/process/run-plan families, preserve the distinction between planning inputs (`WorkflowDefinition`, `ProcessDefinition`) and the immutable compiled execution contract (`RunPlan`)
