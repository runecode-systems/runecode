# Design

## Overview
Make canonical RuneContext project substrate in user repositories a first-class product foundation instead of an implicit assumption.

## Key Decisions
- Canonical project truth remains under `runecontext/` with `runecontext.yaml` at the repo root; there is no `.runecontext/` mirror and no daemon-private planning store.
- RuneCode must support both adopting existing compatible RuneContext state and initializing new compatible RuneContext state.
- RuneCode initialization should create the same canonical substrate shape that `runectx init` would create for the selected substrate version; RuneCode must not introduce a reduced or product-private initialization layout.
- Each RuneCode release should declare the project-substrate compatibility range it supports and should expose the active project compatibility posture through broker diagnostics.
- Hard compatibility enforcement for RuneCode-managed repos remains in RuneCode; RuneContext may still provide generic advisory warnings.
- Unsupported or non-verified project states are fail-closed normal-operation blocks with safe diagnostics and remediation flows only.
- Upgrade flows must be previewable, explicit, auditable, and compatible with direct RuneContext usage outside RuneCode.
- Compatibility is evaluated against the repository's declared and validated substrate contract, not against each developer's installed RuneCode or `runectx` version.
- Direct `runectx` usage and direct human edits remain valid inputs; RuneCode is not the only author of canonical project state.
- Discovery and posture evaluation are read-only; RuneCode must never auto-upgrade or silently rewrite project substrate during normal use.
- Run planning, verification, audit, attestation, and git-proof flows should bind to a validated project-substrate snapshot digest rather than to ambient local repository assumptions, raw paths, or version strings alone.
- Broker-owned typed contracts are the authority surface for project posture and later operator decisions; TUI, CLI, and future dashboard prompts remain thin clients of those contracts.

## Canonical Project Substrate Contract

The feature should define one explicit versioned project-substrate contract for user repositories rather than treating arbitrary files under `runecontext/` as equivalent.

For `v0`, the contract should distinguish:
- required canonical anchors:
  - repo-root `runecontext.yaml`
  - canonical source path rooted at `runecontext/`
  - verified posture declaration
  - canonical assurance path under `runecontext/assurance/`
- canonical substrate shape produced by `runectx init` for the selected substrate version
- optional canonical content surfaces that may be created lazily by later workflows and product features, as long as they remain within the reviewed substrate contract

RuneCode may initialize repositories lazily in terms of feature usage, but it must not diverge from the canonical `runectx init` shape. If a richer canonical folder set is part of the selected substrate version, RuneCode init should produce that same shape rather than a RuneCode-specific shortcut.

## Discovery Model

- Project discovery should resolve one authoritative repository root and inspect only that root for canonical RuneContext substrate.
- Discovery should not walk arbitrarily upward looking for the nearest `runecontext.yaml`; repository identity must remain deterministic.
- Discovery should classify at least:
  - substrate missing
  - substrate invalid
  - substrate present but non-verified
  - substrate present and supported
  - substrate present but unsupported or upgradeable
- Discovery must not mutate project state.

## Validation And Snapshot Identity

- Validation should normalize the authoritative project-substrate inputs and produce one validated snapshot identity.
- That identity should be a canonical digest of the validated project-substrate snapshot, not a raw repo path and not only `runecontext_version` plus assurance tier.
- Snapshot identity should remain stable across irrelevant local-machine differences and should bind the concrete substrate inputs that later planning, workflow, audit, attestation, and verification features depend on.
- Assurance history should remain canonical under `runecontext/assurance/`, but the primary project-context identity should bind the validated substrate snapshot rather than every mutable historical assurance artifact.

## Compatibility Lifecycle

- Discovery should identify:
  - presence or absence of canonical RuneContext substrate
  - verified-mode posture
  - active declared substrate version or contract identity
  - supported range for the running RuneCode release
  - compatibility posture and stable reason codes
  - blocked or degraded reasons when normal operation is unsafe
