# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `go test ./internal/protocolschema`
- `just test`

## Verification Notes
- Confirm the split preserves model-gateway trust-boundary requirements from the prior combined change.
- Confirm provider-facing features reference this gateway feature for egress controls.
- Confirm `LLMRequest`, `LLMResponse`, and `LLMStreamEvent` remain the canonical model boundary.
- Confirm request-execution gateway actions bind `payload_hash` to canonical request identity.
- Confirm destination identity, redirect posture, and gateway operation vocabulary are explicit and shared rather than provider-local conventions.
- Confirm quota handling can represent both token-metered APIs and request-entitlement products without widening the trust boundary.
- Confirm any operator-facing posture remains broker-projected rather than becoming a second daemon-specific public API.

## Close Gate
Use the repository's standard verification flow before closing this change.
