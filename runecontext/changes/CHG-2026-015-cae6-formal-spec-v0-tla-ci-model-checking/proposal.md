## Summary
Key workflow-kernel security invariants are formally specified and continuously model-checked, starting with approval binding and consumption, stage-sign-off supersession, broker-versus-runner authority, gate-result and override linkage, and the minimal audit obligations required before high-risk state transitions are considered authoritative. This change also directly owns the small protocol and trusted-runtime refinements required to make that formal model implementable and durable.

## Problem
RuneCode already has the right kernel contracts spread across protocol schemas, trusted Go services, and runner durability rules, but those semantics are still easy to drift in subtle ways:
- approval decision acceptance can be confused with approval consumption
- stage sign-off depends on a canonical stage-summary hash, but that contract is not yet frozen in one place
- runner-supplied gate evidence can look authoritative unless the broker-owned materialization rule stays explicit
- partial blocking can accidentally leak into a second public lifecycle vocabulary
- transition-to-audit obligations can remain implicit and diverge across broker, audit, runner, and future gateway features

Without a formal model rooted in the canonical workflow kernel, later pre-MVP work such as the advanced TUI, Git Gateway, and future proof/attestation features are more likely to inherit incompatible interpretations of the same trust and routing semantics.

## Proposed Change
- Freeze the shared workflow-kernel semantics the formal model must treat as authoritative.
- Define and add the canonical stage-summary binding contract used by stage sign-off and supersession.
- Own the minimal protocol and trusted-runtime refinements required by the model instead of deferring them to a separate follow-on change.
- Write a TLA+ security-kernel model focused on approval, plan, gate, and broker-authority invariants.
- Add a small audit-transition-obligation model for the highest-risk authoritative state changes.
- Run TLC model checking in CI with bounded, deterministic configs and explicit traceability back to schemas and runtime modules.

## Why Now
This work now lands in `v0.1.0-alpha.5`, after the core workflow, policy, audit, broker, and runner foundations are in place, so the highest-risk invariants can be frozen before more user-facing and outbound capabilities build on them. Doing this now reduces the chance that `TUI Multi-Session + Power Workspace v0`, `Git Gateway (Commit/Push/PR)`, later concurrency work, and future proof or attestation features will each encode their own slightly different authority or approval semantics.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.
- The formal model should anchor to the existing protocol and trusted-service kernel rather than inventing feature-local approximations.
- TLC is the authoritative CI checker for `v0`; the spec should still stay finite-state and structured so later Apalache experimentation remains possible without a rewrite.

## Out of Scope
- Modeling arbitrary AI reasoning or prompt semantics.
- Replacing the runtime contracts with TLA-specific abstractions that no longer map cleanly to schemas and trusted modules.
- Unrelated feature work outside the kernel semantics or contract refinements required to support the formal model.
- Full audit-ledger, receipt, and verification semantics in the first model beyond the minimal transition-obligation slice needed for the workflow kernel.
- Re-introducing legacy Agent OS planning paths as canonical references.

## Impact
Keeps Formal Spec v0 (TLA+ + CI Model Checking) reviewable as a RuneContext-native change and freezes one strong security-kernel foundation that later UX, gateway, concurrency, and assurance features can extend without a second semantics rewrite later. The change now explicitly owns the enabling schema and trusted-runtime tightening needed for that foundation instead of leaving critical gaps for a later cleanup pass.
