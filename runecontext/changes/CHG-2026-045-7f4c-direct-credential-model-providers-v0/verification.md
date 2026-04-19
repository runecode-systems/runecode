# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just lint`
- `just test`

## Verification Notes
- Confirm direct credentials land as one provider-auth path inside a shared provider substrate rather than as a second provider architecture.
- Confirm provider-profile identity remains stable across auth-mode changes and credential rotation.
- Confirm OpenAI-compatible and Anthropic-compatible adapters stay below the canonical typed model boundary.
- Confirm `v0` adapter targets are OpenAI-compatible Chat Completions and Anthropic Messages, and that future OpenAI Responses support remains an additive extension beneath the same substrate.
- Confirm environment-variable and command-line secret injection remain forbidden.
- Confirm raw secret values do not travel in ordinary typed broker request or response bodies.
- Confirm CLI and TUI both use the same broker-owned setup flow and that TUI secret entry is masked and aligned with the established Bubble Tea and Lip Gloss shell patterns.
- Confirm allowlisted model identity remains canonical and provider discovery or probes remain advisory inputs rather than trust roots.
- Confirm readiness, compatibility, and setup posture are broker-projected and explicitly reusable by later OAuth and bridge-runtime features.
- Confirm readiness posture distinguishes configuration, credential, connectivity, compatibility, and effective-readiness dimensions rather than flattening them into one status string.
- Confirm the roadmap and change text both place this feature in `v0.1.0-alpha.6`.

## Close Gate
Use the repository's standard verification flow before closing this change.
