# Design

## Overview
This change is an alpha hardening umbrella for turning RuneCode's existing foundations into a coherent, useful, beta-ready Linux-first product slice.

It does not redefine the major architecture already in place. Instead, it sequences the remaining work needed to make the current architecture show up truthfully and usefully in normal operation.

The central rule of this lane is:

RuneCode should not claim beta readiness until the local canonical RuneContext lifecycle and productive workflow loop run through the real trusted and untrusted execution path, produce inspectable artifacts and audit evidence, and remain understandable to an operator using the normal product surfaces.

This lane also coordinates directly with `CHG-2026-053-9d2b-performance-baselines-verification-gates-v0` for the surfaces that beta users actually experience. Dogfooding-driven polish must improve those surfaces without making them less authoritative or less measurable.

## Scope
This lane covers six connected concerns:

1. end-to-end workflow execution wiring
2. trusted `RunPlan` production adoption
3. canonical RuneContext project-substrate lifecycle proof
4. truthful runtime-attestation posture handoff to `CHG-2026-054-6c1e-runtime-attestation-post-handshake-gating-v0`
5. product and TUI polish discovered during dogfooding
6. release-surface and verification-smoke-path alignment

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
2. operator inspects, adopts, initializes, or upgrades canonical RuneContext project substrate through broker-owned product surfaces when needed
3. operator triggers a supported first-party RuneContext workflow through the normal product path
4. trusted code validates project substrate and execution preconditions
5. trusted code selects and adopts the authoritative built-in workflow assets and compiles the exact immutable `RunPlan`
6. trusted code persists the authoritative plan and its execution bindings
7. the actual runner or isolate-backed execution path starts from that plan
8. runner checkpoints and results flow back through the broker's real typed surfaces
9. operator-visible session, run, artifact, approval, and audit surfaces all reflect that real path

This lane is complete only when that path is real, not simulated by local-only state updates.

## Trusted RunPlan Adoption
`CompileAndPersistRunPlan` already exists as a trusted foundation. This lane makes production workflow execution consume it as the real authority path rather than leaving it mostly validated by tests.

The production path should make it obvious that:

- the workflow assets are selected by trusted code
- the compiled plan is persisted before execution
- the runner or isolate consumes the authoritative plan identity
- later run-state, gate-state, and evidence links resolve back to that exact plan

## Required Beta Workflow Slice
This lane requires the complete local canonical RuneContext workflow loop rather than a single draft-only demonstration.

Required operations:

- `change_draft` from prompt to typed change-draft artifact through the real execution path
- `spec_draft` from prompt to typed spec-draft artifact through the same real execution path
- `draft_promote_apply` for a reviewed change draft into canonical `runecontext/changes/`
- `draft_promote_apply` for a reviewed spec draft into canonical `runecontext/specs/`
- `approved_change_implementation` from one reviewed implementation input set containing one or more approved change/spec inputs by exact digest

The proof chain should show that planning, canonical RuneContext mutation, and local implementation mutation all use the same broker-owned workflow authority model rather than separate product-local shortcuts.

`approved_change_implementation` should be allowed to update required RuneContext lifecycle metadata when the approved input set calls for it. Examples include `tasks.md`, `status.yaml`, verification status, roadmap entries, and release-note surfaces. This lane should not add a separate fifth workflow operation for lifecycle close or metadata promotion unless later reviewed work needs it as an independently runnable command.

The broader implementation-track decomposition and isolated-worktree execution roadmap remains follow-on unless a narrow part is required to make this approved implementation proof real.

### Approved Implementation Input-Set Identity
The beta workflow slice must make the approved implementation input-set identity contract explicit before future implementation, collaboration, or git publication features depend on it.

The contract has two digest domains:

- `workflow_routing.bound_input_artifacts[].artifact_digest` identifies the exact stored canonical JSON artifact bytes supplied to the run.
- `implementation_input_set.input_set_digest` identifies the semantic input-set body: the canonical JSON object after omitting `input_set_digest` itself.

Trusted broker validation must recompute the semantic digest from the stored payload and require it to match `implementation_input_set.input_set_digest`. Validation must also keep using the bound artifact digest for artifact retrieval and byte-level identity. A stored artifact can therefore be addressed by one digest while carrying a self-excluded semantic identity that is stable across storage wrappers and safe to use as the approved input-set identity.

Run, audit, and projection code should avoid conflating the names. Where both are relevant, use `input_set_artifact_digest` for the routing-bound artifact identity and `input_set_digest` for the recomputed semantic input-set identity.

## Project-Substrate Lifecycle Proof
Beta owns the canonical RuneContext lifecycle for a repository. This lane therefore requires a normal product proof for project substrate, not only a workflow proof.

Required lifecycle coverage:

- inspect and report current project-substrate posture
- adopt compatible existing substrate without silently rewriting it
- initialize missing substrate through explicit preview/apply
- upgrade compatible older substrate through explicit preview/apply
- re-run validation and status after apply
- keep normal productive workflow execution blocked when substrate posture is missing, invalid, non-verified, or unsupported

These flows remain setup and remediation lifecycle, not built-in productive workflow operations. They still must be broker-owned, typed, auditable where apply occurs, and visible through TUI or CLI surfaces.

