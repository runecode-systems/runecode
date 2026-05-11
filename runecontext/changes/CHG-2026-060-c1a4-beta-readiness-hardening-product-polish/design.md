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

The TUI polish goal for this lane is to move the current interface from a dense development/debug console toward a polished, professional, production-feeling operator product. The existing TUI foundations are strong: a unified shell, route workbenches, inspectors, command mode, palette discovery, themes, live watch projection, and broker-owned state surfaces. The remaining product gap is presentation hierarchy and operator confidence. Primary screens should explain what is happening, whether the operator needs to act, and what to do next before exposing protocol, schema, or debug-level detail.

The preserved full snapshot audit reinforced that goal. A desktop `full-audit` bundle covering Dashboard, Chat, Runs, Approvals, Action Center, Audit, Status, Model Providers, Git Setup, and Git Remote confirmed that the product now has real route structure and operator intent, but it also showed that the default shell still spends too much space on chrome and footer surfaces, many rendered panes still read like styled control-plane views, and the current deterministic audit coverage still omits Artifacts, overlay surfaces, and compact/mobile layouts.

This lane should capture polish work discovered while testing the real workflow path, especially in:

- run and session state clarity
- attach, reconnect, and resume ergonomics
- project-substrate remediation guidance
- approval visibility and follow-up cues
- audit, artifact, and verification discoverability
- route naming, wording, and operator confidence signals
- waiting, blocked, degraded, and failed state communication

The TUI is the highest-priority polish surface because it is the normal user-facing shell for the local product.

The first once-over identified these concrete polish directions:

- Treat `Action Center` as the operator home for attention, blocked work, approvals, failed or waiting workflows, substrate remediation, audit/runtime degradation, and follow-up cues.
- Keep `Dashboard` as a calm executive overview that summarizes system health, current workflow posture, and the most important next action rather than a dense dump of readiness fields.
- Make every route answer the same operator questions: what is happening, is it healthy, am I blocked, what should I do next, and where is the evidence?
- Prefer product language in primary panes and move protocol names, raw contract terms, and debug-level identifiers into inspectors, detail modes, or copy actions.
- Rework global chrome so the top bar and footer feel calm: show route, product posture, sync truth, and a small set of primary actions without repeating long trust-boundary or shortcut text on every screen.
- Make loading, empty, waiting, blocked, degraded, failed, resumed, completed, and approval-required states use one state-card pattern with a state label, reason, next action, and shortcut or route cue.
- Make project-substrate setup and remediation feel like a guided broker-owned flow: inspect current posture, preview init or upgrade, review the planned mutation, apply intentionally, then validate/status again.
- Make Chat and run progress feel alive but honest by surfacing broker-owned execution stages such as waiting, plan compiled, runner active, approval required, artifact ready, failed, and completed without inventing optimistic progress.
- Make evidence trails obvious across runs, sessions, approvals, artifacts, audit records, and verification actions so users can follow a workflow result to the artifacts and audit proof behind it.
- Shorten primary keyboard help and keep exhaustive shortcuts discoverable through command discovery, leader help, or route detail surfaces so the footer stays readable.

### TUI Once-Over Findings
The focused source review of the current TUI implementation found that the product is partway through this polish goal, but not yet at the professional operator-product bar required for beta. The code already uses a real Bubble Tea shell, route models, route workbenches, inspectors, command and leader surfaces, live watch projection, themes, Lip Gloss-styled panes, Bubbles textarea/viewport/help/spinner primitives, and broker-owned state. The gap is no longer whether the implementation has TUI foundations; the gap is whether the default presentation feels calm, coherent, and designed rather than like a dense control-plane console.

### TUI Interaction Performance Findings
The dogfooding review also found that the TUI currently feels sluggish during common interaction loops. This is beta-blocking product polish because delayed or dropped input makes the product feel unproductive even when the underlying broker state is correct.

The source review identified these concrete causes:

