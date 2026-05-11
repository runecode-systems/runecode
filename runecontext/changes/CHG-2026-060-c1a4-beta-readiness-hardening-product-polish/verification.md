# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `go test ./...`
- `cd runner && npm run lint`
- `cd runner && npm test`
- `cd runner && npm run boundary-check`
- `just test`
- `just ci`

## Implemented This Pass
- Fixed broker-side session execution tests around the real runner bridge by keeping production stdio subprocess launch intact while adding narrow in-process runner launch seams for deterministic tests.
- Verified the supported workflow slice now compiles and persists authoritative `RunPlan` state, launches/proxies the runner path, accepts runner checkpoint/result reports, and keeps run/session projections plan-authoritative.
- Added/updated smoke coverage that exercises `change_draft`, `spec_draft`, `draft_promote_apply`, and `approved_change_implementation`, then inspects runs, artifacts, approvals, audit records, evidence snapshot, inclusion, and offline bundle verification.
- External audit anchoring remains environment-conditional and is not claimed as a completed always-on smoke in this closure pass.
- TUI polish remains explicitly pending and is not marked complete here. The preserved full snapshot audit confirmed that the TUI has real route structure and product intent, but it also showed heavy chrome/footer pressure, rendered-mode debug leakage, route-specific product-language gaps, and missing overlay/responsive audit proof that must be closed before the TUI acceptance criterion can pass.

## TUI Polish Review Findings
- Current TUI implementation does use Bubble Tea, Bubbles, and Lip Gloss foundations: `tea.NewProgram`, `tea.Model`/`Init`/`Update`/`View`, async `tea.Cmd` route loading, `tea.Tick` for broker-watch polling/activity, Bubbles textarea/viewport/help/spinner primitives, and Lip Gloss theme/style/composition primitives.
- Bubble Tea architecture is broadly sound, but polish work must preserve deterministic `View` rendering, route-local `Update` state transitions, explicit keyboard ownership for text/secret entry and overlays, command-based broker operations, centralized resize behavior, and honest watch-driven progress.
- Lip Gloss usage is competent but not yet product-polished. Current styling provides panes, borders, badges, muted text, and theme tokens, but the professional bar requires stronger shared components, accent rails, focused cards, modal overlays, deliberate state colors, display-width-safe truncation, and high-contrast verification.
- Interaction performance is currently below the desired product bar. The source review found synchronous per-keystroke overlay filtering, full-match row formatting before visible-window trimming, repeated shell-background render work while overlays are active, repeated route-surface/layout generation in hot paths, unnecessary leader binding/choice recomputation, and some synchronous persistence writes from interactive paths.
- These latency findings match Bubble Tea's serialized `Update` plus render model: any synchronous filter, layout, render, or file-write cost directly increases key-to-render latency and can make rapid input feel dropped. TUI polish therefore includes interaction responsiveness, not only wording and visual hierarchy.
- The highest-value implementation fixes are to precompute normalized search text, reuse match buffers, render only visible overlay rows, cache/reuse unchanged shell background work while overlays are active, reduce duplicate route-surface work in focus/overlay flows, remove synchronous persistence from critical key paths, and add deterministic benchmarks/tests for keypress-heavy scenarios.
- The preserved `full-audit` desktop bundle covered 10 routes and 13 scenarios: Dashboard, Chat, Runs, Approvals, Action Center, Audit, Status, Model Providers, Git Setup, and Git Remote. That bundle is useful proof of route direction, but it is not enough to close TUI polish because it does not yet cover Artifacts, overlay surfaces, or compact/mobile behavior.
- The TUI partially satisfies CHG-060: Dashboard now has an executive-summary shape, Action Center is a credible triage route, Chat communicates active-work state honestly, Status/project substrate has understandable lifecycle concepts, and the product has real inspectors, copy actions, and plan-authoritative workflow surfaces.
- The TUI does not fully satisfy CHG-060 yet: shell chrome and footer rows still dominate too much vertical space, Action Center cards and inspectors are too dense, and Runs, Approvals, Audit, Status, Model Providers, Git Setup, and Git Remote still expose too much control-plane/debug detail in rendered primary surfaces.
- Dashboard is the closest route to the target, but it still needs calmer chrome and less duplicated state/badge language; Git Remote, Model Providers, Git Setup, Audit, and Runs remain the clearest rendered-mode cleanup priorities.
- The clean layout target should follow the provided inspiration: dark spacious base, elevated panels, subdued secondary text, strong selected rows, compact command palette, left accent rails for active blocks, careful color semantics, and low-noise bottom help.

## Required Product Smokes
- Project-substrate lifecycle smoke: inspect/posture, adopt compatible existing substrate, init preview/apply for missing substrate, upgrade preview/apply for supported older substrate, and validate/status after apply.
- Workflow smoke: run `change_draft` and `spec_draft` through the real trusted `RunPlan` and runner path.
- Promote/apply smoke: promote a reviewed change draft into `runecontext/changes/` and a reviewed spec draft into `runecontext/specs/` through the shared audited mutation path.
- Implementation smoke: run `approved_change_implementation` from one reviewed implementation input set and verify resulting local workspace mutation plus required RuneContext lifecycle metadata updates when included in the approved input.
- Approved implementation identity smoke: verify trusted code rejects an input set whose embedded `input_set_digest` does not equal the canonical semantic body digest recomputed with `input_set_digest` omitted, while still using the bound artifact digest for exact stored payload retrieval.
- Evidence smoke: inspect run/session/artifact/approval/audit surfaces, capture an evidence snapshot, verify at least one record-inclusion result, export an evidence bundle, and verify the bundle offline.
- External anchoring smoke: exercise external audit anchoring on the real workflow path where environment and policy allow.
- Git publication posture: confirm beta messaging does not claim push, pull request, prompt-to-PR, or team-collaboration publishing unless a separate reviewed git remote smoke path is added.