## Runner Integration Goal
The runner kernel currently exposes transport seams that can still default to noop behavior outside explicit wiring. This lane should close that ambiguity for the real workflow path.

Completion shape:

- the actual workflow path uses a real broker transport for checkpoint and result reporting
- missing transport configuration is no longer the silent or default normal-operation story for a real run
- operator-visible run progress derives from real execution progress instead of only control-plane projection shortcuts

## Attestation Truthfulness
This lane does not replace `CHG-2026-054-6c1e-runtime-attestation-post-handshake-gating-v0`, but it treats that change as a required companion for truthful beta assurance.

The key integration rule is:

- do not present supported `attested` posture as the settled beta story until `CHG-2026-054-6c1e-runtime-attestation-post-handshake-gating-v0` is implemented, verified, and integrated into the supported path

This lane should therefore keep product and UX surfaces aligned with the true current posture while alpha hardening is in progress.

## Git Remote Publication Boundary
Git remote mutation is not required for this beta close gate unless product messaging explicitly claims remote publication, prompt-to-PR, push, or team collaboration.

The useful beta story can be local-first:

- repo-scoped product lifecycle works
- project-substrate lifecycle works
- planning artifacts are generated
- canonical RuneContext files are mutated through audited promote/apply
- approved implementation mutates the local workspace through the real workflow path
- evidence, audit, and TUI surfaces make the work inspectable

If beta messaging later expands to publishing or collaboration, add a narrow git remote `prepare -> exact approval -> execute` smoke path using the existing gateway contracts. That is intentionally not part of the required CHG-060 close shape.

## Planning-Time Implementation Findings
The planning assessment found existing seams that this lane should close or explicitly replace:

- `internal/brokerapi/local_api_session_execution_trigger_bridge.go` currently behaves as a synthetic bridge by recording a checkpoint and setting a run active rather than compiling, persisting, and launching from an authoritative plan.
- `internal/brokerapi/local_api_session_execution_trigger_binding.go` initializes run status and runtime facts for session execution, but that is not the same as real runner or isolate launch.
- `runner/src/broker-client.ts` still exposes noop runner broker-client behavior that must not be the normal supported workflow path.
- `runner/package.json` does not expose an obvious normal product runner launch entrypoint for the supported path.
- `internal/brokerapi/local_api_run_summary_ops.go` still has artifact-inferred workflow identity behavior that should become plan-authoritative for supported path projections.
- `cmd/runecode-tui/route_chat_state.go` should route supported beta operations intentionally rather than relying on an accidental or misleading default workflow operation.

The positive foundation is also clear: trusted `RunPlan` compile/persist, active plan selection, built-in workflow catalog authority, broker runner report operations, project-substrate lifecycle APIs, and evidence/export/offline verification surfaces already exist and should be integrated rather than replaced.

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

TUI acceptance is intentionally dogfooding-gated rather than fully preplanned. Issues found during walkthroughs of the required paths should be captured, blockers and misleading product-truth issues should be fixed before closure, and non-blocking polish can be recorded as follow-up.

That polish should stay aligned with the reviewed performance-contract discipline in `CHG-053`, especially:

- attach, reconnect, and resume surfaces should continue to reflect broker-owned lifecycle truth rather than client-local optimistic shortcuts
- waiting, blocked, degraded, and failed states should remain operator-visible without reintroducing misleading high-activity rendering paths or synthetic progress cues
- dogfooding fixes should preserve the same authoritative surfaces that the MVP performance gates measure rather than optimizing around those gates with less truthful UI behavior

## Verification Smoke Path
This lane should require that the real workflow path also exercises the verification surfaces already present in the repository.

The alpha.11 smoke path should include:

- run project-substrate inspect/adopt/init/upgrade coverage as applicable for deterministic fixtures
- run `change_draft` and `spec_draft` through the real workflow path
- promote/apply a reviewed change draft and a reviewed spec draft into canonical RuneContext files
- run `approved_change_implementation` from a reviewed implementation input set
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
- the blank or incomplete alpha.11 roadmap feature-change wording should be resolved
- `CHG-2026-054-6c1e-runtime-attestation-post-handshake-gating-v0` should be represented consistently as a verified/integrated beta closure dependency when assurance wording depends on it
- product docs should describe the actual useful workflow story and current assurance posture honestly
- help text and operator-facing wording should not imply a stronger end-to-end or attestation story than the code provides

## Exit Criteria
This alpha lane is complete when:

- project-substrate lifecycle is proven through broker-owned product surfaces for inspect/adopt/init/upgrade/validate/status cases
- `change_draft`, `spec_draft`, `draft_promote_apply`, and `approved_change_implementation` run through the real trusted and untrusted path for the supported beta slice
- the path uses authoritative trusted `RunPlan` adoption in production
- run and session progress surfaces reflect real execution rather than only synthetic projection
- verification artifacts are generated and exercised from that real workflow path
- TUI and surrounding operator surfaces are polished enough that a new user can test the product coherently on Linux
- the beta story is honest about assurance and execution behavior
- workflow-path and TUI polish remain compatible with the authoritative surfaces and honest measurement boundaries frozen by `CHG-053`
- git remote publication is either explicitly out of beta messaging or covered by a separate reviewed smoke path before any publishing claim is made
