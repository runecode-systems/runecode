## Summary
Expand RuneCode's verification evidence coverage so the most important operator, reviewer, and auditor questions can be answered from canonical evidence rather than reconstructed from guesswork, partial logs, or mutable convenience views.

This feature closes the gaps around control-plane provenance, approval basis, provider and egress provenance, degraded posture, meta-audit, negative capability evidence, verifier identity, and missing-evidence findings.

## Problem
RuneCode already has strong foundations in audit, policy, and runtime assurance, but several high-value evidence categories are still under-specified or unevenly captured.

Without explicit coverage expansion, RuneCode cannot answer enough of the questions that matter most in practice:

- what exact workflow, tool manifest, prompt, and protocol bundle shaped a run
- what exact diff, artifact set, or preview an approver saw and authorized
- which provider, model, endpoint, secret lease, and network target were used or denied
- whether degraded posture was explicit, acknowledged, and approved when required
- who viewed, exported, imported, restored, or reconfigured sensitive verification surfaces
- whether required evidence was missing, not merely whether present evidence was valid
- which verifier implementation and trust roots produced a verification report

The verification plane cannot be credible if it captures only successful happy-path events while leaving denials, omissions, degraded posture, and meta-audit outside the same evidence model.

## Proposed Change
- Expand evidence coverage across all of the first-class facts the verification plane must prove.
- Add stronger control-plane provenance capture, including workflow definition, tool manifest, prompt or template digest where policy permits, protocol bundle manifest hash, verifier implementation digest, and trust-root or trust-policy digest.
- Add approval-basis evidence so RuneCode preserves not just that approval happened but what exact diff, artifact set, scope digest, and preview the approver approved.
- Add provider, network, and secret provenance receipts and summaries.
- Add explicit degraded-posture receipts and summaries.
- Add meta-audit coverage for evidence view, export, import, restore, retention, and verifier-configuration events.
- Add negative capability summaries so RuneCode can support claims that something did not happen.
- Strengthen verification reports with missing-evidence findings, verifier identity, trust-root identity, and clearer anchoring posture.
- Extend audit verification reason codes to capture anchoring and missing-evidence outcomes explicitly.

## Why Now
This feature should follow the index and preservation work because it defines what evidence the broader foundation must actually carry.

If RuneCode does not close these gaps now, later verification work can still export, index, and display evidence efficiently while failing to preserve the evidence that real users and auditors most want to inspect.

Coverage work also makes the difference between a system that can say what happened and a system that can explain why a trust claim should be accepted, denied, or treated as degraded.

## Assumptions
- The most important verification target is the chain of authority and side effects.
- Negative claims matter in practice and therefore require explicit summary receipts where absence cannot be proven from raw logs alone.
- High-risk control-plane outcomes should be modeled as signed receipts or strongly committed evidence rather than left as UI summaries.
- Verification reports should identify the verifier and trust roots used so verification itself can be audited.

## Out of Scope
- Replacing canonical audit history with UI-only summaries.
- Treating raw transcript retention as the main solution to provenance gaps.
- Making proof-specific cryptographic machinery the main answer to ordinary verification questions.
- Solving multi-machine federation fully in this specific lane.

## Impact
This feature makes the verification plane answer real-world questions routinely instead of partially.

If completed, RuneCode will be able to:

- prove what exact control-plane inputs shaped a run
- show what exact scope and preview an approval covered and what final action consumed it
- show which provider, secret lease, endpoint, and network target were used or denied
- make degraded posture, deferrals, and break-glass paths explicit
- audit access to the verification plane itself
- produce verification reports that explain missing required evidence rather than only invalid present evidence
