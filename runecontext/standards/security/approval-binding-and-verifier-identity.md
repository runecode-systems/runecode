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
- When exact-action approval gates an instance-scoped runtime posture change, require the bound immutable action inputs to include the targeted runtime `instance_id` so approvals for one launcher instance cannot be replayed against a later restarted instance
- Distinguish exact-action approval from stage sign-off at the binding layer:
  - exact-action approvals bind the canonical `ActionRequest` hash
  - stage sign-off approvals bind the canonical stage summary hash
- Treat the canonical trusted `ApprovalRecord` as the source of truth for approval lifecycle; runner-facing approval waits and other advisory summaries are derived mirrors, not a second approval authority
- When trusted services persist a canonical approval together with a runner-advisory mirror, perform that work in one atomic trusted store operation or fail closed with rollback that restores both in-memory and durable mirror state consistently
- When policy context participates in approval binding, require `manifest_hash` to mean the compiled effective policy-context hash rather than one raw source-manifest digest
- Treat `ApprovalBoundScope` and similar bound-scope summaries as operator-facing metadata only; do not accept them as substitutes for signed artifacts, request hashes, or stage-summary hashes
- Reject approvals when any bound action hash, stage-summary hash, or compiled policy-context hash has changed since request issuance, even if human-readable scope fields still appear to match
- Reject instance-scoped posture approvals when the currently active runtime `instance_id` no longer matches the signed approval request target, even if the requested backend kind or other human-readable scope fields still appear to match
- Reject approval artifacts when any bound request hash, approver identity, or relevant artifact digest does not match the current trusted runtime inputs
- Treat trust-surface `key_id_value` fields as canonical lowercase hex only; reject uppercase or non-canonical encodings even if they would otherwise decode successfully
