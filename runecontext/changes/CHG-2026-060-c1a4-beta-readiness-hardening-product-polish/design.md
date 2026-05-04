# Design

## Overview
This change is an alpha hardening umbrella for turning RuneCode's existing foundations into a coherent, useful, beta-ready Linux-first product slice.

It does not redefine the major architecture already in place. Instead, it sequences the remaining work needed to make the current architecture show up truthfully and usefully in normal operation.

The central rule of this lane is:

RuneCode should not claim beta readiness until one real workflow path runs through the real trusted and untrusted execution path, produces inspectable artifacts and audit evidence, and remains understandable to an operator using the normal product surfaces.

## Scope
This lane covers five connected concerns:

1. end-to-end workflow execution wiring
2. trusted `RunPlan` production adoption
3. truthful runtime-attestation posture handoff to `CHG-2026-054-6c1e-runtime-attestation-post-handshake-gating-v0`
4. product and TUI polish discovered during dogfooding
5. release-surface and verification-smoke-path alignment

## Execution Integration Goal

### Current Shape
The current implementation already provides:

- durable session and run state
- session execution trigger flows
- workflow-pack assets and workflow routing contracts
- trusted `RunPlan` compilation and persistence machinery
- runner kernel foundations and report schemas
- launcher and runtime-evidence foundations
- audit, artifact, and verification surfaces in the broker and TUI

The main remaining gap is production wiring.

### Required Product Shape
The required alpha.11 execution path is:

1. operator starts or attaches to the repo-scoped product via `runecode`
2. operator triggers a supported first-party RuneContext workflow through the normal product path
3. trusted code validates project substrate and execution preconditions
4. trusted code selects and adopts the authoritative built-in workflow assets and compiles the exact immutable `RunPlan`
5. trusted code persists the authoritative plan and its execution bindings
6. the actual runner or isolate-backed execution path starts from that plan
7. runner checkpoints and results flow back through the broker's real typed surfaces
8. operator-visible session, run, artifact, approval, and audit surfaces all reflect that real path

This lane is complete only when that path is real, not simulated by local-only state updates.

## Trusted RunPlan Adoption
`CompileAndPersistRunPlan` already exists as a trusted foundation. This lane makes production workflow execution consume it as the real authority path rather than leaving it mostly validated by tests.

The production path should make it obvious that:

- the workflow assets are selected by trusted code
- the compiled plan is persisted before execution
- the runner or isolate consumes the authoritative plan identity
- later run-state, gate-state, and evidence links resolve back to that exact plan

## First Useful Workflow Slice
This lane should require one useful first-party workflow slice rather than trying to complete every possible path at once.

Recommended minimum target:

- `change_draft` or `spec_draft` from prompt to produced artifact through the real execution path

Preferred target if scope remains manageable:

- `change_draft` or `spec_draft` plus `draft_promote_apply`, so the verified RuneContext lifecycle is also exercised through the same honest path

`approved_change_implementation` may remain follow-on work if needed, but only if beta is explicitly framed as planning- or drafting-first rather than coding-first.

## Runner Integration Goal
The runner kernel currently exposes transport seams that can still default to noop behavior outside explicit wiring. This lane should close that ambiguity for the real workflow path.

Completion shape:

- the actual workflow path uses a real broker transport for checkpoint and result reporting
- missing transport configuration is no longer the silent or default normal-operation story for a real run
- operator-visible run progress derives from real execution progress instead of only control-plane projection shortcuts

## Attestation Truthfulness
This lane does not replace `CHG-2026-054-6c1e-runtime-attestation-post-handshake-gating-v0`, but it treats that change as a required companion for truthful beta assurance.

The key integration rule is:

- do not present supported `attested` posture as the settled beta story until `CHG-2026-054-6c1e-runtime-attestation-post-handshake-gating-v0` lands

This lane should therefore keep product and UX surfaces aligned with the true current posture while alpha hardening is in progress.

## Product Polish Goal
Dogfooding should be part of the plan, not an afterthought.

This lane should capture polish work discovered while testing the real workflow path, especially in:

- run and session state clarity
- attach, reconnect, and resume ergonomics
- project-substrate remediation guidance
- approval visibility and follow-up cues
- audit, artifact, and verification discoverability
- route naming, wording, and operator confidence signals
- waiting, blocked, degraded, and failed state communication

The TUI is the highest-priority polish surface because it is the normal user-facing shell for the local product.

## Verification Smoke Path
This lane should require that the real workflow path also exercises the verification surfaces already present in the repository.

The alpha.11 smoke path should include:

- run one useful workflow
- inspect resulting runs, artifacts, and audit records in the TUI or broker surfaces
- capture evidence snapshot
- inspect at least one record inclusion result
- export a bundle and verify it offline
- exercise external anchoring when appropriate and available

The goal is not to finish every planned verification-plane feature here. The goal is to prove that beta ships with real evidence continuity from a real workflow path.

## Release-Surface Alignment
This lane should end with roadmap, docs, and messaging that match the real state of the product.

Specifically:

- roadmap entries should reflect alpha.11 as the hardening lane and beta.1 as the milestone outcome
- product docs should describe the actual useful workflow story and current assurance posture honestly
- help text and operator-facing wording should not imply a stronger end-to-end or attestation story than the code provides

## Exit Criteria
This alpha lane is complete when:

- one useful first-party workflow runs through the real trusted and untrusted path
- the path uses authoritative trusted `RunPlan` adoption in production
- run and session progress surfaces reflect real execution rather than only synthetic projection
- verification artifacts are generated and exercised from that real workflow path
- TUI and surrounding operator surfaces are polished enough that a new user can test the product coherently on Linux
- the beta story is honest about assurance and execution behavior
