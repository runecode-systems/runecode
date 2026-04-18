## Summary
RuneCode can adopt existing or initialize new canonical `runecontext/` project state, enforce verified-mode compatibility, publish supported RuneContext version ranges per release, and perform safe auditable upgrades without creating a second project-truth surface.

## Problem
Accepted migration decisions already say RuneContext is canonical, verified mode is required for normal RuneCode operation, and compatibility enforcement belongs in RuneCode. But there is still no product-level change that turns those decisions into one usable feature surface.

Without an explicit feature, the product risks three bad outcomes:
- a hidden RuneCode-only planning mirror instead of canonical `runecontext/`
- late or inconsistent failure on unsupported RuneContext state
- upgrade flows that are either manual-only, non-auditable, or incompatible with direct RuneContext usage

## Proposed Change
- Project discovery, adoption, and initialization for canonical `runecontext/` state.
- Release compatibility policy and supported RuneContext version reporting.
- Safe upgrade and remediation lifecycle.
- Binding of RuneContext project state into run planning, audit, attestation, and verification.
- Broker/TUI/CLI diagnostics, blocked-state, and remediation surfaces.

## Why Now
This work now lands in `v0.1.0-alpha.6`, because the first usable end-to-end product cut needs canonical project context and compatibility gating before later session-execution, workflow-pack, and assurance features build on it.

Landing adoption, initialization, and upgrade semantics together now keeps RuneCode and direct RuneContext usage compatible from the start instead of forcing a later rewrite of project identity, compatibility, and assurance binding.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.
- RuneContext may emit generic advisory compatibility warnings, but hard compatibility gating for RuneCode-managed repos remains a RuneCode responsibility.

## Out of Scope
- Re-introducing legacy Agent OS planning paths as canonical references.
- Maintaining a hidden RuneCode-only project-context store in parallel with `runecontext/`.
- Treating unsupported or non-verified project state as a normal operating posture.

## Impact
Turns RuneContext from a planning assumption into a real product substrate for adoption, initialization, compatibility, upgrades, and assurance binding while preserving compatibility with direct RuneContext use in the same repository.
