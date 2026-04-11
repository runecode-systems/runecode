---
schema_version: 1
id: security/policy-evaluation-foundations
title: Policy Evaluation Foundations
status: active
suggested_context_bundles:
    - go-control-plane
    - protocol-foundation
---

# Policy Evaluation Foundations

When trusted RuneCode services evaluate whether work is allowed, denied, or requires human approval:

- Compile one canonical effective policy context from fixed invariants, the active role manifest, the active run capability manifest, the active stage capability manifest when present, and signed allowlist artifacts referenced by that active manifest set
- Apply composition precedence in a fixed order: invariants first, then role envelope, then run capability scope, then stage capability scope, then referenced allowlist inputs; anything absent from the active signed context is denied fail-closed
- Treat `manifest_hash` as the digest of the compiled effective policy context, not as shorthand for one raw source manifest; keep contributing source manifest and allowlist hashes explicit in `policy_input_hashes`
- Evaluate one canonical typed `ActionRequest` contract and derive `action_request_hash` from canonical RFC 8785 JCS bytes of that contract
- Evaluate gateway egress actions only when the typed gateway payload is structurally complete, including an explicit `operation`; missing policy-critical fields must fail closed at schema and evaluation boundaries rather than degrade to optional semantics
- Fail closed on unknown `action_kind`, unknown action-payload schema IDs, unknown profile values, unknown role kinds, and unknown destination descriptor kinds
- Use fixed decision precedence `deny -> require_human_approval -> allow`
- Emit `PolicyDecision` for every successful evaluation, including deny outcomes; reserve the shared protocol `Error` envelope for failed evaluation or failed approval-consumption behavior
- Keep machine semantics split: `policy_reason_code` explains the policy outcome, `approval_trigger_code` explains why human approval is required, and `error.code` explains system-level failures
- Bind exact-action approvals to the canonical `ActionRequest` hash and bind stage sign-off approvals to the canonical stage summary hash; when any bound hash changes, require a new approval request rather than silently reusing the old one
- Treat `ApprovalBoundScope` and similar operator-facing scope summaries as derived UX metadata rather than a substitute for signed artifacts and bound hashes
- Keep the fixed high-blast-radius assurance floor in a taxonomy separate from approval triggers so user-involvement profiles can vary approval timing without redefining non-negotiable minimum assurance
- Route authorization semantics through one shared trusted policy engine boundary; component-local checks may validate structure or integrity, but must not invent competing allow/deny/approval semantics
- Keep trusted executor-binding and plan-compilation inputs separate from untrusted runner execution; evaluation may reason about executor posture and action contracts, but the runner must not become the authority for policy inputs, executor identity, or compiled gate semantics
- Evaluate public egress only through explicit gateway role-family actions using typed destination descriptors and signed allowlist inputs; do not treat raw URLs, transport identity, or ambient process context as policy authority
- Persist policy decisions as typed objects with stable digests and signed audit binding; do not add a separate policy-signing authority unless a later trust boundary concretely requires it
