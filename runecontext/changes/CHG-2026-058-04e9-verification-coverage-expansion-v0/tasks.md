# Tasks

## Control-Plane Provenance

- [ ] Capture workflow definition digest, tool manifest digest, prompt or template digest where policy permits, protocol bundle manifest hash, verifier implementation digest, and trust-root or trust-policy digest.
- [ ] Capture initiating principal identity and triggering source alongside the trusted control-plane digests that shaped behavior.

## Approval-Basis Evidence

- [ ] Capture diff digest, artifact-set digest, scope digest, summary or preview digest, and approval-consumption linkage.
- [ ] Preserve what the approver saw, not only that approval happened.
- [ ] Emit canonical approval-resolution and approval-consumption evidence rather than leaving those transitions only in derived approval state.
- [ ] Preserve expiry, supersession, and revocation linkage where those states affect approval admissibility.

## Provider, Network, And Secrets Provenance

- [ ] Add provider invocation authorization and deny receipts.
- [ ] Add provider request and response artifact digests.
- [ ] Add secret lease issue and revoke receipts.
- [ ] Add network target descriptors and digest identities.
- [ ] Add final summary receipts that can state no provider invocation or no secret lease occurred.
- [ ] Preserve provider profile identity, model identity, and endpoint identity where those facts are available from trusted runtime or control-plane evidence.

## Degraded Posture And Negative Evidence

- [ ] Add degraded-posture receipts and final summaries.
- [ ] Record why assurance degraded, what changed in the trust claim, whether the user acknowledged it, and whether approval or override was required.
- [ ] Add negative-capability summary receipts for categories where absence matters.
- [ ] Make negative-evidence support explicit for the high-value absence claims: no secret lease issued, no network egress occurred, no approval was consumed, and no artifact crossed a given boundary.
- [ ] Emit canonical network and secret usage summary receipts that preserve explicit support posture for absence claims.

## Meta-Audit

- [ ] Record evidence export events.
- [ ] Record evidence import and restore events.
- [ ] Record retention-policy changes.
- [ ] Record archival operations.
- [ ] Record trust-root updates and verifier configuration changes.
- [ ] Record sensitive evidence view events where appropriate.
- [ ] Preserve actor identity, scope, object identity, and operation result for meta-audit events that affect verification surfaces.

## Verification Report And Reason Codes

- [ ] Strengthen verification reports with verifier identity and trust-root identity.
- [ ] Add missing-evidence findings.
- [ ] Add explicit anchoring posture reporting.
- [ ] Extend audit verification reason codes with `external_anchor_valid`, `external_anchor_deferred_or_unavailable`, `external_anchor_invalid`, `missing_required_approval_evidence`, `missing_runtime_attestation_evidence`, `negative_capability_summary_missing`, `verifier_identity_missing_or_unknown`, and `evidence_export_incomplete`.
- [ ] Add verifier checks for required-evidence invariants: attested runs must carry attestation evidence and verification records; boundary crossings must carry authorization receipts; production-affecting mutations must carry approval or explicit exception evidence.
- [ ] Add verifier checks that distinguish explicit absence evidence from limited or unknown evidence support for negative-capability claims.

## Verification

- [ ] Add degraded-posture tests.
- [ ] Add fail-closed tests for missing required evidence.
- [ ] Add approval-consumption linkage tests.
- [ ] Add provider and egress provenance tests.
- [ ] Add negative-capability summary tests.
- [ ] Add meta-audit event tests.
- [ ] Add verification-report reason-code tests.
- [ ] Add receipt-family tests for canonical approval, boundary, publication, override, and summary evidence lanes.

## Acceptance Criteria

- [ ] The most important auditor questions can be answered from canonical evidence.
- [ ] Degraded posture is explicit, not hidden.
- [ ] Missing required evidence produces explicit findings.
- [ ] Verification reports identify the verifier and trust roots used.
- [ ] Provider, secret, approval-basis, and meta-audit coverage are first-class evidence lanes rather than UI-only summaries.
- [ ] Negative evidence is explicit enough to support concrete absence claims for secret use, network egress, approval consumption, and boundary crossing.
- [ ] Control-plane provenance preserves both the trusted digests and the initiating or triggering context needed to explain why a run happened.
- [ ] Canonical receipt families cover material policy, capability, approval, publication, boundary, override, and summary evidence for the foundation lane.
