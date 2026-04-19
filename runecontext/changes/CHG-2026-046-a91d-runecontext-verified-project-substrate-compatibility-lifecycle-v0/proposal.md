## Summary
RuneCode can adopt existing or initialize new canonical RuneContext project substrate in user repositories, enforce verified-mode compatibility against the repository's declared substrate contract, publish supported substrate ranges per release, and perform safe auditable upgrades without creating a second project-truth surface.

## Problem
Accepted migration decisions already say RuneContext is canonical, verified mode is required for normal RuneCode operation, and compatibility enforcement belongs in RuneCode. But there is still no product-level change that turns those decisions into one usable feature surface.

Without an explicit feature, the product risks three bad outcomes:
- a hidden RuneCode-only planning mirror instead of canonical `runecontext/`
- late or inconsistent failure on unsupported RuneContext state
- upgrade flows that are either manual-only, non-auditable, or incompatible with direct RuneContext usage

For end users and teams, the missing surface is specifically in the repositories they work on with RuneCode. When a user installs RuneCode on a machine and points it at a repository, RuneCode needs to discover, adopt, initialize, validate, and upgrade the repository's canonical RuneContext substrate under the hood while keeping that substrate fully compatible with direct `runectx` usage.

Without that explicit contract, mixed teams will drift into frustrating states:
- different RuneCode versions block for different reasons against the same repository
- direct `runectx` users and RuneCode users accidentally work against different assumptions about canonical project state
- one developer's initialization or upgrade silently changes repository truth for everyone else
- future session, workflow, audit, attestation, and verification features bind to ambient local assumptions instead of one reviewed project-context identity

## Proposed Change
- Project discovery, adoption, and initialization for canonical RuneContext project substrate in user repositories.
- A versioned project-substrate contract with deterministic discovery, validation, and compatibility posture evaluation.
- Release compatibility policy based on supported substrate ranges per RuneCode release rather than on local tool-version matching.
- Safe upgrade and remediation lifecycle with explicit preview, apply, and validate steps.
- Binding of validated RuneContext project-substrate snapshots into run planning, audit, attestation, and verification.
- Broker-owned diagnostics, blocked-state, and remediation surfaces, with future dashboard/operator-decision integration built on the same typed authority model.

## Why Now
This work now lands in `v0.1.0-alpha.6`, because the first usable end-to-end product cut needs canonical project context and compatibility gating before later session-execution, workflow-pack, and assurance features build on it.

Landing adoption, initialization, and upgrade semantics together now keeps RuneCode and direct RuneContext usage compatible from the start instead of forcing a later rewrite of project identity, compatibility, and assurance binding.

This is also the narrowest point to freeze the mixed-tool team model before more user-facing features depend on it. If project discovery, compatibility posture, and upgrade authority stay implicit until later, `CHG-2026-047`, `048`, `049`, and `050` will each be more likely to invent slightly different ideas of project setup, blocked state, and project-context identity.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.
- RuneContext may emit generic advisory compatibility warnings, but hard compatibility gating for RuneCode-managed repos remains a RuneCode responsibility.
- RuneCode initialization should not create a RuneCode-specific reduced folder set; when RuneCode initializes a repository it should produce the same canonical RuneContext substrate shape that `runectx init` would produce for the selected substrate version.
- Direct `runectx` usage, direct human edits, and mixed RuneCode versions may coexist in the same repository; compatibility decisions therefore must target the repository's declared substrate contract rather than each developer's installed tool version.

## Out of Scope
- Re-introducing legacy Agent OS planning paths as canonical references.
- Maintaining a hidden RuneCode-only project-context store in parallel with `runecontext/`.
- Treating unsupported or non-verified project state as a normal operating posture.
- Auto-upgrading repository substrate during normal RuneCode use.
- Making the TUI or dashboard authoritative for project-state decisions; future dashboard prompts remain a presentation layer over broker-owned typed contracts.

## Impact
Turns RuneContext from a planning assumption into a real product substrate for user repositories: one canonical project substrate, one explicit mixed-version compatibility model, one previewable upgrade path, and one validated project-context identity that later planning, workflow, audit, attestation, and verification features can all reuse without a second semantics rewrite.
