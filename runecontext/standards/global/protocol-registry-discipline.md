---
schema_version: 1
id: global/protocol-registry-discipline
title: Protocol Registry Discipline
status: active
suggested_context_bundles:
    - protocol-foundation
---

# Protocol Registry Discipline

Keep machine-consumed protocol codes in separate registries and do not reuse code values across them.

- Use distinct registries for `error.code`, `policy_reason_code`, `approval_trigger_code`, `audit_event_type`, `audit_receipt_kind`, and `audit_verification_reason_code`
- Treat cross-registry code reuse as a fail-closed error
- Keep registry additions explicit and reviewable
- When seeded codes change, update checked-in fixtures and examples in the same change
- Do not encode human prose in registry values; keep messages/details elsewhere
