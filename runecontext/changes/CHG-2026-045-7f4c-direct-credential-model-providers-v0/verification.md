# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm direct credentials land as one provider-auth path inside a shared provider substrate rather than as a second provider architecture.
- Confirm OpenAI-compatible and Anthropic-compatible adapters stay below the canonical typed model boundary.
- Confirm environment-variable and command-line secret injection remain forbidden.
- Confirm readiness, compatibility, and setup posture are broker-projected and explicitly reusable by later OAuth and bridge-runtime features.
- Confirm the roadmap and change text both place this feature in `v0.1.0-alpha.6`.

## Close Gate
Use the repository's standard verification flow before closing this change.
