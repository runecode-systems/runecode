## Summary
Close the remaining integration, truthfulness, and operator-experience gaps between RuneCode's currently implemented foundations and the first beta-ready product slice, while capturing the dogfooding and TUI polish needed to make that slice useful to real users on Linux.

The beta-ready slice must prove the local canonical RuneContext lifecycle and productive workflow loop end to end: project-substrate lifecycle, change and spec drafting, draft promote/apply into canonical RuneContext files, approved implementation from reviewed inputs, and evidence-backed operator inspection through the normal product surfaces.

## Problem
RuneCode now has most of the major foundations needed for a first beta story: verified RuneContext project lifecycle, direct-credential remote model access, local product lifecycle management, workflow-pack assets, signed runtime-asset admission, attestation evidence seams, external audit anchoring, and portable verification evidence surfaces.

The remaining gaps are no longer primarily missing primitives. They are missing product integration and honest operator outcomes.

Today the repo still shows a mismatch between what the product foundations imply and what a beta user can truthfully do:

- session execution creates durable run and session state, but the real path to useful runner- or isolate-backed work is not yet wired end to end
- trusted `RunPlan` compilation exists, but production execution paths do not yet clearly adopt it as the authoritative entry to useful work
- the runner kernel still exposes noop/default transport seams rather than an obviously wired real broker-reporting path in normal operation
- project-substrate init, upgrade, and validation surfaces exist, but the beta needs to prove RuneCode owns that canonical RuneContext lifecycle through the normal product path
- first-party workflow assets exist for `change_draft`, `spec_draft`, `draft_promote_apply`, and `approved_change_implementation`, but beta cannot rely on asset existence alone; those operations need to run through the real trusted and untrusted path
- supported `attested` posture must remain truthful and depend on the post-handshake trusted verification posture from `CHG-2026-054-6c1e-runtime-attestation-post-handshake-gating-v0`
- the TUI and surrounding local-product UX need dogfooding-driven polish so the first beta is understandable, testable, and useful rather than merely impressive in architecture
- roadmap, docs, and product messaging need one explicit pre-beta hardening lane so beta remains a milestone outcome rather than a vague bucket for leftover integration work

Without a dedicated alpha hardening lane, RuneCode risks declaring beta too early, with a product that is rich in control-plane machinery and verification surfaces but still one honest end-to-end workflow short of the usability bar.

## Proposed Change
- Create one alpha.11 umbrella project lane that captures the remaining beta-readiness hardening and product polish work.
- Treat this lane as the integration and dogfooding bridge between implemented foundations and the `v0.1.0-beta.1` milestone outcome.
- Close the remaining end-to-end execution gap from session trigger to real trusted `RunPlan` adoption, runner or isolate launch, runner checkpoint and result reporting, and durable operator-visible state.
- Require the supported beta RuneContext workflow slice to be runnable and inspectable through the normal product path: `change_draft`, `spec_draft`, `draft_promote_apply`, and `approved_change_implementation`.
- Tighten the `approved_change_implementation` input-set identity contract before beta so the bound artifact digest and the semantic input-set digest are distinct, recomputed by trusted code, and fail closed on drift.
- Require canonical RuneContext project-substrate lifecycle proof through RuneCode-owned surfaces: inspect or adopt existing substrate, initialize missing substrate through preview/apply, upgrade supported older substrate through preview/apply, and validate/status the resulting posture.
- Track the production adoption of trusted `RunPlan` compilation rather than leaving it as a largely test-proven foundation seam.
- Track the replacement of effectively noop runner transport defaults with real broker integration in the actual workflow path.
- Fold in the truthful attestation-posture correction from `CHG-2026-054-6c1e-runtime-attestation-post-handshake-gating-v0` as a required pre-beta hardening companion.
- Keep workflow-path dogfooding, TUI polish, and operator-truth improvements aligned with the reviewed performance-contract discipline in `CHG-2026-053-9d2b-performance-baselines-verification-gates-v0`, especially around attach/resume truth, waiting-state communication, and avoidance of client-local shortcuts that would make measured product surfaces less honest.
- Explicitly capture TUI and operator polish discovered while dogfooding the real workflow path, especially around run state clarity, attach or reconnect behavior, remediation cues, approval and audit discoverability, and overall usability.
- Treat TUI interaction latency as a beta-blocking polish concern alongside wording and visual hierarchy: rapid typing in search surfaces, arrow-key traversal, and leader-key navigation must remain responsive under realistic datasets without dropped input caused by synchronous filtering, repeated route-surface rendering, or key-path persistence work.
- Treat the remaining TUI presentation work as a beta-blocking product gate rather than optional visual cleanup: the current TUI has strong Bubble Tea, Bubbles, Lip Gloss, route, inspector, command, and broker-state foundations, but the full snapshot audit still shows heavy shell chrome and footer pressure, rendered-mode control-plane/debug leakage in primary panes, Action Center and Dashboard hierarchy gaps, setup/evidence routes that still read like state dumps, and missing overlay plus responsive audit proof. Closure therefore requires a calmer shell, professional modal overlays, stricter rendered/structured/raw separation, route-by-route product-language cleanup, display-width-safe styling, and captured desktop plus compact/mobile walkthrough evidence before the TUI can be called polished.
- Require deterministic TUI interaction-performance coverage in CI before this lane can close: benchmark keypress-heavy update/filter/view paths, add no-dropped-keys tests for fast scripted overlay input, and gate regressions with a stable Linux benchmark job rather than leaving interaction responsiveness to manual dogfooding alone.
- Align roadmap and product-surface messaging with the real shipped state once the honest workflow path exists.
- Require the alpha lane to exercise verification artifacts on the real workflow path so beta ships with strong evidence continuity instead of a later degraded verification posture.
- Keep git remote publication out of the required beta close gate unless beta messaging explicitly claims prompt-to-PR, push, or team collaboration. Local canonical lifecycle and implementation are required; remote publication remains adjacent follow-on scope.

