# Tasks

## Phase 1: Close The End-To-End Execution Gap

- [x] Trace the normal product path from session execution trigger to useful work and remove any remaining synthetic-only bridges.
- [x] Wire trusted workflow selection and production adoption of authoritative built-in workflow assets for the supported first-party beta slice.
- [x] Wire trusted `RunPlan` compilation and persistence into the real execution path rather than leaving it as largely test-proven foundation.
- [x] Ensure the real execution path starts from the persisted authoritative plan identity.
- [x] Ensure real runner checkpoint and result reporting reaches the broker through the typed production path.
- [x] Remove ambiguity around noop/default runner transport behavior for the real supported workflow path.
- [x] Add or document the normal product runner launch entrypoint for the supported path.
- [x] Make run, session, and TUI workflow projections plan-authoritative for the supported path rather than artifact-inferred.

## Phase 2: Prove Canonical RuneContext Project Lifecycle

- [ ] Prove project-substrate inspect and posture reporting through normal product surfaces.
- [ ] Prove compatible existing substrate adoption remains read-only and does not silently rewrite discovered state.
- [ ] Prove missing substrate initialization through explicit preview/apply and follow-up validation/status.
- [ ] Prove supported older substrate upgrade through explicit preview/apply and follow-up validation/status.
- [ ] Prove normal productive workflow execution remains blocked for missing, invalid, non-verified, or unsupported substrate posture.
- [ ] Keep apply flows broker-owned, typed, auditable where mutation occurs, and visible through TUI or CLI surfaces.

## Phase 3: Make The Required Workflow Loop Honestly Useful

- [x] Deliver `change_draft` from prompt to typed change-draft artifact through the real product path.
- [x] Deliver `spec_draft` from prompt to typed spec-draft artifact through the same real product path.
- [x] Deliver `draft_promote_apply` for a reviewed change draft into canonical `runecontext/changes/`.
- [x] Deliver `draft_promote_apply` for a reviewed spec draft into canonical `runecontext/specs/`.
- [x] Deliver `approved_change_implementation` from one reviewed implementation input set containing one or more approved change/spec inputs by exact digest.
- [x] Enforce the approved implementation input-set identity contract: bound artifact digest for exact stored bytes, `input_set_digest` for the broker-recomputed canonical body with `input_set_digest` omitted, and fail-closed validation on drift.
- [x] Keep approved implementation run, audit, and projection fields from conflating `input_set_artifact_digest` and semantic `input_set_digest` where both identities matter.
- [x] Allow approved implementation to update required RuneContext lifecycle metadata when the approved input set requires it, without adding a separate lifecycle-close workflow operation in this lane.
- [x] Keep the supported workflow loop inspectable through runs, sessions, artifacts, approvals, and audit surfaces.
- [x] Ensure the supported workflow loop remains Linux-first and does not depend on future platform work.

## Phase 4: Align Runtime Assurance Truthfulness

- [ ] Coordinate the user-facing assurance story with `CHG-2026-054-6c1e-runtime-attestation-post-handshake-gating-v0`.
- [ ] Avoid presenting supported `attested` posture as settled beta truth until post-handshake gating is implemented, verified, and integrated into the supported path.
- [ ] Ensure product surfaces distinguish current runtime evidence state from the final intended beta attestation story.

## Phase 5: TUI And Product Polish During Dogfooding

- [ ] Capture TUI polish items discovered while testing the real workflow path.
- [ ] Improve clarity for waiting, blocked, degraded, failed, resumed, and completed states.
- [ ] Improve attach, reconnect, and resume ergonomics where dogfooding reveals rough edges.
- [ ] Improve project-substrate remediation and workflow follow-up guidance where operator confusion appears.
- [ ] Improve discoverability for artifacts, audit evidence, approvals, and verification actions.
- [ ] Tighten wording, route labels, and status cues so the product reads like one coherent local system.
- [ ] Fix blockers and misleading product-truth issues found during TUI walkthroughs before closure.
- [ ] Record non-blocking polish follow-ups when they do not block the supported beta proof.

## Phase 6: Verification Smoke Path

- [ ] Run the supported project-substrate lifecycle proof through normal product surfaces.
- [x] Run `change_draft`, `spec_draft`, `draft_promote_apply`, and `approved_change_implementation` through the real product path and confirm canonical evidence is generated.
- [x] Inspect the resulting run, artifacts, and audit records through normal product surfaces.
- [x] Exercise audit evidence snapshot on the real workflow path.
- [x] Exercise audit record inclusion on at least one real workflow-generated record.
- [x] Exercise evidence bundle export and offline verification on the real workflow path.
- [ ] Exercise external audit anchoring on the real workflow path where environment and policy allow.

## Phase 7: Release-Surface Alignment

- [x] Update roadmap and product-facing docs so alpha.11 is the hardening lane and beta.1 remains the milestone outcome.
- [x] Resolve incomplete alpha.11 roadmap wording, including any blank `Feature changes:` entry.
- [x] Represent `CHG-2026-054-6c1e-runtime-attestation-post-handshake-gating-v0` consistently where beta assurance wording depends on its verified integration.
- [ ] Align README, help text, and operator-facing wording with the real workflow and assurance story.
- [x] Ensure release messaging does not imply a stronger end-to-end or attestation posture than the code actually provides.
- [x] Keep git remote publication out of required beta messaging unless a separate reviewed publishing smoke path is added.

## Acceptance Criteria

- [ ] RuneCode proves canonical project-substrate lifecycle through inspect/adopt/init/upgrade/validate/status surfaces.
- [x] RuneCode runs `change_draft`, `spec_draft`, `draft_promote_apply`, and `approved_change_implementation` through the real trusted and untrusted execution path.
- [x] RuneCode promotes reviewed change and spec drafts into canonical RuneContext files through the shared audited mutation path.
- [x] RuneCode implements one reviewed implementation input set through the shared workflow system, including local workspace mutation and required RuneContext lifecycle metadata updates when approved.
- [x] Trusted `RunPlan` compilation and persistence are part of the real production workflow path.
- [x] Runner progress shown to operators comes from real reporting integration for the supported path.
- [x] The supported path is inspectable through session, run, artifact, approval, and audit surfaces.
- [x] Verification artifacts are generated and exercised from the same real workflow path.
- [ ] TUI and surrounding operator surfaces are polished enough that a new Linux user can test the product coherently.
- [x] Git remote publication is not implied unless explicitly verified by a separate publishing smoke path.
- [x] The beta story is more truthful and less scaffold-heavy after this alpha lane completes.