- overlay search currently performs synchronous per-keystroke filtering on the Bubble Tea event loop for the command palette and session switcher, while repeatedly lowercasing and scanning large entry sets in hot input paths
- overlay list rendering formats every match before the bounded list window trims to the visible rows, so short overlays still pay formatting cost for the full result set
- overlay typing still re-renders the full shell workbench behind the modal surface, including route surface generation, pane layout, sidebar, inspector, and frame scrubbing work that does not change for most overlay keystrokes
- shell surface generation is duplicated in hot paths: layout planning, overlay-height calculation, focus traversal, and final `View` rendering can all trigger repeated route `ShellSurface` work for a single key press
- leader navigation recomputes bindings and next-choice sets more often than necessary, including repeated scans and sorting during open/step flows
- some interactive navigation and workbench actions still perform synchronous persistence writes from key-driven paths, which can introduce visible hitches when the local filesystem is slow

These findings align with Bubble Tea's execution model: `Update` and render are serialized on the main event loop, so any synchronous filter, render, layout, or file-write cost directly increases key-to-render latency and can make rapid input feel dropped.

### TUI Interaction Performance Recommendations
The TUI polish plan should therefore treat interaction latency as a first-class product-quality requirement, not as a later optimization pass.

Recommended closure order:

1. Make hot key paths cheap.
   Keep text-entry, leader, focus-traversal, and navigation updates close to state mutation only. Avoid repeated route-surface generation, repeated layout planning, and any synchronous file I/O from those key paths.

2. Precompute and reuse searchable text.
   Palette and quick-switch entries should carry normalized search text so typing does not repeatedly lowercase and concatenate fields. Match buffers should reuse backing storage rather than forcing unnecessary allocation churn.

3. Render only visible rows for overlay lists.
   Command palette, session switcher, and leader-help lists should format only the bounded visible window plus gap markers rather than building rows for the full match set.

4. Cache unchanged shell frame work while overlays are active.
   When the user is typing into command palette, session switcher, or leader help, the background workbench should be reused until route state, watch state, window size, theme, or focus requires invalidation.

5. Remove unnecessary leader recomputation.
   Leader mode should avoid repeated binding scans and sorting for unchanged prefixes; open/start flows should not rebuild the same choice set multiple times.

6. Move persistence off the critical input path.
   Workbench-state persistence should be queued or coalesced so route/layout actions remain responsive without weakening durability expectations.

7. Add deterministic latency-focused tests and benchmarks.
   TUI polish is not complete until the repository has repeatable benchmarks and regression tests for keypress-heavy interactions and a CI job that makes regressions visible.

The current implementation partially satisfies the CHG-060 intent, but the full snapshot audit clarifies the remaining gaps:

- `Dashboard` is the best-aligned route. It now has an executive state card, high-value counts, and next-action language, but it still shares the screen with too much shell chrome and still repeats state/badge language that should feel calmer and more intentional.
- `Action Center` is the strongest triage surface and should become the operator home for follow-up, but selected cards are still too dense and the inspector largely repeats the same reason, impact, owner/action, and target text as raw key/value detail.
- `Chat` has the strongest route-level alignment around active session, composer state, broker-owned execution state, and linked evidence counts, but the route still feels crowded because active-session state, workflow state, approval state, session directory, mode tabs, inspector, route keys, and shell footer all compete at once.
- `Runs` keeps outcomes, coordination, approvals, artifacts, and audit evidence together, but rendered content still leaks implementation vocabulary such as projection or plan-authority terms, backend labels, approval profile detail, and raw posture taxonomy before the route has cleanly answered the operator's first questions.
- `Approvals` distinguishes approval-required, pending, resolved, expired, unsupported, lifecycle, policy, and trigger cues, but the rendered view still surfaces policy codes, trigger codes, gate bindings, and bound-scope detail too early for a non-debug operator.
- `Artifacts`, `Audit`, and verification still need to read as one evidence trail. The product has useful continuity and copyable raw identifiers, but digest labels, event names, verification codes, anchoring posture terms, and full raw identities still dominate too much of the primary visual hierarchy.
- `Status` and project-substrate remediation have the right broker-owned lifecycle concepts, but the main route still reads too much like a diagnostic dump because subsystem booleans, protocol/version details, compatibility tokens, preview handles, and normal-operation flags remain in the primary pane.
- `Model Providers`, `Git Setup`, and `Git Remote` remain the least polished product surfaces. They still read like broker state dumps because rendered views expose provider family, endpoint, profile IDs, auth modes, bootstrap modes, policy booleans, prepared mutation IDs, long request or approval hashes, credential leases, lifecycle reason codes, and derived git metadata before the operator story is clear.
- The shell chrome is improved from a raw debug frame, but the snapshot audit shows that top status, sync strip, pane spacer, bottom strip, status row, footer help, and the repeated broker-truth sentence still consume too much vertical space. The workbench feels dense before route content begins.
- Overlay behavior is directionally right in code, but the deterministic route bundle did not yet prove command palette, session switcher, leader help, inspector sheet, sidebar drawer, or quit confirmation as polished modal surfaces. Those surfaces remain an explicit TUI acceptance gap until they are captured and reviewed.
- The implementation uses Lip Gloss styles and tokens, but the styling is still mostly utility-level: bordered panes, badges, muted text, and surface colors. It does not yet deliver a strong visual system with accent rails, differentiated message blocks, sparse typography, focused cards, subdued secondary text, and consistent color semantics across all routes and responsive layouts.

