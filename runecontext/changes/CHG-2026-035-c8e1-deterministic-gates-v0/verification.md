# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm the split preserves deterministic gate and evidence requirements from the prior combined change.
- Confirm gate outcomes and override paths are fail-closed and auditable.
- Confirm the docs define gates as typed first-class checks with stable identity, explicit lifecycle, and separate attempt semantics.
- Confirm the docs introduce a dedicated typed gate-evidence model and a dedicated evidence data-class direction rather than leaving gate proof as generic logs only.
- Confirm overrides remain canonical policy actions tied to exact gate identity and current policy context rather than feature-local exceptions.

## Close Gate
Use the repository's standard verification flow before closing this change.
