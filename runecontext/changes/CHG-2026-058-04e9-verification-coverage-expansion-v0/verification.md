# Verification

## Planned Checks
- `just test`
- `go test ./internal/protocolschema`
- `cd runner && node --test scripts/protocol-fixtures.test.js`

## Verification Notes
- Confirm the design enumerates the auditable facts across initiation, policy, approval, runtime, boundary crossing, provider and network use, artifact lineage, anchoring, degraded posture, meta-audit, and negative evidence.
- Confirm the feature explicitly captures both allow and deny outcomes where relevant.
- Confirm approval-basis evidence includes what the approver actually saw and what final action consumed the approval.
- Confirm provider and egress provenance includes provider profile, model, endpoint, network target, secret-lease evidence, and request or response digests.
- Confirm degraded-posture capture includes why assurance changed, whether the user acknowledged it, and whether approval or override was required.
- Confirm meta-audit coverage includes evidence view, export, import, restore, retention, and verifier-configuration events.
- Confirm missing-evidence findings are treated as first-class verification outcomes.
- Confirm verification reports preserve verifier identity and trust-root identity.
- Confirm the reason-code list includes explicit anchoring and missing-evidence reasons.

## Close Gate
Use the repository's standard verification flow before closing this change.