The result is a TUI that is operationally useful and honest, but still short of the desired polished/professional experience. The remaining work should be treated as beta-blocking product polish, not optional beautification, because the TUI is the normal local product shell and the main place where a new Linux user will decide whether RuneCode feels understandable and trustworthy.

### Professional TUI Target
The target experience should feel closer to a high-quality terminal product than a protocol inspector. The reference direction is a dark, spacious workbench with strong text hierarchy, restraint, and a few high-confidence color accents. Screens should look intentionally composed even when the underlying state is complex.

The target visual language is:

- dark base surface with elevated panels for prompts, summaries, command palettes, and long-form detail
- a restrained accent palette: green for verified/successful/ready, amber for attention/waiting, red or magenta for blocked/failed/danger, purple for active planning/commands, cyan or blue for links/evidence/navigation, gray for secondary metadata
- high-contrast but low-noise typography: primary status in clear text, secondary facts dimmed, raw IDs shortened unless selected or copied
- one prominent operator card per route that answers: what is happening, is it healthy, am I blocked, what should I do next, and where is the evidence
- visual accent rails or sidebars for active message blocks, selected work items, approval-required cards, and command input rather than wrapping every object in heavy borders
- fewer always-visible words in chrome; route-specific detail should live in the route body, inspector, command palette, or copy actions, and the footer should not repeat trust-boundary or shortcut prose that overwhelms the route
- focused overlays that appear as centered, elevated panels with optional background dimming instead of appended blocks
- compact command discovery similar to the inspiration: title, escape hint, search field, grouped suggestions, selected row with strong contrast, and right-aligned shortcuts or command IDs
- diff/log/evidence views that use colored line blocks and columns intentionally, while keeping trust-sensitive raw content redacted and secondary unless selected

The target interaction model is:

- start at `Action Center` when there is any operator follow-up; otherwise start at `Dashboard` or show a Dashboard card whose primary call-to-action is Action Center
- make `Action Center` the triage home, not just another route: all blocked, waiting, degraded, approval, stale, setup, and evidence-health issues should be visible there first
- keep `Dashboard` calm and non-action-heavy: a single executive state, a small count strip, the current workflow posture, and the next best action
- keep `Chat` as the active work loop: session, composer, current broker-owned execution state, and evidence links
- keep `Runs`, `Approvals`, `Artifacts`, and `Audit` as drill-down evidence workbenches with rendered, structured, and raw modes
- keep `Status` as the product lifecycle and project-substrate remediation route
- keep `Model Providers`, `Git Setup`, and `Git Remote` behind product-language setup/review cards, with raw broker fields moved to structured/raw detail modes