## Required TUI Polish Smokes
- Launch through the canonical product path, not only `go run ./cmd/runecode-tui`, so the TUI is reviewed with broker-owned lifecycle and sibling-binary resolution behavior.
- Capture terminal frames or deterministic render snapshots for Dashboard, Action Center, Chat, Runs, Approvals, Artifacts, Audit, Status/project substrate, Model Providers, Git Setup, Git Remote, command palette, session switcher, leader help, inspector sheet, sidebar drawer, quit confirmation, desktop, compact, and mobile layouts.
- Require the preserved deterministic audit bundle used for closure to cover Artifacts and overlay surfaces in addition to the current desktop route set; desktop-only route captures are insufficient proof of TUI polish.
- Verify rapid typing in command palette and session switcher remains responsive and deterministic under realistic fixture sizes, without dropped characters or visible event-loop stalls.
- Verify arrow-key focus/navigation and leader-key start/step flows remain responsive under normal route and overlay state, including when the active route has populated inspector/detail content.
- For every primary route, verify the first visible card answers: what is happening, whether it is healthy, whether the operator is blocked, what the operator should do next, and where the evidence or source of truth lives.
- Verify Dashboard is a calm executive overview and does not surface readiness fields, watch-family details, protocol bundle data, version/build detail, or low-level audit facts as primary content.
- Verify Action Center is the clearest operator home for attention, with product-language buckets for approvals, operational attention, and blocked work, and with raw watch-family/debug facts moved to inspector/detail.
- Verify Action Center selected cards remain scannable and the inspector adds drill-down value instead of restating the same reason, impact, owner/action, and target fields verbatim.
- Verify Chat only shows broker-owned execution stages when broker/session/run state supports them and does not invent optimistic progress.
- Verify Runs, Approvals, Artifacts, Audit, Status, Model Providers, Git Setup, and Git Remote separate rendered/product language from structured and raw modes.
- Verify Model Providers, Git Setup, and Git Remote no longer read like broker API dumps in their primary rendered views.
- Verify command palette and overlays render as professional centered/elevated modal surfaces with bounded height, selected-row styling, grouped content, escape hints, and no duplicate footer noise.
- Verify route footers and global chrome remain calm while exhaustive commands stay discoverable through command palette, leader help, inspectors, or generated help; repeated trust-boundary prose and verbose shortcut lists should not dominate every screen.
- Verify all state-card variants include state label, message, reason, next action, route cue, shortcut cue, and evidence/source cue where applicable.
- Verify display-width-sensitive rows remain aligned with styled text, wide characters, and ANSI sequences; raw rune-count truncation should not break selected rows, directories, overlays, or inspector summaries.
- Verify dark, dusk, and high-contrast themes preserve legibility for ready/success, waiting/attention, blocked/danger, active/command, link/evidence, selected, muted, and raw/digest text.

## Required TUI Interaction Performance Checks
- Add focused Go benchmarks for palette typing/filtering, session-switcher typing/filtering, leader open/step, and overlay-heavy shell view rendering with deterministic fixed datasets.
- Add a deterministic no-dropped-keys test that scripts rapid overlay text-entry input and verifies all characters are consumed in order.
- Keep benchmark runs deterministic by fixing dataset size, terminal dimensions, and benchmark environment; do not rely on PTY wall-clock timing variance across the full CI matrix.
- Run the TUI performance gate on a stable Linux CI job and fail on meaningful regressions rather than treating interaction responsiveness as manual-only QA.

## Verification Notes
- Confirm `runecontext/project/roadmap.md` places this change under `v0.1.0-alpha.11` and keeps `v0.1.0-beta.1` as the milestone framing.
- Confirm `CHG-2026-054-6c1e-runtime-attestation-post-handshake-gating-v0` is reflected consistently wherever beta assurance wording depends on its verified integration.
- Confirm the proposal treats this lane as integration and dogfooding hardening, not a replacement architecture.
- Confirm the design requires canonical project-substrate lifecycle proof through broker-owned product surfaces.
- Confirm the design requires `change_draft`, `spec_draft`, `draft_promote_apply`, and `approved_change_implementation` through the real trusted and untrusted execution path.
- Confirm the design requires production adoption of authoritative trusted `RunPlan` compilation and persistence.
- Confirm the design calls out runner transport and reporting integration rather than leaving noop/default runner transport ambiguous for the real path.
- Confirm run/session/TUI projections are plan-authoritative for the supported path rather than artifact-inferred.
- Confirm approved implementation can carry required RuneContext lifecycle metadata updates without creating a separate fifth workflow operation for this lane.
- Confirm approved implementation keeps `input_set_artifact_digest` and semantic `input_set_digest` distinct and fail-closed on embedded semantic digest drift.
- Confirm the tasks explicitly capture TUI and operator polish discovered while testing.
- Confirm the TUI polish task list includes the current once-over findings, professional target experience, route-by-route gap closure, Bubble Tea/Bubbles/Lip Gloss best-practice expectations, and visual dogfooding evidence requirements.
- Confirm the change requires exercising evidence snapshot, record inclusion, bundle export, and offline verification on the real workflow path.
- Confirm the design keeps product messaging and assurance wording aligned with actual implementation state.
- Confirm git remote publication remains out of required beta messaging unless a separate reviewed publishing smoke path is added.

## Close Gate
Use the repository's standard verification flow before closing this change, with `just ci` as the parity gate after targeted workflow and product smokes pass. The TUI acceptance criterion must remain open until the required TUI polish smokes pass and captured frames show a professional, calm, product-language TUI rather than a dense debug console.