## Why Now
The repository is no longer blocked mainly on basic platform capability.

It is now at the point where the most important pre-beta work is to make the implemented pieces behave like one coherent product and to prove that the resulting path is useful in practice.

Doing that as an explicit `v0.1.0-alpha.11` lane keeps the beta milestone clean:

- alpha.11 becomes the hardening and dogfooding release
- beta.1 remains the first usable release outcome

That split is easier to reason about than continuing to leave integration and polish work implicitly hidden under beta itself.

## Assumptions
- The current foundations for project lifecycle, model access, workflow-pack assets, local broker lifecycle, signed runtime assets, attestation evidence, audit evidence export, and anchoring are strong enough that the main remaining risk is integration quality rather than missing architecture.
- RuneCode should ship beta only when the local canonical workflow loop runs through the honest trusted and untrusted execution path and is inspectable through the normal product surfaces.
- The required local canonical workflow loop includes project-substrate lifecycle, `change_draft`, `spec_draft`, `draft_promote_apply`, and `approved_change_implementation`.
- `approved_change_implementation` may include required RuneContext lifecycle metadata updates, such as `tasks.md`, `status.yaml`, verification status, roadmap, or release-note updates, when those updates are part of the approved implementation input set; a separate fifth workflow operation is not required for this lane.
- `approved_change_implementation` must preserve two digest domains: the routing-bound artifact digest identifies the exact stored payload bytes, while `input_set_digest` identifies the canonical input-set body with `input_set_digest` omitted.
- TUI and product polish discovered while dogfooding are legitimate alpha hardening work and should be planned explicitly rather than treated as incidental cleanup.
- TUI polish means more than using Bubble Tea and Lip Gloss primitives. The beta bar requires a professional operator experience: calm hierarchy, reduced shell chrome and footer noise, product language in primary panes, debug detail moved to structured/raw modes, clear state cards, intentional command discovery, responsive desktop/compact/mobile layouts, and honest broker-owned progress without synthetic optimism.
- TUI polish also requires strong interaction performance. The TUI should feel immediate under normal operator input, which means keeping Bubble Tea `Update` and render hot paths small, precomputing searchable text, rendering only visible overlay rows, avoiding duplicated route-surface work, and moving persistence off critical key paths where needed.
- Verification artifacts generated from the real workflow path must remain first-class deliverables of this lane so later verification work strengthens rather than backfills the beta story.
- Product polish in this lane must improve operator clarity without undermining the authoritative broker-owned and persisted surfaces that `CHG-053` measures and protects.

## Out of Scope
- Replacing the broader beta milestone with a new version target.
- Replanning the full verification-plane foundation, performance-baseline program, or cross-machine replication roadmap.
- Treating polish work as a reason to expand the trust boundary or create new product-truth surfaces.
- Adding an independent fifth built-in workflow operation for lifecycle close or promotion metadata before beta.
- Requiring git remote push, pull-request creation, or team-collaboration publishing as a beta blocker unless beta messaging is expanded to claim remote publication.
- Completing the broader implementation-track decomposition and isolated-worktree execution roadmap from `CHG-2026-051-4b9d-implementation-track-decomposition-git-worktree-execution-v0` beyond what is needed to prove the approved implementation workflow through the shared beta path.

## Impact
If completed, this change gives RuneCode one explicit alpha lane to finish the work that matters most before beta:

- canonical RuneContext project-substrate lifecycle proof
- change and spec drafting through the honest workflow path
- reviewed draft promote/apply into canonical RuneContext files
- approved implementation through the honest workflow path
- truthful runtime assurance posture
- dogfooded, visually reviewed, and professionally polished TUI and operator surfaces
- release messaging that matches reality
- real workflow-generated verification artifacts that can anchor later trust improvements

That should let `v0.1.0-beta.1` mean a usable product milestone instead of an architectural aspiration.