### Bubble Tea, Bubbles, And Lip Gloss Standards
The TUI already implements Bubble Tea and Lip Gloss, but the polish closure should raise the implementation to a stricter product-quality standard.

Bubble Tea expectations:

- preserve the `tea.Model`/`Init`/`Update`/`View` architecture and keep all blocking broker, file, or network work inside `tea.Cmd`
- keep route models focused on state transitions and route-specific rendering; avoid hidden global side effects during `View`
- keep `View` deterministic and side-effect free so performance and snapshot-style tests remain meaningful
- keep async polling honest: use broker watch state, route reload commands, and timed ticks only to reflect real broker-owned state, never to create client-local optimistic completion
- keep keyboard ownership explicit for text entry, secret entry, overlays, command mode, and normal route navigation
- keep window-size handling and route viewport resizing centralized so narrow, medium, and wide layouts remain predictable
- keep mouse capture optional and reversible for text selection workflows
- keep route activation, palette actions, and reference jumps typed and non-destructive

Bubbles expectations:

- use `textarea` for multiline composer behavior instead of ad-hoc line editing
- use `viewport` or equivalent bounded long-form primitives for transcripts, diffs, logs, artifacts, and audit records rather than clipping raw strings manually
- use `help`/`key` for generated concise help where possible, and keep exhaustive key discovery in command/leader/help surfaces
- use `spinner` or small activity indicators only for real in-flight commands or watch activity, not to imply progress that the broker has not reported

Lip Gloss expectations:

- build a small design-token system before adding more one-off styles: surfaces, foregrounds, accents, state colors, borders, selected rows, dimmed text, code/digest text, links, and destructive actions
- prefer composable card, rail, row, pill, command-palette, and evidence-trail components over per-route string concatenation
- use `lipgloss.Width`, ANSI-aware truncation, and display-cell-aware clipping for styled and wide-character content; do not truncate styled rows by raw rune count when alignment matters
- use `JoinHorizontal` and `JoinVertical` for real composition, but budget widths and heights before render so panes and overlays do not overflow or collapse unpredictably
- keep borders intentional: use fewer full boxes, more padding, accent rails, muted dividers, and selected-row backgrounds
- make overlays genuinely overlay-like within terminal constraints: centered panel, bounded height, clear escape hint, optional dimmed underlying frame, and no duplicate footer noise
- maintain accessibility in high-contrast theme: selected rows, warnings, and dangerous actions must remain legible without relying on hue alone

### Gap Closure Plan
To reach a polished and professional TUI experience, close the gaps in this order:

1. Establish a visual system and shared components.
   Define product-level primitives for hero state card, compact count strip, action item, evidence trail, setup stepper, command palette, modal panel, detail viewport, and selected directory row. These should be Lip Gloss components that routes compose instead of bespoke line dumps.

2. Reframe the shell.
   Reduce always-visible chrome to product title/route, sync truth, active work, and the smallest viable action hint. Move diagnostics, path/history, raw focus/layout state, clipboard/copy counts, repeated broker-truth reminders, and exhaustive shortcuts into command discovery, inspector, or Status. Wide layout can keep sidebar/main/inspector, but medium and narrow layouts should feel like focused single-pane work with overlays for nav and inspector.

3. Make overlays professional.
   Replace appended overlay blocks with centered modal panels. Command palette should have a search row, grouped suggestions, high-contrast selected row, right-aligned command IDs/shortcuts, and dimmed background where terminal rendering allows. Session switcher and leader help should follow the same modal grammar.

4. Promote Action Center.
   Make Action Center the first place to look when anything needs attention. The shell should either start there when follow-up exists or make the Dashboard hero card route there. Rename primary bucket labels into product language: `Approvals`, `Operational Attention`, and `Blocked Work` rather than internal token names. Every item should include state, reason, impact, owner/action, target, and evidence source, with raw watch-family facts moved to inspector/detail. The main pane and inspector should complement each other rather than repeating the same triage card verbatim.

