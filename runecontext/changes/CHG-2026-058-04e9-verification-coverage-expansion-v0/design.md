# Design

## Overview
This feature expands what RuneCode captures and verifies.

The goal is not more logging for its own sake. The goal is to ensure the most important verification claims have first-class canonical evidence behind them and that missing required evidence becomes visible as a verification finding rather than as a silent gap.

## Core Coverage Areas

### Run Initiation And Authority
Capture:

- run identity
- session identity
- workspace identity
- stage identity where relevant
- initiating principal identity
- triggering source
- project or repository identity
- repo-scoped product-instance identity
- persistent ledger identity where the run's canonical history is committed
- project-substrate snapshot identity when a snapshot-scoped reconstruction claim is made
- workflow definition identity
- tool manifest identity
- prompt or request artifact digests where policy permits
- protocol bundle manifest hash

### Policy And Capability Envelope
Capture:

- policy bundle identity
- evaluation input digests
- allow and deny decisions
- reason codes
- capability envelope granted to the run
- capability denials
- policy overrides or explicit exceptions

### Approval Chain
Capture:

- approval request digest
- approval decision digest
- approver identity
- exact scope digest
- what the approver saw
- expiry
- approval consumption event
- supersession or revocation

### Runtime Identity And Hardening
Capture:

- runtime image descriptor digest
- toolchain identity
- backend kind
- isolate identity
- session binding digest
- launch context digest
- handshake transcript digest or equivalent binding
- hardening posture digest
- attestation evidence digest where present
- attestation verification record digest where present
- explicit reduced-assurance posture where attestation is unavailable or degraded

### Boundary Crossings
Capture:

- producer and consumer roles
- source family and destination family
- data class
- artifact digest
- authorization receipt
- deny receipt when blocked

### Secrets, Providers, And Network Egress
Capture:

- secret lease issue and revoke evidence
- provider profile identity
- model identity
- endpoint identity
- network target identity
- request and response artifact digests
- authorization or denial receipts
- explicit non-use summaries at run finalization where needed

Rules:

- do not store raw secret values
- do not default to storing raw prompts or raw provider payloads when a digest, data classification, and controlled artifact reference are enough

### Artifact Lineage
Capture:

- input artifact digests
- output artifact digests
- manifest identities
- transformation or derivation receipts
- publication or mutation receipts

### Anchoring And Verification
Capture:

- segment ids
- segment seal digests
- previous seal digest links
- segment Merkle roots
- anchor evidence digests
- anchor sidecar digests
- verification report digests
- verifier implementation identity
- verifier trust-root identity

### Degraded Posture And Fallbacks
Capture:

- why assurance was degraded
- what changed in the trust claim
- whether the user acknowledged it
- whether an approval or override was required

Examples:

- fallback from attested to non-attested posture
- anchoring deferred
- explicit reduced-assurance container mode
- offline verification when a stronger mode was expected
- break-glass or manual override

### Meta-Audit
Capture:

- who viewed sensitive evidence
- who exported evidence bundles
- who imported or restored evidence
- retention changes
- archival operations
- trust-root or verifier configuration changes

### Completeness And Negative Evidence
Support claims like:

- no secret lease was issued
- no network egress occurred
- no approval was consumed
- no artifact crossed a certain boundary

Because absence is hard to prove from raw logs alone, RuneCode should emit explicit summary receipts at finalization time for categories where negative claims matter.

## Receipt Expansion
This feature should extend receipt families to cover the high-value control and summary events missing from the current model.

Recommended priority receipt kinds:

- `run_start`
- `run_finalize`
- `policy_decision_allow`
- `policy_decision_deny`
- `capability_grant`
- `capability_deny`
- `approval_resolution`
- `approval_consumption`
- `boundary_crossing_authorized`
- `boundary_crossing_denied`
- `secret_lease_issued`
- `secret_lease_revoked`
- `provider_invocation_authorized`
- `provider_invocation_denied`
- `artifact_published`
- `override_or_break_glass`
- `degraded_posture_summary`
- `network_usage_summary`
- `secret_usage_summary`
- `verification_finalize`

