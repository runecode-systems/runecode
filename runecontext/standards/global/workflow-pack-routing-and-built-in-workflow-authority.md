---
schema_version: 1
id: global/workflow-pack-routing-and-built-in-workflow-authority
title: Workflow Pack Routing And Built-In Workflow Authority
status: active
suggested_context_bundles:
    - project-core
    - go-control-plane
    - protocol-foundation
---

# Workflow Pack Routing And Built-In Workflow Authority

When RuneCode exposes workflow-pack-backed session execution and product-shipped built-in workflows:

- Treat `SessionWorkflowPackRouting` as a broker-owned execution-selection and policy-binding contract rather than as a client-local hint.
- Keep TUI, CLI, chat, and autonomous entrypoints as thin adapters over the same broker-owned workflow-routing semantics; do not let one surface invent alternate workflow identity or binding rules.
- Treat product-shipped built-in workflow identities, workflow definition hashes, process definition hashes, and related catalog metadata as reviewed product authority; they are not repository-overridable.
- Keep `change_draft` and `spec_draft` artifact-first and non-mutation-bearing for the `v0` shared-workspace baseline; those routes must not smuggle mutation-bearing behavior through extra bound artifacts or client-local interpretation.
- Treat `draft_promote_apply` and `approved_change_implementation` as mutation-bearing shared-workspace routes for the `v0` baseline; at most one active mutation-bearing shared-workspace run may exist per authoritative repository root unless a later reviewed standard extends that posture explicitly.
- Require `approved_change_implementation` to bind exactly one authoritative `implementation_input_set` artifact and no unexpected extra bound artifact refs.
- Validate approved-implementation authority from exact reviewed bindings, including product-shipped workflow/process definition hashes and validated project-substrate digest, rather than from ambient repository state or client-local workflow assumptions.
- Keep overlap-admission, replay, and recovery classification fail-closed: unknown, missing, malformed, or internally inconsistent workflow-routing metadata must not silently downgrade a mutation-bearing route into an overlap-safe one.
