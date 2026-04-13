# Tasks

## Child Feature Tracking

- [ ] Track `CHG-2026-031-7a3c-secretsd-core-v0` to completion.
- [ ] Track `CHG-2026-032-4d1f-model-gateway-v0` to completion.
- [ ] Keep `CHG-2026-036-a4f9-secure-model-provider-access-v0` alignment current.

## Cross-Feature Contract Alignment

- [ ] Keep the typed `SecretLease` contract and `secret_access` lifecycle semantics aligned across the parent and child features, including issue, renew, revoke, consumer binding, and audit binding.
- [ ] Keep `LLMRequest`, `LLMResponse`, and `LLMStreamEvent` as the default canonical model boundary for downstream auth, bridge, and provider lanes.
- [ ] Keep `destination_ref`, gateway operation taxonomy, and `payload_hash` request binding aligned across the child features and shared policy foundations.
- [ ] Keep auth-gateway and model-gateway role separation explicit so downstream lanes do not reintroduce combined egress and credential roles.
- [ ] Keep broker-projected subsystem readiness and posture as the only long-lived operator-facing surface for secrets and gateway state.
- [ ] Keep the shared quota model aligned with token-metered providers and request-entitlement products.

## Cross-Feature Coordination

- [ ] Keep child feature sequencing aligned with `CHG-2026-007-2315-policy-engine-v0` and `CHG-2026-008-62e1-broker-local-api-v0` whenever new typed gateway, lease, readiness, or quota contracts are introduced.
- [ ] Keep child feature sequencing aligned with `CHG-2026-018-5900-auth-gateway-role-v0`.
- [ ] Keep child feature sequencing aligned with `CHG-2026-019-40c5-bridge-runtime-protocol-v0`.
- [ ] Keep child feature sequencing aligned with provider lanes (`CHG-2026-020-4425-openai-chatgpt-subscription-provider-oauth-codex-bridge`, `CHG-2026-022-8051-github-copilot-subscription-provider-official-runtime-bridge`).

## Acceptance Criteria

- [ ] Child features remain linked, sequenced, and aligned to the same security invariants.
- [ ] No child or downstream lane creates a second credential cache, a second policy/approval authority, or a second long-lived user-facing truth source for secrets or gateway posture.
- [ ] Parent-project status remains an accurate integration view rather than duplicating feature implementation detail.
