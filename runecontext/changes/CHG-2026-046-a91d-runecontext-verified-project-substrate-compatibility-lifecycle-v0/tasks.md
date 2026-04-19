# Tasks

## Project Substrate Contract + Validation

- [ ] Define one explicit versioned RuneContext project-substrate contract for user repositories instead of treating arbitrary `runecontext/` files as equivalent.
- [ ] Define the required canonical anchors for `v0`, including repo-root `runecontext.yaml`, canonical `runecontext/` source path, verified posture declaration, and canonical assurance path under `runecontext/assurance/`.
- [ ] Keep RuneCode initialization aligned with the canonical substrate shape that `runectx init` produces for the selected substrate version; do not introduce a RuneCode-specific reduced layout.
- [ ] Define deterministic repo-root discovery rules that inspect one authoritative repository root rather than walking arbitrarily for the nearest `runecontext.yaml`.
- [ ] Add read-only validation that normalizes authoritative project-substrate inputs and produces a validated snapshot identity.
- [ ] Bind project-context identity to a validated snapshot digest rather than only to repo path, local machine assumptions, or `runecontext_version` text alone.
- [ ] Forbid creation of a second RuneCode-only project-truth surface.

## Discovery, Adoption, and Initialization

- [ ] Detect existing canonical RuneContext project substrate and verified-mode posture.
- [ ] Support read-only adoption of existing compatible canonical substrate.
- [ ] Support initializing canonical RuneContext substrate for repos that do not already have it.
- [ ] Keep adopt-existing and initialize-new flows compatible with direct RuneContext usage and direct human edits.
- [ ] Make initialization explicit, previewable, and idempotent.
- [ ] Refuse initialization when conflicting candidate state exists instead of silently overwriting or normalizing it.

## Compatibility Policy + Mixed-Version Team Model

- [ ] Define the project-substrate compatibility policy that each RuneCode release reports and enforces.
- [ ] Publish compatibility as a supported substrate range plus recommended target posture rather than as an exact local tool-version match requirement.
- [ ] Treat the repository's declared and validated substrate contract as the compatibility authority, with local RuneCode and `runectx` versions as diagnostics only.
- [ ] Define explicit compatibility posture states and stable reason codes for at least missing, invalid, non-verified, supported-current, supported-with-upgrade-available, unsupported-too-old, and unsupported-too-new cases.
- [ ] Surface active project substrate version or contract identity, supported range, compatibility posture, and blocked reasons through broker diagnostics.
- [ ] Allow normal operation for supported-current and supported-with-upgrade-available posture only.
- [ ] Keep incompatible or non-verified states fail closed for RuneCode normal operation while preserving safe diagnostic and remediation access.

## Upgrade + Remediation Lifecycle

- [ ] Define safe upgrade flows with preview, apply, validate, and remediation steps.
- [ ] Ensure upgrade preview enumerates intended file changes, preconditions, expected resulting posture, and required follow-up.
- [ ] Keep upgrades explicit and auditable repository mutations rather than implicit startup or attach-time normalization.
- [ ] Never auto-upgrade substrate during ordinary RuneCode use.
- [ ] Preserve direct `runectx` compatibility before and after reviewed upgrade flows.

## Planning, Assurance, and Verification Binding

- [ ] Bind validated project-substrate snapshot identity into run planning and project-context selection.
- [ ] Bind validated project-substrate snapshot identity into audit, attestation, and verification outputs where project context matters.
- [ ] Keep assurance history under `runecontext/assurance/` and preserve compatibility with verified-mode RuneContext expectations.
- [ ] Ensure project-context binding does not rely on ambient repo-path assumptions or client-local heuristics.

## Broker, TUI, and CLI Surfaces

- [ ] Add a dedicated broker-owned typed project-substrate posture surface instead of overloading readiness as the only diagnostics contract.
- [ ] Extend broker version and readiness surfaces with summary RuneContext compatibility posture.
- [ ] Surface adopt/init/upgrade/remediation posture in TUI and CLI flows without making clients authoritative.
- [ ] Keep normal product UX in RuneCode while invoking RuneContext capabilities under the hood where appropriate.
- [ ] Keep blocked-state explanation and remediation guidance broker-projected rather than client-local.

## Future Dashboard / Operator Decision Path

- [ ] Reserve a future broker-owned typed operator-decision path for setup and lifecycle prompts surfaced in the dashboard or equivalent RuneCode UX.
- [ ] Keep future dashboard prompts as a presentation layer over broker-owned typed posture and mutation contracts rather than a second authority surface.
- [ ] Reuse the shared approval and hard-floor model for high-risk or policy-gated project-lifecycle apply steps instead of inventing a setup-only approval lane.

## Acceptance Criteria

- [ ] RuneCode can adopt existing compatible RuneContext project substrate and initialize new canonical substrate state in user repositories without diverging from `runectx init` folder/output expectations.
- [ ] RuneCode releases publish supported substrate compatibility ranges and evaluate compatibility against the repository substrate contract rather than per-user installed tool versions.
- [ ] Mixed teams using different RuneCode versions and direct `runectx` usage can continue working against one canonical repository substrate without RuneCode creating a private lock or mirror.
- [ ] Missing, invalid, non-verified, and unsupported substrate states fail closed for normal RuneCode operations while still permitting diagnostics and remediation.
- [ ] Compatible-but-older substrate remains usable with explicit upgrade advisory rather than hard-blocking normal operation.
- [ ] Upgrade flows are explicit, previewable, auditable, and compatible with direct RuneContext use in the same repository.
- [ ] Run planning, audit, attestation, and verification can bind to validated project-substrate snapshot identity without introducing a second project-truth surface.
- [ ] Broker, TUI, and CLI surfaces remain thin adapters over one broker-owned project-substrate authority model.
