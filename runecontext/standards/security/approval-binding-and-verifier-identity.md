---
schema_version: 1
id: security/approval-binding-and-verifier-identity
title: Approval Binding And Verifier Identity
status: active
suggested_context_bundles:
    - go-control-plane
    - protocol-foundation
---

# Approval Binding And Verifier Identity

When trusted RuneCode services accept signed approval artifacts for promotion or other policy-gated actions:

- Require both the signed approval request artifact and the signed approval decision artifact when the decision claims to resolve a prior request
- Verify the signed approval request and signed approval decision against trusted verifier records before consuming their payloads
- Require the trusted verifier selected for an approval decision to have `approval_authority` purpose and `user` scope
- Fail closed unless the trusted verifier's `owner_principal` exactly matches the approval decision `approver` identity
- Treat `approval_request_hash` as a binding to the canonical approval request payload bytes, not ad-hoc local serialization or unsigned ambient context
- Require approval-request action binding fields to cover the immutable action inputs and the exact relevant artifact digests for the action being approved
- Reject approval artifacts when any bound request hash, approver identity, or relevant artifact digest does not match the current trusted runtime inputs
- Treat trust-surface `key_id_value` fields as canonical lowercase hex only; reject uppercase or non-canonical encodings even if they would otherwise decode successfully
