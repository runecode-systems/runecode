# Design

## Overview
Use this change as the project-level tracker for secrets and model-gateway work while feature-level implementation lands in child changes.

The parent change owns the cross-feature architecture and trust-boundary posture that later auth, bridge, and provider features must inherit. Child features own runtime implementation detail, but they should not redefine the core contracts captured here.

## Key Decisions
- Child features own runtime implementation detail.
- Parent project owns sequencing, cross-feature contract alignment, and integration posture.
- Security invariants remain non-negotiable: deny-by-default egress, least-privilege role separation, leases-only secret use outside `secretsd`, typed and auditable control-plane contracts, fail-closed recovery, and local-only operational posture projected through the broker.
- No feature in this lane may introduce a second policy authority, a second approval authority, or a second long-lived operator-facing truth source for secrets or gateway posture.

## Cross-Feature Foundation Decisions

### Secret Custody

- `secretsd` is the only long-lived credential store in the system.
- A typed `SecretLease` family is the reusable contract for both persisted secrets and future derived short-lived tokens.
- Raw secret bytes are never a normal boundary-visible protocol object.
- Secret consumers outside `secretsd` use short-lived, scope-bound leases only.

### Gateway Role Separation

- `auth-gateway` is the only gateway role for auth-provider egress.
- `model-gateway` is the only gateway role for model-provider egress.
- No role may combine workspace access, public egress authority, and long-lived secret custody.
- Model traffic never performs auth exchange or refresh in place; auth traffic never becomes a back door to arbitrary model egress.

### Canonical Model Boundary

- `LLMRequest`, `LLMResponse`, and `LLMStreamEvent` remain the canonical typed model request, response, and stream families.
- Provider adapters and bridge/runtime specifics live below those typed objects unless a later feature proves that a new typed extension is required for policy, audit, or replay semantics.
- Provider-specific SDK payloads are not the canonical control-plane contract source of truth.

### Destination and Operation Identity

- `destination_ref` uses one canonical `host[:port][/path]` form with no scheme, query, fragment, or credentials.
- Gateway operations use one closed shared registry rather than per-feature ad hoc strings.
- Scope-change gateway operations stay distinct from request-execution operations.
- Request-execution gateway actions bind `payload_hash` to the canonical request object hash.

### Operator Posture and Quotas

- Daemon-local health remains a supervision surface rather than a user-facing public API.
- Broker projects operator-facing subsystem posture through one typed readiness summary rather than separate daemon-specific user APIs.
- Quota accounting uses one trusted abstraction that can model provider request limits, token limits, streamed-byte limits, concurrency caps, spend ceilings, and request-entitlement products such as premium requests.

## Main Workstreams
- `CHG-2026-031-7a3c-secretsd-core-v0`
- `CHG-2026-032-4d1f-model-gateway-v0`
- Cross-lane integration with auth/bridge/provider features
- Broker/operator posture alignment

## Sequencing Notes

- Freeze the shared secret-lease, gateway identity, and request-binding contracts in the child features before downstream auth, bridge, and provider lanes treat them as reusable dependencies.
- Keep `CHG-2026-018-5900-auth-gateway-role-v0` downstream of the reusable lease and role-separation foundation rather than allowing auth flows to redefine secret custody or model egress semantics.
- Keep `CHG-2026-019-40c5-bridge-runtime-protocol-v0` downstream of the canonical model boundary so bridge/runtime integrations stay beneath typed request and response contracts rather than replacing them.
- Keep provider-specific lanes downstream of the shared quota and destination-identity model so each provider does not invent its own egress identity and usage-accounting semantics.

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