Additional coverage expectations:

- approval resolution and approval consumption should be emitted as canonical evidence, not only implied by derived approval state
- provider provenance should preserve provider profile identity, model identity, and endpoint identity where those facts are available from trusted control-plane or runtime evidence
- control-plane provenance should preserve initiating principal identity and triggering source alongside the digests that shaped behavior

Each receipt should bind:

- actor identity
- scope
- policy or approval identity where relevant
- relevant input digests
- relevant output digests
- previous checkpoint where relevant
- timestamp
- reason codes

## Verification Report Strengthening
Verification reports should summarize:

- cryptographic validity
- historical admissibility
- current degradation status
- integrity status
- anchoring status
- storage posture status
- segment lifecycle status
- findings and hard failures
- verifier implementation identity
- verifier trust-root identity or digest
- protocol bundle manifest hash used for verification

This lane should make verification reports say not only what verified successfully, but also what required evidence was missing.

## Reason-Code Expansion
Recommended initial audit verification reason-code additions:

- `external_anchor_valid`
- `external_anchor_deferred_or_unavailable`
- `external_anchor_invalid`
- `missing_required_approval_evidence`
- `missing_runtime_attestation_evidence`
- `negative_capability_summary_missing`
- `verifier_identity_missing_or_unknown`
- `evidence_export_incomplete`

These reason codes make anchoring posture and missing-evidence posture understandable to users and auditors instead of leaving them implicit.

## Gap Closure Map

### Control-Plane Provenance Gap
RuneCode needs stronger capture of workflow definition, tool manifest, prompt or template digest where policy permits, protocol bundle manifest hash, verifier implementation digest, and trust-root or trust-policy digest.

### Approval-Basis Evidence Gap
RuneCode needs to preserve not only that an approval happened, but what the approver actually approved: diff digest, artifact-set digest, scope digest, summary or preview digest, and approval-consumption linkage.

### Provider And Egress Provenance Gap
RuneCode needs explicit provenance around provider use, secret use, request and response digests, endpoints, and allowed or denied network targets.

That provenance should also preserve provider-profile identity, model identity, and endpoint identity where those facts are available from trusted evidence so auditors can answer provider-usage questions without relying on UI-only state.

### Degraded Posture Gap
RuneCode needs stronger receipts and final summaries for reduced assurance, deferrals, and break-glass paths.

### Meta-Audit Gap
RuneCode needs evidence for export, import, restore, retention, and verifier-configuration events.

Meta-audit evidence should also make actor identity, scope, affected object digest, and operation result explicit so it remains useful for security and compliance review rather than only activity counting.

### Completeness And Omission Detection Gap
RuneCode needs stronger checks for missing required evidence, not only invalid present evidence.

### Verification-Of-Verification Gap
Verification reports should state which verifier and trust roots were used.

### Negative Capability Evidence Gap
RuneCode needs explicit summary evidence when the claim is that something did not happen.

Summary receipts should remain canonical evidence and bind the support posture for absence claims so a verifier can distinguish explicit absence evidence from limited or unknown evidence support.

## Failure Posture
- Verifiers should fail closed or degrade explicitly when required evidence is missing.
- Degraded posture must not be silently accepted.
- Denied, deferred, and overridden paths must be preserved, not hidden behind success-only summaries.

## Test Requirements
- degraded-posture tests
- fail-closed tests for missing required evidence
- approval-consumption linkage tests
- provider and egress provenance tests
- negative capability summary tests
- meta-audit event tests
- verification report reason-code tests

## Key Design Rule
The verification plane is incomplete if it can prove only successful happy-path history. It must also preserve what was denied, what was missing, what degraded, and who touched the verification system itself.
