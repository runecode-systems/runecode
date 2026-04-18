# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`

## Verification Notes
- Confirm all child features remain linked from this project change.
- Confirm child feature boundaries preserve the intended trust model without shortcut paths.
- Confirm the umbrella now explicitly names the inherited foundation from `secretsd` and `model-gateway`: canonical lease handoff, canonical model boundary, shared destination identity and request binding, broker-projected posture, and shared quota semantics.
- Confirm downstream provider lanes are framed as inheriting those contracts rather than redefining them.
- Confirm the umbrella explicitly keeps direct-credential providers on the same provider-profile and auth-material substrate that later OAuth and bridge-runtime lanes will reuse.

## Close Gate
Use the repository's standard verification flow before closing this change.
