# Tasks

## Phase 1: Close The End-To-End Execution Gap

- [ ] Trace the normal product path from session execution trigger to useful work and remove any remaining synthetic-only bridges.
- [ ] Wire trusted workflow selection and production adoption of authoritative built-in workflow assets for the supported first-party beta slice.
- [ ] Wire trusted `RunPlan` compilation and persistence into the real execution path rather than leaving it as largely test-proven foundation.
- [ ] Ensure the real execution path starts from the persisted authoritative plan identity.
- [ ] Ensure real runner checkpoint and result reporting reaches the broker through the typed production path.
- [ ] Remove ambiguity around noop/default runner transport behavior for the real supported workflow path.

## Phase 2: Make One Workflow Honestly Useful

- [ ] Deliver one first-party RuneContext workflow slice that a user can run usefully on a real project through the normal product path.
- [ ] Prefer `change_draft` or `spec_draft` as the minimum honest useful workflow.
- [ ] If scope remains manageable, also wire `draft_promote_apply` through the same real path so verified RuneContext lifecycle mutation is exercised end to end.
- [ ] Keep the chosen slice inspectable through runs, sessions, artifacts, approvals, and audit surfaces.
- [ ] Ensure the supported workflow path remains Linux-first and does not depend on future platform work.

## Phase 3: Align Runtime Assurance Truthfulness

- [ ] Coordinate the user-facing assurance story with `CHG-2026-054-6c1e-runtime-attestation-post-handshake-gating-v0`.
- [ ] Avoid presenting supported `attested` posture as settled beta truth until post-handshake gating lands.
- [ ] Ensure product surfaces distinguish current runtime evidence state from the final intended beta attestation story.

## Phase 4: TUI And Product Polish During Dogfooding

- [ ] Capture TUI polish items discovered while testing the real workflow path.
- [ ] Improve clarity for waiting, blocked, degraded, failed, resumed, and completed states.
- [ ] Improve attach, reconnect, and resume ergonomics where dogfooding reveals rough edges.
- [ ] Improve project-substrate remediation and workflow follow-up guidance where operator confusion appears.
- [ ] Improve discoverability for artifacts, audit evidence, approvals, and verification actions.
- [ ] Tighten wording, route labels, and status cues so the product reads like one coherent local system.

## Phase 5: Verification Smoke Path

- [ ] Run the supported useful workflow through the real product path and confirm canonical evidence is generated.
- [ ] Inspect the resulting run, artifacts, and audit records through normal product surfaces.
- [ ] Exercise audit evidence snapshot on the real workflow path.
- [ ] Exercise audit record inclusion on at least one real workflow-generated record.
- [ ] Exercise evidence bundle export and offline verification on the real workflow path.
- [ ] Exercise external audit anchoring on the real workflow path where environment and policy allow.

## Phase 6: Release-Surface Alignment

- [ ] Update roadmap and product-facing docs so alpha.11 is the hardening lane and beta.1 remains the milestone outcome.
- [ ] Align README, help text, and operator-facing wording with the real workflow and assurance story.
- [ ] Ensure release messaging does not imply a stronger end-to-end or attestation posture than the code actually provides.

## Acceptance Criteria

- [ ] RuneCode has one honest useful workflow path that runs through the real trusted and untrusted execution path.
- [ ] Trusted `RunPlan` compilation and persistence are part of the real production workflow path.
- [ ] Runner progress shown to operators comes from real reporting integration for the supported path.
- [ ] The supported path is inspectable through session, run, artifact, approval, and audit surfaces.
- [ ] Verification artifacts are generated and exercised from the same real workflow path.
- [ ] TUI and surrounding operator surfaces are polished enough that a new Linux user can test the product coherently.
- [ ] The beta story is more truthful and less scaffold-heavy after this alpha lane completes.
