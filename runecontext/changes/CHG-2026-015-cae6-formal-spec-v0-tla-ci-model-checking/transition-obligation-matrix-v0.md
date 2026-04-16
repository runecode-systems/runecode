# Transition-Obligation Matrix (v0)

Closed `v0` authoritative-transition obligation matrix.

Each listed authoritative transition MUST have the required audit facts before authoritative state advances.

| Authoritative transition | Required audit facts (`v0`) |
| --- | --- |
| Approval request creation | canonical `approval_request_digest`; bound `{manifest_hash, action_request_hash/stage_summary_hash}`; requester principal identity; `requested_at`/`expires_at` |
| Approval decision acceptance | canonical `approval_decision_digest`; `approval_request_digest` linkage; approver principal identity; trusted verifier binding; `decided_at` |
| Approval consumption | canonical `approval_id`; accepted decision linkage; consumed continuation binding identity (`bound_action_hash` or `bound_stage_summary_hash`); `consumed_at` |
| Stage sign-off consumption | canonical `stage_summary_hash`; stage logical scope (`run_id`, `stage_id`); active `plan_id` binding when present; supersession outcome (`consumed` or `superseded`) |
| Gate result acceptance with canonical evidence linkage | gate identity tuple (`gate_id`, `gate_kind`, `gate_version`, `gate_attempt_id`); trusted plan placement (`plan_checkpoint_code`, `plan_order_index` when applicable); normalized input digests; canonical gate evidence reference |
| Gate override continuation | override action request hash; override policy decision digest; exact failed-result reference (`overridden_failed_result_ref`); gate identity + attempt binding; policy-context hash binding |
| Authoritative run terminal transition | run identity; terminal lifecycle state; terminal reason/evidence refs as available; terminal transition timestamp |
| Plan supersession / authoritative reconciliation | prior and next plan identities; supersession or reconciliation reason code; authoritative reconciliation timestamp; impacted run identity |

Notes:

- `v0` treats these as required evidence facts, not full ledger proof semantics.
- Unknown/missing required facts fail closed for authoritative state advance.