5. Calm Dashboard.
   Dashboard should show one executive state card, one count strip, workflow posture, current active work, and the next best action. Readiness fields, version/build data, watch-family summaries, protocol bundle detail, low-level audit facts, and raw evidence posture fields should move to Status, Action Center detail, or inspector.

6. Normalize state cards.
   All loading, empty, ready, waiting, blocked, degraded, failed, completed, resumed, and approval-required states should use a rich state-card spec with state label, message, reason, next action, route cue, shortcut cue, and optional evidence/source cue. The basic three-argument state-card helper should be retired from primary route states or wrapped so missing reason/next-action cannot regress.

7. Split rendered, structured, and raw modes more strictly.
   Rendered mode should be product language. Structured mode can show stable fields and counts. Raw mode can show protocol, schema, hashes, contract method names, binding kinds, policy codes, request/decision identities, and debug details. Runs, Approvals, Artifacts, Audit, Status, Model Providers, Git Setup, and Git Remote need this separation most.

8. Improve evidence trail navigation.
   Every workflow result should have an obvious path: Chat or Runs -> Artifacts -> Audit -> verification posture -> export/offline verification -> anchoring where available. Use short stable labels in primary views and keep full digests copyable through copy actions or raw mode.

9. Polish setup/review routes.
   Model Providers should read like credential setup with safe secret ingress, current readiness, next action, and evidence that no secret leaked. Git Setup should read like account and identity setup, not a broker state dump. Git Remote should read like a guarded review-and-execute flow with approval and lease prerequisites, not a hash inventory.

10. Verify with real product walkthroughs.
     Capture terminal frames and deterministic bundle coverage for Dashboard, Action Center, Chat, Runs, Approvals, Artifacts, Audit, Status/project substrate, Model Providers, Git Setup, Git Remote, command palette, session switcher, leader help, inspector sheet, sidebar drawer, quit confirmation, blocked/degraded/disconnected states, and desktop plus compact/mobile layouts. Review those frames against CHG-060 before marking the TUI acceptance criterion complete; desktop-only route snapshots are not enough.

11. Close interaction-latency gaps with measured verification.
    Optimize palette/session filtering, overlay rendering, leader-step handling, and focus/navigation hot paths using the smallest correct changes. Add targeted benchmarks for keypress update latency, search/filter latency, and overlay view cost, plus deterministic tests that confirm rapid scripted input is fully consumed without dropped keys.

TUI acceptance is intentionally dogfooding-gated rather than fully preplanned. Issues found during walkthroughs of the required paths should be captured, blockers and misleading product-truth issues should be fixed before closure, and non-blocking polish can be recorded as follow-up.

That polish should stay aligned with the reviewed performance-contract discipline in `CHG-053`, especially:

- attach, reconnect, and resume surfaces should continue to reflect broker-owned lifecycle truth rather than client-local optimistic shortcuts
- waiting, blocked, degraded, and failed states should remain operator-visible without reintroducing misleading high-activity rendering paths or synthetic progress cues
- dogfooding fixes should preserve the same authoritative surfaces that the MVP performance gates measure rather than optimizing around those gates with less truthful UI behavior
- polished wording must not imply a stronger attestation, execution, publication, or collaboration posture than the current broker-owned state can prove

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

The TUI polish close path should also include interaction-performance verification:

- benchmark palette typing, session-switcher typing, leader open/step, and overlay-heavy shell view paths against realistic deterministic datasets
- verify rapid scripted key input is fully consumed for overlay text-entry flows without lost characters
- keep the benchmark gate deterministic by fixing term size, data shape, and benchmark OS in CI rather than relying on PTY timing variance across platforms

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
- TUI and surrounding operator surfaces are polished enough that a new user can test the product coherently on Linux, as demonstrated by captured route, overlay, and responsive-layout audit frames
- the beta story is honest about assurance and execution behavior
- workflow-path and TUI polish remain compatible with the authoritative surfaces and honest measurement boundaries frozen by `CHG-053`
- git remote publication is either explicitly out of beta messaging or covered by a separate reviewed smoke path before any publishing claim is made
