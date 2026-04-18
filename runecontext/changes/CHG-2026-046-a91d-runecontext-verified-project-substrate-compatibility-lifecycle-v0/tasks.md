# Tasks

## Discovery, Adoption, and Initialization

- [ ] Detect existing canonical `runecontext/` state and verified-mode posture.
- [ ] Support initializing canonical RuneContext state for repos that do not already have it.
- [ ] Keep adopt-existing and initialize-new flows compatible with direct RuneContext usage.
- [ ] Forbid creation of a second RuneCode-only project-truth surface.

## Compatibility and Upgrade Lifecycle

- [ ] Define the RuneContext compatibility policy that each RuneCode release reports and enforces.
- [ ] Surface active project RuneContext version, supported range, compatibility posture, and blocked reasons through broker diagnostics.
- [ ] Define safe upgrade flows with preview, apply, validate, and remediation steps.
- [ ] Keep unsupported or non-verified states fail closed for normal operation while preserving safe diagnostic and remediation access.

## Assurance and Verification Binding

- [ ] Bind concrete RuneContext project state into run planning and project-context selection.
- [ ] Bind concrete RuneContext project state into audit, attestation, and verification outputs where project context matters.
- [ ] Keep assurance history under `runecontext/assurance/` and preserve compatibility with verified-mode RuneContext expectations.

## Broker, TUI, and CLI Surfaces

- [ ] Extend broker version and readiness surfaces with RuneContext compatibility posture.
- [ ] Surface adopt/init/upgrade/remediation posture in TUI and CLI flows without making clients authoritative.
- [ ] Keep normal product UX in RuneCode while invoking RuneContext capabilities under the hood where appropriate.

## Acceptance Criteria

- [ ] RuneCode can adopt existing compatible RuneContext state and initialize new compatible RuneContext state.
- [ ] RuneCode releases publish supported RuneContext compatibility ranges and fail closed on unsupported normal-operation states.
- [ ] Upgrade flows are explicit, auditable, and compatible with direct RuneContext use in the same repo.
- [ ] Run planning, audit, attestation, and verification can bind to concrete RuneContext project state without introducing a second project-truth surface.
