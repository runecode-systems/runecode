## Summary
Close the remaining integration, truthfulness, and operator-experience gaps between RuneCode's currently implemented foundations and the first beta-ready product slice, while capturing the dogfooding and TUI polish needed to make that slice useful to real users on Linux.

## Problem
RuneCode now has most of the major foundations needed for a first beta story: verified RuneContext project lifecycle, direct-credential remote model access, local product lifecycle management, workflow-pack assets, signed runtime-asset admission, attestation evidence seams, external audit anchoring, and portable verification evidence surfaces.

The remaining gaps are no longer primarily missing primitives. They are missing product integration and honest operator outcomes.

Today the repo still shows a mismatch between what the product foundations imply and what a beta user can truthfully do:

- session execution creates durable run and session state, but the real path to useful runner- or isolate-backed work is not yet wired end to end
- trusted `RunPlan` compilation exists, but production execution paths do not yet clearly adopt it as the authoritative entry to useful work
- the runner kernel still exposes noop/default transport seams rather than an obviously wired real broker-reporting path in normal operation
- attested posture still has one ordering gap before post-handshake trusted verification closes the claim fully
- the TUI and surrounding local-product UX need dogfooding-driven polish so the first beta is understandable, testable, and useful rather than merely impressive in architecture
- roadmap, docs, and product messaging need one explicit pre-beta hardening lane so beta remains a milestone outcome rather than a vague bucket for leftover integration work

Without a dedicated alpha hardening lane, RuneCode risks declaring beta too early, with a product that is rich in control-plane machinery and verification surfaces but still one honest end-to-end workflow short of the usability bar.

## Proposed Change
- Create one alpha.11 umbrella project lane that captures the remaining beta-readiness hardening and product polish work.
- Treat this lane as the integration and dogfooding bridge between implemented foundations and the `v0.1.0-beta.1` milestone outcome.
- Close the remaining end-to-end execution gap from session trigger to real trusted `RunPlan` adoption, runner or isolate launch, runner checkpoint and result reporting, and durable operator-visible state.
- Require at least one honest useful first-party RuneContext workflow slice to be runnable and inspectable through the normal product path.
- Track the production adoption of trusted `RunPlan` compilation rather than leaving it as a largely test-proven foundation seam.
- Track the replacement of effectively noop runner transport defaults with real broker integration in the actual workflow path.
- Fold in the truthful attestation-posture correction from `CHG-2026-054-6c1e-runtime-attestation-post-handshake-gating-v0` as a required pre-beta hardening companion.
- Keep workflow-path dogfooding, TUI polish, and operator-truth improvements aligned with the reviewed performance-contract discipline in `CHG-2026-053-9d2b-performance-baselines-verification-gates-v0`, especially around attach/resume truth, waiting-state communication, and avoidance of client-local shortcuts that would make measured product surfaces less honest.
- Explicitly capture TUI and operator polish discovered while dogfooding the real workflow path, especially around run state clarity, attach or reconnect behavior, remediation cues, approval and audit discoverability, and overall usability.
- Align roadmap and product-surface messaging with the real shipped state once the honest workflow path exists.
- Require the alpha lane to exercise verification artifacts on the real workflow path so beta ships with strong evidence continuity instead of a later degraded verification posture.

## Why Now
The repository is no longer blocked mainly on basic platform capability.

It is now at the point where the most important pre-beta work is to make the implemented pieces behave like one coherent product and to prove that the resulting path is useful in practice.

Doing that as an explicit `v0.1.0-alpha.11` lane keeps the beta milestone clean:

- alpha.11 becomes the hardening and dogfooding release
- beta.1 remains the first usable release outcome

That split is easier to reason about than continuing to leave integration and polish work implicitly hidden under beta itself.

## Assumptions
- The current foundations for project lifecycle, model access, workflow-pack assets, local broker lifecycle, signed runtime assets, attestation evidence, audit evidence export, and anchoring are strong enough that the main remaining risk is integration quality rather than missing architecture.
- RuneCode should ship beta only when at least one real useful workflow runs through the honest trusted and untrusted execution path and is inspectable through the normal product surfaces.
- TUI and product polish discovered while dogfooding are legitimate alpha hardening work and should be planned explicitly rather than treated as incidental cleanup.
- Verification artifacts generated from the real workflow path must remain first-class deliverables of this lane so later verification work strengthens rather than backfills the beta story.
- Product polish in this lane must improve operator clarity without undermining the authoritative broker-owned and persisted surfaces that `CHG-053` measures and protects.

## Out of Scope
- Replacing the broader beta milestone with a new version target.
- Replanning the full verification-plane foundation, performance-baseline program, or cross-machine replication roadmap.
- Treating polish work as a reason to expand the trust boundary or create new product-truth surfaces.
- Promising multiple equally complete first-party workflow families before beta.

## Impact
If completed, this change gives RuneCode one explicit alpha lane to finish the work that matters most before beta:

- one honest useful end-to-end workflow path
- truthful runtime assurance posture
- dogfooded and more coherent TUI and operator surfaces
- release messaging that matches reality
- real workflow-generated verification artifacts that can anchor later trust improvements

That should let `v0.1.0-beta.1` mean a usable product milestone instead of an architectural aspiration.
