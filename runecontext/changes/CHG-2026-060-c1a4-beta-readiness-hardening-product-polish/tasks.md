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

- [ ] Run a focused TUI once-over against the supported beta workflow path and capture implementation findings before broad polish changes: launch through the canonical `runecode` path, inspect Dashboard, Action Center, Chat, Runs, Approvals, Artifacts, Audit, Status, Model Providers, and setup/remediation routes, and record where the UI reads like a dense debug console, where primary actions are unclear, where route terminology leaks protocol internals, and where product-truth cues are missing or misleading.
- [ ] Reframe the global shell chrome into a calm product workbench rather than a constant debug/status dump: keep route, focus, product posture, sync truth, and active work visible, but reduce repeated long trust-boundary text, raw layout/focus diagnostics, history noise, clipboard noise, and exhaustive shortcuts in the primary frame; preserve detailed truth through command discovery, inspectors, status/detail surfaces, or copy actions instead of deleting it.
- [ ] Promote `Action Center` as the operator home for attention and follow-up: make it the clearest place to see pending approvals, blocked workflow impact, failed or degraded runs, project-substrate remediation needs, audit/runtime degradation, waiting work, stale queues, and next actions; every item should state the reason, impact, owner/action required, target route/object, and evidence link source without inferring state outside broker-owned surfaces.
- [ ] Polish `Dashboard` into a calm executive overview that points operators to `Action Center`: show the most important current state first (`healthy`, `needs attention`, `blocked`, `degraded`, `active work`, or `no work yet`), summarize product readiness and workflow posture in product language, show a small number of high-value counts and next actions, and move raw readiness fields, protocol bundle details, watch-family terminology, and low-level audit facts into secondary/detail views.
- [ ] Create a unified state-card UX pattern for loading, empty, ready, waiting, blocked, degraded, failed, resumed, completed, and approval-required states: each card should include a short state label, a plain-English reason, the authoritative source or object identity when useful, the next recommended action, and the shortcut/route to take that action; the pattern must avoid fake progress and must distinguish broker-known state from unknown or unavailable state.
- [ ] Make project-substrate setup and remediation feel like a guided broker-owned flow in the TUI: expose inspect/current posture, compatible adoption, init preview, init apply, upgrade preview, upgrade apply, and post-apply validation/status as an understandable sequence; preview/apply screens should explain whether mutation will occur, what handle or digest was acquired without leaking sensitive paths, what blocks normal operation, and what the operator should do next if preview/apply is unavailable or fails.
- [ ] Improve Chat route workflow clarity so execution feels alive, useful, and honest: show the active canonical session, composer state, current or latest execution state, waiting reason, approval requirement, linked run/artifact/audit counts, and follow-up action in product language; surface real broker/runner stages such as plan compiled, runner active, checkpoint received, approval required, artifact ready, failed, and completed only when broker-owned state supports them.
- [ ] Improve Runs route clarity around execution outcomes and evidence: make the run directory and inspector clearly show status, workflow operation, plan authority, runner/reporting posture, failure or blocked reason, linked approvals, artifacts, and audit records; provide obvious navigation/copy cues from a selected run to its session, artifacts, approvals, and audit evidence.
- [ ] Improve Approvals route confidence and safety: distinguish pending, resolved, expired, unsupported, and approval-required workflow states; show why the approval exists, what exact object/action it gates, what will happen after approval, what evidence or artifact should be reviewed first, and why an approval cannot be resolved when validation blocks the action.
- [ ] Improve Artifacts, Audit, and verification discoverability as one evidence trail: make it obvious how to move from a workflow result to artifacts, from artifacts to audit records, from audit records to verification posture, and from verification posture to export/offline verification or anchoring actions where available; keep raw digests copyable but do not make long digests the primary visual content when a shorter stable identity or label is available.
- [ ] Shorten route key hints and footer wording so the TUI feels calm while remaining discoverable: primary route footers should show only the most useful keys for the current state, while exhaustive route commands remain available through command mode, fuzzy discovery, leader help, inspectors, or a dedicated help surface generated from the action graph.
- [ ] Tighten route labels, headings, badges, and status messages into consistent product language: prefer terms like `needs attention`, `blocked`, `waiting`, `ready`, `evidence available`, `approval required`, and `setup required` in primary panes; reserve protocol names, schema names, raw contract method names, and implementation terms for detail modes where they help debugging or verification.
- [ ] Preserve trust, attach, and performance-contract truth while polishing: do not create client-local optimistic progress, synthetic completion, local-only lifecycle inference, fake attestation confidence, or hidden degraded states; attach/reconnect/resume, waiting, blocked, degraded, and failed states must continue to reflect broker-owned lifecycle, run/session, watch, audit, and project-substrate surfaces.
- [ ] Fix TUI blockers and misleading product-truth issues discovered during walkthroughs before marking this phase complete: misleading wording, unavailable next actions, hidden blocked reasons, unclear approval consequences, missing evidence links, confusing setup remediation, or UI states that imply stronger execution/attestation/publication behavior than the implementation provides are blockers for this polish phase.
- [ ] Record non-blocking TUI polish follow-ups separately when they do not block the beta proof: visual refinements, optional theme changes, secondary keyboard customization, advanced filtering, additional inspector modes, or future collaboration/publishing affordances should be tracked as follow-up unless they are required for a new Linux user to coherently test the supported local workflow path.

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