- Upgrade lifecycle should support:
  - inspect current posture
  - preview compatible upgrade actions
  - apply reviewed upgrade steps
  - re-run validation and assurance checks
- Broker version and readiness surfaces should report both product compatibility policy and active project posture.

Recommended compatibility posture states for this feature:
- `missing`
- `invalid`
- `non_verified`
- `supported_current`
- `supported_with_upgrade_available`
- `unsupported_too_old`
- `unsupported_too_new`

Normal operation should be permitted only for `supported_current` and `supported_with_upgrade_available`.

### Mixed-Version Team Model

- Multiple RuneCode versions and direct `runectx` usage may coexist against the same repository.
- Local tool versions are diagnostics only; they are not the compatibility authority.
- The repository's declared and validated substrate contract is the compatibility target.
- If one RuneCode version supports the current repository substrate and another does not, only the unsupported client should block its own normal operation; the repository itself is not placed into a RuneCode-private locked state.
- Compatible-but-older substrate should remain usable with explicit upgrade advisory rather than a hard block.
- Unsupported-too-new or unsupported-too-old substrate should fail closed for RuneCode normal operations, while still allowing diagnostics and remediation.

## Initialization, Adoption, And Upgrade Semantics

### Adoption
- Adoption is read-only recognition of existing compatible canonical project substrate.
- Adoption must not silently normalize, repair, or rewrite discovered substrate.
- Compatible but older substrate should adopt successfully and report `supported_with_upgrade_available`.

### Initialization
- Initialization should be explicit, previewable, and idempotent.
- Initialization should write only canonical RuneContext substrate files and directories.
- Initialization should refuse to overwrite conflicting candidate state and instead guide the operator to adopt or remediate.
- Initialization must preserve parity with `runectx init` folder/output expectations for the selected substrate version.

### Upgrade
- Upgrade should be an explicit repository mutation flow with `preview -> apply -> validate` semantics.
- Upgrade preview should enumerate intended file changes, preconditions, expected resulting posture, and any required operator follow-up.
- Upgrade apply should record auditable evidence and then re-run validation.
- RuneCode must never auto-upgrade substrate during ordinary startup, attach, status, or planning flows.

## Blocked-State And Remediation UX

- Missing, invalid, non-verified, and unsupported substrate states are blocked normal-operation states.
- In blocked states, RuneCode should allow diagnostics and remediation flows only, including:
  - inspect posture
  - explain incompatibility or invalidity
  - initialize substrate when missing
  - preview and apply upgrade or remediation when safe and explicit
- Routine planning, workflow execution, repository mutation, and other normal managed operations should stay blocked until posture becomes supported.
- Compatible-but-upgradeable posture should not be blocked; it should surface advisory upgrade guidance.

## Broker, TUI, And Future Dashboard Contract

- The broker should expose a dedicated typed project-substrate posture surface rather than overloading readiness as a catch-all diagnostics blob.
- Existing version and readiness surfaces should include only summary posture signals needed for high-level status views.
- TUI and CLI should remain thin adapters over broker-owned typed posture, initialization, adoption, upgrade-preview, upgrade-apply, and remediation contracts.
- Future dashboard-driven questions, prompts, and setup decisions should build on the same broker-owned typed authority surface rather than creating a second interactive setup protocol.
- Where an apply step becomes policy-gated or high-risk, the eventual decision path should reuse the shared approval and hard-floor model rather than inventing a project-setup-only approval lane.

## Main Workstreams
- Project Substrate Contract + Validation.
- Project Discovery, Adoption, and Initialization.
- Compatibility Policy + Mixed-Version Team Model.
- Upgrade + Remediation Lifecycle.
- Snapshot Digest Binding for Planning, Audit, Attestation, and Verification.
- Broker/TUI/CLI Diagnostics and Blocked-State UX.
- Future Dashboard/Operator-Decision Integration on Shared Typed Contracts.

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
