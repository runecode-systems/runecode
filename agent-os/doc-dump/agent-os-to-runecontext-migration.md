# Agent OS to RuneContext Migration Plan

## Purpose

This document defines how the RuneCode repository will migrate its repo-local planning, standards, and product-governance content from `agent-os/` to RuneContext-managed content under `runecontext/`.

This migration is only about how this repository manages its own docs, standards, roadmap, and planned work.

This document is a temporary operator migration guide. It is intended to help perform the cutover step by step, then be discarded once the migration is complete. It is not itself intended to become canonical RuneContext project content.

This migration does include rewriting the current and future planning artifacts so they describe the final RuneContext-oriented future state for RuneCode, rather than preserving Agent OS assumptions inside newly migrated canonical files.

It does not cover:

- integrating RuneContext into the RuneCode runtime or trust-boundary architecture
- changing `cmd/`, `internal/`, `runner/`, or `protocol/` to consume RuneContext artifacts at runtime
- redefining RuneContext core semantics for other repositories

## Decisions

- RuneContext becomes the canonical repo-local planning and standards system for this repository.
- All canonical RuneContext project content for this repository lives under `./runecontext/`, with root configuration in `./runecontext.yaml`.
- This repository starts in RuneContext `verified` assurance mode from the beginning of the migration.
- For this repository, assurance artifacts live under `./runecontext/assurance/`.
- This is a phased cutover, but not a long-lived dual-track migration.
- This migration is foundation-first and direct-to-final-state; do not first import Agent OS assumptions unchanged and schedule a second semantic rewrite later.
- When a planned feature folder is migrated, rewrite Agent OS-specific planning and integration assumptions during that migration so the new artifact reflects the intended RuneContext future state on day one.
- Top-level decisions that affect later feature rewrites must be captured early as RuneContext decisions and an umbrella migration change, rather than remaining only in temporary notes.
- For future RuneCode planning captured in this repository, assume RuneCode uses a bundled-by-default RuneContext companion, with external RuneContext support as a possible later advanced option.
- For future RuneCode planning captured in this repository, assume RuneCode-managed repos must use RuneContext `verified` mode.
- Migrated planning artifacts should assume generic machine-friendly RuneContext capabilities and metadata, with RuneCode-specific orchestration and UX living in RuneCode rather than as RuneContext-only semantics.
- For future RuneCode planning captured in this repository, assume RuneCode owns the user-facing command set and UX while invoking bundled RuneContext capabilities under the hood; end users should not need to use `runectx` directly in normal RuneCode workflows.
- For future RuneCode planning captured in this repository, assume RuneContext may later expose generic advisory consumer-compatibility warnings during upgrade flows, but hard compatibility enforcement for RuneCode-managed repos remains in RuneCode.
- If a RuneCode feature depends on project context, standards, changes, specs, bundles, context packs, or assurance, the migrated plan should assume that feature ships with its RuneContext integration complete for that feature surface rather than shipping a temporary non-RuneContext path first.
- `agent-os/` content is deleted only after equivalent RuneContext artifacts exist, validate, have their references rewritten, and have their migration captured in assurance history.
- `agent-os/doc-dump/project-idea.md` remains frozen historical material during the migration and is not edited as part of normal cutover work.

## Current Source Inventory

Current `agent-os/` content to migrate:

- Product docs: 3 files under `agent-os/product/`
- Standards docs: 39 Markdown standards plus `agent-os/standards/index.yml`
- Planned/completed spec folders: 34 folders under `agent-os/specs/`
- Historical archive material: `agent-os/doc-dump/`

## Current RuneContext Contract Snapshot

This migration is currently operating against the installed `runectx metadata` contract and the repo's current RuneContext project state.

Current metadata-backed snapshot:

- `runectx` release: `0.1.0-alpha.14`
- metadata schema version: `2`
- default project version: `0.1.0-alpha.14`
- directly supported project versions: `0.1.0-alpha.5`, `0.1.0-alpha.6`, `0.1.0-alpha.7`, `0.1.0-alpha.8`, `0.1.0-alpha.14`
- upgradeable-from project versions: `0.1.0-alpha.12`, `0.1.0-alpha.13`
- project profile: `portable_project_root`
- root config path: `runecontext.yaml`
- content root: `runecontext/`
- assurance path: `runecontext/assurance/`
- manifest path: `runecontext/manifest.yaml`
- indexes root: `runecontext/indexes/`
- assurance tiers: `plain`, `verified`
- verified artifact support: baseline artifacts plus receipt families `context-packs`, `changes`, `promotions`, `verifications`
- canonicalization/hash contract:
  - context packs: `runecontext-canonical-json-v1`, `sha256`
  - assurance artifacts: `runecontext-canonical-json-v1`, `sha256`
- relevant migration helper command surfaces currently available:
  - `runectx standard create`
  - `runectx standard update`
  - `runectx standard list`
  - `runectx change assess-intake`
  - `runectx change assess-decomposition`
  - `runectx change decomposition-plan`
  - `runectx change decomposition-apply`

Current repo starting point:

- `runecontext.yaml` already exists
- `runecontext_version` is already `0.1.0-alpha.14`
- `assurance_tier` is already `verified`
- embedded source is already configured
- `runecontext/assurance/baseline.yaml` already exists
- `runectx validate --json` currently succeeds
- `runectx status --json` currently succeeds
- `runecontext/changes/` and `runecontext/bundles/` are currently empty

Practical implication:

- for this repository, the foundational RuneContext bootstrap is already established
- operators should normally begin this migration from Phase 1, not by redoing the entire bootstrap from scratch
- if the repo later falls out of a directly supported project version or stops validating cleanly, stop content migration work and repair the RuneContext foundation first

## Target State

The target RuneContext tree for this repository is:

```text
runecontext.yaml
runecontext/
  project/
    mission.md
    roadmap.md
    tech-stack.md
    standards-inventory.md        # optional human inventory
  standards/
    ...                           # canonical reusable standards
  bundles/
    ...                           # reusable context selectors for this repo
  changes/
    CHG-.../                      # planned and in-flight work items
  specs/
    ...                           # durable current-state specs
  decisions/
    ...                           # durable architecture/process decisions
  operations/
    ...                           # repo-local operations/reference docs when needed
  manifest.yaml                   # generated, non-authoritative discovery artifact
  indexes/
    ...                           # generated, non-authoritative discovery artifacts
  assurance/
    baseline.yaml
    receipts/
      context-packs/
      changes/
      promotions/
      verifications/
```

Generated artifacts such as `runecontext/manifest.yaml` and `runecontext/indexes/` are useful for discovery, browsing, and validation hints, but they are not the canonical source of truth.

## Canonical Mapping

| Current source | RuneContext target | Notes |
| --- | --- | --- |
| `agent-os/product/mission.md` | `runecontext/project/mission.md` | Direct content migration |
| `agent-os/product/roadmap.md` | `runecontext/project/roadmap.md` | Remains a product-facing roadmap, but no longer owns lifecycle state |
| `agent-os/product/tech-stack.md` | `runecontext/project/tech-stack.md` | Direct content migration |
| `agent-os/standards/**/*.md` | `runecontext/standards/**/*.md` | Canonical standards with RuneContext-required metadata |
| `agent-os/standards/index.yml` | bundles plus optional inventory doc | Not a first-class RuneContext core artifact |
| completed `agent-os/specs/*` | `runecontext/specs/*.md` | Durable current-state specs |
| planned `agent-os/specs/*` | `runecontext/changes/CHG-.../` | Stable-path work items |
| meta planning folders in `agent-os/specs/*` | `runecontext/decisions/*.md` or legacy archive | Use a decision if the content is still useful; otherwise archive |
| `agent-os/doc-dump/project-idea.md` | frozen legacy/archive material | Keep read-only and non-canonical |

## Core Migration Rules

### 0. Capture Cross-Cutting Decisions Before Bulk Feature Migration

Before migrating large numbers of feature folders, establish the top-level RuneContext decision record for this repository.

Capture early:

- the decision that RuneContext replaces `agent-os/` as the canonical repo-local planning and standards system
- the decision that this repository uses `verified` assurance from adoption onward
- the decision that this repository keeps assurance under `runecontext/assurance/`
- the decision that future RuneCode planning in this repository assumes bundled-by-default RuneContext integration
- the decision that future RuneCode planning in this repository assumes RuneCode-managed repos require `verified` mode
- the decision that future RuneCode planning in this repository assumes RuneCode owns the user-facing command set while invoking RuneContext under the hood
- the decision that future RuneCode planning in this repository assumes generic advisory consumer-compatibility warnings may exist in RuneContext, while hard compatibility enforcement remains in RuneCode
- the umbrella migration change that tracks the repo-wide cutover

These decisions should exist before most feature-folder migration begins, so later migrated changes/specs/decisions can reference them as canonical context instead of re-deriving them ad hoc.

### 1. Verified Assurance Starts Immediately

There is no temporary plain-mode phase for this repository.

The migration begins only after:

- `runecontext.yaml` exists
- `assurance_tier` is set to `verified`
- `runectx assurance enable` has been run for this repository

From that point forward:

- new RuneContext-authored artifacts are captured as native verified history
- pre-adoption `agent-os/` material may be backfilled as imported history
- old content is not rewritten to pretend it was always native RuneContext evidence

### 2. Product Docs Move First

`agent-os/product/` becomes `runecontext/project/` before planned work is converted into changes.

This ensures that:

- the new roadmap points at RuneContext artifacts instead of `agent-os/specs/`
- product context is canonical before change creation begins
- later change `proposal.md` and `standards.md` files can refer to canonical RuneContext paths

### 3. Standards Become Canonical Files, Not Discovery Output

The current `agent-os/standards/**/*.md` files should be imported directly into `runecontext/standards/**/*.md` as authored canonical standards.

Do not use `runectx standard discover` as the primary import mechanism.

Reason:

- `standard discover` is advisory-only
- the current standards are already canonical repo policy content
- the initial migration should preserve current standard bodies and convert them into RuneContext-native standard files with the required metadata

`runectx promote` remains the correct path for future new standards promoted out of changes, but it is not the best fit for the initial bulk import of already-existing standards.

### 4. Planned Spec Folders Do Not All Become Changes Blindly

`agent-os/specs/` contains more than one kind of artifact.

Before conversion, each folder must be classified as one of:

- durable current-state `spec`
- durable `decision`
- active or planned `change`
- legacy or migration-only archive material

### 4a. Do Not Preserve Agent OS Semantics Inside Newly Migrated Canonical Files

When migrating a planned feature folder into RuneContext:

- do not preserve references that treat Agent OS as the future planning/control system for this repository
- do not preserve old instructions that assume later retrofitting of RuneContext after feature delivery
- do rewrite future-facing content so it describes the intended RuneContext-backed final state for that feature
- do preserve historical or superseded Agent OS context only when useful for traceability, references, or imported-history assurance

In short: preserve history where needed, but migrate canonical meaning directly to the final RuneContext-aware form.

### 4b. Migrate Once, Not Twice

Do not use a two-step pattern of:

1. migrating an `agent-os/specs/*` folder into RuneContext with old Agent OS assumptions still intact, then
2. opening a second change to swap those assumptions to RuneContext later

Instead:

- establish the cross-cutting decisions first
- migrate each feature directly into its intended RuneContext-era meaning
- capture any remaining legacy assumptions only as historical notes or references

### 5. Delete Legacy Files Only After Verified Replacement

An `agent-os/` file or folder is deleted only when all of the following are true:

- the RuneContext replacement artifact exists
- `runectx validate` succeeds for the new state
- inbound references have been updated
- the migration is captured in verified assurance history

### 6. Use Metadata-Driven Preflight Checks During Migration

Before any substantial migration batch:

- run `runectx metadata`
- confirm the repo remains on a directly supported project version for the currently installed RuneContext release
- confirm the project profile still matches `portable_project_root`
- confirm `runecontext/assurance/` remains the active assurance path
- confirm the expected verified artifact families are still supported
- run `runectx validate --json`
- run `runectx status --json`

Stop and repair or upgrade before continuing if:

- the repo is no longer on a directly supported project version
- the project has moved to an upgrade-only or unsupported state
- the project profile no longer matches the migration assumptions
- validation or status no longer succeeds cleanly

## How Each Content Family Migrates

### Product Docs

Directly migrate:

- `agent-os/product/mission.md` -> `runecontext/project/mission.md`
- `agent-os/product/roadmap.md` -> `runecontext/project/roadmap.md`
- `agent-os/product/tech-stack.md` -> `runecontext/project/tech-stack.md`

Product-doc migration rules:

- preserve the user-facing intent and content structure where possible
- replace `agent-os/specs/...` roadmap links with RuneContext change IDs/paths for planned work and RuneContext spec paths for completed work
- stop using the roadmap as the lifecycle source of truth for work status
- rewrite future-facing product language so it assumes RuneContext is the canonical planning/standards system for this repository
- where product docs discuss future RuneCode behavior around project context/planning, update them to describe RuneContext-based behavior rather than Agent OS-based behavior

After cutover:

- `runecontext/project/roadmap.md` is a human-facing product summary
- lifecycle state for in-flight work lives in `runecontext/changes/*/status.yaml`
- durable current-state outcomes live in `runecontext/specs/*.md`

### Standards

Directly migrate all 39 current standards into `runecontext/standards/`, preserving the current topic structure where useful:

- `backend/`
- `ci/`
- `global/`
- `javascript/`
- `product/`
- `security/`
- `testing/`

Standards migration rules:

- preserve current body text initially
- add RuneContext-required metadata/frontmatter
- mark current active repo policy standards as active unless there is a reason to import them as draft or deprecated
- rewrite standard references to use canonical `runecontext/standards/...` paths
- use migration aliases or replacement metadata where helpful for preserving traceability from old paths
- if a standard references Agent OS as the canonical governance/planning substrate for this repository, rewrite it to reference RuneContext instead during migration
- preserve old Agent OS wording only when it is intentionally historical or explanatory, not as current repo policy

`agent-os/standards/index.yml` does not map cleanly to a RuneContext core artifact. Replace its function with:

- canonical standard paths under `runecontext/standards/**`
- reusable bundles in `runecontext/bundles/*.yaml`
- an optional human inventory doc such as `runecontext/project/standards-inventory.md`

Practical mutation guidance:

- keep the strategic rule that the initial migration should not depend on `runectx standard discover`
- use `runectx standard create` and `runectx standard update` as convenient mutation surfaces where they help preserve metadata correctness and repeatability
- use `runectx standard list` as a post-migration verification aid after standards batches land

### Completed Specs

The following currently completed `agent-os/specs/` folders should be migrated as durable RuneContext specs:

- `agent-os/specs/2026-03-08-1128-dev-env-ci-nix-flakes/`
- `agent-os/specs/2026-03-08-1039-scaffold-ci-matrix/`
- `agent-os/specs/2026-03-13-1415-source-quality-guardrails-v0/`
- `agent-os/specs/2026-03-08-1039-protocol-schemas-v0/`

Completed-spec migration rules:

- do not keep the old folder-per-spec shape as the canonical RuneContext state
- create one durable `runecontext/specs/*.md` file per migrated completed spec
- distill durable current-state content out of `plan.md`, `shape.md`, `references.md`, and `standards.md`
- keep one-time planning detail only if it is still relevant to current-state understanding
- move one-time architecture/process choices into `runecontext/decisions/*.md` when that shape fits better than a current-state spec
- if the completed spec includes future-facing Agent OS assumptions that are no longer intended, rewrite the durable spec to reflect the actual intended current-state or accepted future-state model
- do not preserve superseded Agent OS planning language as if it were still normative current-state behavior

### Meta Planning Folder

The following folder is migration/meta-planning material, not an ongoing feature spec:

- `agent-os/specs/2026-03-08-1039-initial-spec-suite-mvp/`

Target:

- migrate the enduring rationale into a RuneContext decision if the content is still useful
- otherwise retain it only as frozen historical archive material until the final legacy cleanup step

### Planned Spec Folders

The remaining planned feature folders should be migrated into RuneContext changes.

#### Default status mapping

- Roadmap items in explicit release buckets should default to `planned`
- `vNext` items may start as `proposed` if the team wants to preserve looser commitment, or `planned` if the existing shaping is strong enough to justify it

Practical triage guidance:

- use `runectx change assess-intake` when deciding whether an old planned folder should stay a single change or needs further decomposition
- use `runectx change assess-decomposition`, `runectx change decomposition-plan`, and `runectx change decomposition-apply` when an old Agent OS folder is best represented as an umbrella change with sub-changes in RuneContext

#### Source-file mapping for each planned spec folder

For each planned `agent-os/specs/<folder>/` that becomes a change:

- `plan.md` -> `proposal.md` after rewriting into RuneContext's required proposal structure
- `shape.md` -> `design.md`
- `standards.md` -> `standards.md` with rewritten canonical standard refs
- `references.md` -> `references.md`
- acceptance criteria and task breakdown -> `tasks.md` and/or `verification.md` as needed

Additional rewrite rules for planned spec folders:

- remove future-state language that assumes Agent OS remains the canonical planning/spec/standards substrate for this repository
- rewrite future-state language so it assumes RuneContext is the canonical planning/spec/standards substrate
- if the feature includes project-context, standards, spec, change, bundle, context-pack, or assurance behavior, update the migrated content to assume deeper RuneContext integration from the point that feature exists
- avoid vague placeholders like "swap in RuneContext later" or "future RuneContext integration" when the intended end state is already known from the migration decisions
- preserve historical design reasoning from the old folder only when it still adds decision context; otherwise omit it from canonical migrated content

#### Planned change batches

##### Batch 1: `v0.1.0-alpha.2`

- `agent-os/specs/2026-03-08-1039-artifact-store-data-classes-v0/`
- `agent-os/specs/2026-03-08-1039-crypto-key-mgmt-v0/`
- `agent-os/specs/2026-03-08-1039-audit-log-verify-v0/`
- `agent-os/specs/2026-03-08-1039-audit-anchoring/`

##### Batch 2: `v0.1.0-alpha.3`

- `agent-os/specs/2026-03-08-1039-broker-local-api-v0/`
- `agent-os/specs/2026-03-08-1039-policy-engine-v0/`
- `agent-os/specs/2026-03-08-1039-launcher-microvm-backend-v0/`
- `agent-os/specs/2026-03-08-1039-container-backend-opt-in-v0/`

##### Batch 3: `v0.1.0-alpha.4`

- `agent-os/specs/2026-03-08-1039-secretsd-model-gateway-v0/`
- `agent-os/specs/2026-03-08-1039-workflow-workspace-roles-gates-v0/`
- `agent-os/specs/2026-03-08-1039-minimal-tui-v0/`

##### Batch 4: `v0.1.0-beta.1`

- `agent-os/specs/2026-03-08-1039-formal-spec-tla-v0/`
- `agent-os/specs/2026-03-08-1039-zk-proof-v0/`

##### Batch 5: `v0.2 (Post-MVP)`

- `agent-os/specs/2026-03-08-1039-git-gateway/`
- `agent-os/specs/2026-03-10-1530-approval-profiles-v0/`
- `agent-os/specs/2026-03-13-1600-workflow-extensibility-v0/`
- `agent-os/specs/2026-03-12-1030-auth-gateway-role-v0/`
- `agent-os/specs/2026-03-13-1601-bridge-runtime-protocol-v0/`
- `agent-os/specs/2026-03-11-1920-openai-chatgpt-subscription-provider-v0/`
- `agent-os/specs/2026-03-11-1921-github-copilot-subscription-provider-v0/`
- `agent-os/specs/2026-03-13-1602-local-ipc-protobuf-transport-v0/`
- `agent-os/specs/2026-03-08-1039-web-research-role/`
- `agent-os/specs/2026-03-08-1039-deps-fetch-cache/`
- `agent-os/specs/2026-03-13-1603-external-audit-anchoring-v0/`
- `agent-os/specs/2026-03-08-1039-image-toolchain-signing/`
- `agent-os/specs/2026-03-08-1039-windows-microvm-runtime/`
- `agent-os/specs/2026-03-08-1039-macos-virtualization-polish/`

##### Batch 6: `vNext`

- `agent-os/specs/2026-03-13-1730-workflow-concurrency-v0/`
- `agent-os/specs/2026-03-13-1731-isolate-attestation-v0/`

## RuneCode Planning Rewrite Rules During Migration

This repository migration is not only a folder/layout migration. It is also the moment to replace the old Agent OS-centered future planning model with the intended RuneContext-centered future planning model for RuneCode.

Apply the following rewrite rules whenever a migrated artifact talks about the future behavior of RuneCode.

### 1. RuneContext Is The Canonical Planning And Project-Context System

When a migrated artifact discusses:

- planned changes
- current-state specs
- standards
- project mission/roadmap/tech stack context
- durable decisions
- reusable context selection

rewrite it so the canonical system is RuneContext, not `agent-os/`.

### 2. RuneCode Uses RuneContext As A Bundled Companion By Default

When a migrated artifact describes how RuneCode will consume project context, rewrite it to assume:

- RuneCode ships with a compatible bundled RuneContext companion by default
- RuneCode releases declare the supported RuneContext version range
- RuneCode does not silently track arbitrary latest RuneContext releases
- external RuneContext support may be added later as an advanced option, but it is not the default integration model
- RuneCode users should not need to use raw `runectx` commands during normal RuneCode workflows; RuneCode owns the user-facing command set and invokes RuneContext under the hood

### 3. RuneCode-Managed Repositories Require Verified Mode

When a migrated artifact describes RuneCode interacting with repo context, planning, or assurance, rewrite it to assume:

- the repo is using RuneContext `verified` mode
- RuneCode blocks incompatible, stale, or non-verified project states rather than silently downgrading
- bootstrap, doctor, and upgrade flows may be exceptions, but normal RuneCode-managed operation assumes verified mode

### 4. RuneContext Stays Generic; RuneCode Owns The Deeper Product Integration

When a migrated artifact discusses product integration, rewrite it so:

- RuneContext owns portable core artifacts and generic machine-friendly operations
- RuneCode owns orchestration, compatibility gating, typed broker delivery, audit/provenance binding, isolate delivery, UI/UX, and other RuneCode-specific behavior
- RuneContext does not become a RuneCode-only semantic layer

### 5. Context-Aware RuneCode Features Ship With Their RuneContext Integration

When a migrated artifact plans a RuneCode feature that depends on project context or project knowledge, rewrite it to assume that feature ships with its RuneContext integration complete for that feature surface.

This applies especially to features involving:

- changes, specs, standards, bundles, decisions, project docs
- deterministic context packs
- verified assurance and audit evidence
- workflow gating, approvals, or review context
- typed context delivery to trusted components or isolates

Do not preserve planning language that assumes a temporary non-RuneContext implementation will ship first and be retrofitted later.

### 6. Prefer Generic Machine-Friendly RuneContext Enhancements Over RuneCode-Only RuneContext Commands

If a migrated feature suggests RuneContext needs additional information for RuneCode, the preferred planning assumption is:

- add generic machine-readable metadata, capabilities, descriptors, or structured output to RuneContext when appropriate
- keep RuneCode-specific orchestration in RuneCode
- avoid adding RuneCode-only command semantics to RuneContext unless the need truly cannot be expressed generically

### 7. Prefer Generic Advisory Consumer-Compatibility Warnings In RuneContext, With Hard Enforcement In RuneCode

When a migrated artifact discusses project upgrades or compatibility safety, rewrite it to assume:

- RuneContext may expose generic advisory consumer-compatibility warning support during upgrade flows
- those warnings are advisory only and remain generic rather than RuneCode-specific
- RuneCode remains the hard enforcement point for whether a RuneCode-managed repo is usable with the current bundled RuneContext companion
- if a repo is upgraded out-of-band to a newer unsupported RuneContext project version, RuneCode fails closed and allows only safe remediation/diagnostic flows

## Recommended Migration Sequence

### Phase 0: Prepare the RuneContext Repository Root

1. Confirm the installed `runectx metadata` contract is the expected one for this migration.
2. Confirm `runecontext.yaml` exists for this repository.
3. Confirm the source mode is embedded.
4. Confirm the assurance tier is `verified`.
5. Confirm the base `runecontext/` directory tree and `runecontext/assurance/baseline.yaml` exist.
6. Run `runectx validate`.
7. Run `runectx status`.
8. If any of the above are missing, establish them before continuing.

This phase establishes the verified adoption baseline before any major content migration begins.

For this repository today, Phase 0 is already substantially complete. Treat it as a preflight confirmation step rather than a fresh bootstrap unless the project falls out of a valid directly supported state.

### Phase 1: Capture The Cross-Cutting Decisions And Create The Umbrella Migration Change

Before broad content migration, capture the major decisions made for this cutover.

At minimum, create durable decision records covering:

- RuneContext replaces `agent-os/` as the canonical planning/standards system for this repository
- this repository adopts RuneContext in `verified` mode from the start of migration
- this repository keeps assurance under `runecontext/assurance/`
- future RuneCode planning in this repository assumes bundled-by-default RuneContext integration
- future RuneCode planning in this repository assumes RuneCode-managed repos require `verified` mode
- future RuneCode planning in this repository assumes RuneContext stays generic and machine-friendly while RuneCode owns the deeper product-specific integration layer
- future RuneCode planning in this repository assumes RuneCode owns the normal end-user command surface and wraps RuneContext under the hood
- future RuneCode planning in this repository assumes generic advisory consumer-compatibility warnings may exist in RuneContext upgrade flows, while hard compatibility enforcement remains in RuneCode

Create one RuneContext change for the migration itself.

Purpose of the umbrella change:

- track the repo-wide migration work
- hold migration-specific tasks and verification notes
- provide one reviewable change record for the cutover effort

This umbrella change does not replace the imported feature changes. It tracks the migration process itself.

Why this step comes first:

- later migrated changes/specs/standards can point at canonical decisions instead of temporary notes
- feature-folder rewrites can go directly to the intended final RuneContext-era meaning
- the migration avoids a second repo-wide semantic rewrite later

### Phase 2: Migrate `agent-os/product/`

1. Move the three product docs into `runecontext/project/`.
2. Rewrite roadmap references to target RuneContext artifacts.
3. Validate.
4. Capture assurance history.
5. Delete the migrated `agent-os/product/` files once references are updated.

### Phase 3: Migrate `agent-os/standards/`

1. Create canonical standard files under `runecontext/standards/`.
2. Preserve current content bodies initially.
3. Add required RuneContext metadata.
4. Replace old standard refs with canonical RuneContext standard paths.
5. Create initial bundles to replace the old index-centric grouping model.
6. Validate.
7. Capture assurance history.
8. Delete migrated `agent-os/standards/**/*.md` files after reference rewrites are complete.
9. Replace `agent-os/standards/index.yml` with bundles and optional inventory docs, then remove it.

### Phase 4: Import Durable Specs and Decisions

1. Create durable spec files under `runecontext/specs/` for the four completed items.
2. Create a decision or archive plan for `2026-03-08-1039-initial-spec-suite-mvp`.
3. Update inbound references from docs, roadmap, and instructions.
4. Validate.
5. Capture assurance history.
6. Delete the migrated source folders.

### Phase 5: Convert Planned Spec Folders into Changes

Convert the planned feature folders into RuneContext changes in batches.

Recommended workflow for each batch:

1. Create the change with `runectx change new`.
2. Shape it as needed with `runectx change shape`.
3. Rewrite future-facing Agent OS assumptions to the intended final RuneContext-era meaning for that feature.
4. Migrate source content into `proposal.md`, `design.md`, `standards.md`, `references.md`, and optional `tasks.md` / `verification.md`.
5. Ensure the migrated content assumes the cross-cutting decisions already captured in RuneContext decisions.
6. Set lifecycle status appropriately.
7. Update roadmap references to point at the new change path or ID.
8. Validate.
9. Capture assurance history.
10. Delete the old `agent-os/specs/<folder>/` source folder.

Where helpful during change migration:

- use `runectx change assess-intake` before creating a migrated change if the old folder's scope is ambiguous
- use the decomposition commands when a single old folder should become an umbrella change with multiple RuneContext sub-changes

Expanded guidance for step 3:

- if the old folder talks about Agent OS as the future planning/spec/standards substrate, rewrite that to RuneContext
- if the old folder assumes RuneCode will first ship without RuneContext integration for a context-aware feature, rewrite that to the new direct-integration assumption
- if the old folder implies a separate RuneCode-only source-of-truth for project knowledge, rewrite it so RuneContext remains the canonical project-content layer
- if the old folder needs extra RuneContext machine-readable support for RuneCode, prefer planning a generic RuneContext capability or metadata addition rather than a RuneCode-only RuneContext semantic fork

Do not use this pattern:

1. migrate feature folder into RuneContext with old Agent OS assumptions still present
2. create a second follow-up change later to replace those assumptions with RuneContext-aware semantics

Instead, migrate directly to the final intended meaning once.

Do not create all planned changes before the product docs and standards have been migrated. The canonical standards pathing must exist first.

After major migration batches:

- generate indexes/manifest if desired for browsing and discovery
- re-run validation and status checks
- treat generated artifacts as helpful outputs, not as canonical authored state

### Phase 6: Rewrite Non-`agent-os/` References

The following files stay where they are, but their content must stop pointing at `agent-os/` as the canonical planning system:

- `README.md`
- `AGENTS.md`
- `docs/trust-boundaries.md`
- `docs/source-quality.md`
- `.github/copilot-instructions.md`
- `.github/instructions/*.md`

Update rules:

- product and planning references should point to `runecontext/project/`, `runecontext/changes/`, `runecontext/specs/`, `runecontext/decisions/`, and `runecontext/standards/`
- docs that currently cite planned spec folders should temporarily cite RuneContext change paths until durable specs exist
- tool-specific instruction files remain in `.github/` or other tool-owned locations, but their canonical content references move to RuneContext artifacts

### Phase 7: Final Legacy Cleanup

At the end of the cutover:

- remove migrated `agent-os/product/`
- remove migrated `agent-os/standards/`
- remove migrated `agent-os/specs/`
- keep `agent-os/doc-dump/project-idea.md` frozen until there is an explicit decision to archive or relocate it
- remove or archive the remaining `agent-os/` directory structure once nothing canonical depends on it

#### Standard Alias Cleanup

The migrated RuneContext standards may temporarily retain `aliases` entries that preserve traceability from legacy `agent-os/standards/...` IDs.

Those aliases are useful during the active migration window, but they are not required forever.

Remove migrated standard aliases only when all of the following are true:

- all canonical references have been rewritten to `runecontext/standards/...`
- no repo workflow, instruction file, generated artifact, or migration helper still depends on the old `agent-os/standards/...` IDs
- operators no longer need old-ID traceability for the active migration review window
- the standards tree validates cleanly without relying on the alias metadata

Recommended timing:

- do not remove aliases during the initial standards import
- do remove them near the very end of the migration, after spec/change/reference migration is complete and legacy canonical usage has been removed

Practical mutation guidance:

- use `runectx standard update --path standards/<path>.md --replace-aliases` to clear aliases for a migrated standard once it is safe to do so
- prefer one reviewed cleanup batch for alias removal rather than ad hoc piecemeal removal during earlier phases
- after alias removal, run `runectx standard list`, `runectx validate --json`, and `runectx status --json`

Recommended alias-removal verification:

- search the repository for `agent-os/standards/` and confirm any remaining hits are intentionally historical archive or migration-guide material only
- confirm no active RuneContext standard frontmatter still carries migration aliases unless intentionally retained for a documented reason

## Initial Bundle Set To Create

Create an initial set of repo bundles to replace the old standards index-driven grouping model and to support tooling/context selection.

Recommended initial bundles:

- `project-core`
  - includes `runecontext/project/mission.md`, `runecontext/project/roadmap.md`, `runecontext/project/tech-stack.md`, and core repo standards
- `go-control-plane`
  - aligns with trusted Go work in `cmd/`, `internal/`, and `tools/`
- `runner-boundary`
  - aligns with `runner/` trust-boundary and boundary-check work
- `protocol-foundation`
  - aligns with `protocol/`, schema validation, and shared fixtures
- `ci-tooling`
  - aligns with `justfile`, CI workflow, and release/tooling standards
- `product-planning`
  - includes project docs, planning decisions, and the active roadmap view

These bundles are repo-local context selectors. They do not replace change status or durable specs.

## Step-By-Step Operator Checklist

Use this as the practical execution checklist while performing the migration.

### Step 1: Prepare RuneContext In Verified Mode

- run `runectx metadata` and confirm the expected contract snapshot
- confirm the repo is on a directly supported project version
- confirm `runecontext.yaml` already exists and uses embedded source mode
- confirm `assurance_tier: verified`
- confirm `runecontext/assurance/baseline.yaml` exists
- run validation and status checks
- only if any of those are missing or invalid, repair/bootstrap the RuneContext foundation first

Stop here if verified mode is not working correctly. Do not migrate content into a half-initialized plain-mode or ambiguous setup.

### Step 2: Capture The Cross-Cutting Decisions

- create the top-level decisions listed in Phase 1
- create the umbrella migration change
- ensure these artifacts describe the intended final RuneContext-era meaning for the repository and for future RuneCode planning

### Step 3: Migrate Product Docs

- move `mission.md`, `roadmap.md`, and `tech-stack.md` into `runecontext/project/`
- rewrite roadmap references away from `agent-os/specs/...`
- rewrite future-facing references away from Agent OS as the canonical planning system
- validate and capture assurance

### Step 4: Migrate Standards

- import current standards directly into `runecontext/standards/`
- add required metadata/frontmatter
- rewrite references to canonical RuneContext standard paths
- replace `agent-os/standards/index.yml` responsibilities with bundles and optional inventory docs
- use `runectx standard create` / `runectx standard update` where helpful to keep metadata mutation disciplined
- use `runectx standard list` as a verification check after standards migration batches
- validate and capture assurance

### Step 5: Classify Every Existing Spec Folder Before Migrating It

For each `agent-os/specs/*` folder, explicitly classify it as:

- durable spec
- durable decision
- active/planned change
- archive-only material

Do not migrate unclassified folders opportunistically.

### Step 6: Migrate Completed Items To Durable Specs Or Decisions

- create `runecontext/specs/*.md` for completed durable current-state items
- create `runecontext/decisions/*.md` where the source folder was really an enduring decision rather than a current-state spec
- remove superseded Agent OS future-state assumptions during the rewrite
- validate and capture assurance

### Step 7: Migrate Planned Feature Folders Directly To Final RuneContext-Aware Changes

For each planned feature folder:

- use `runectx change assess-intake` first if the scope or target shape is ambiguous
- create/shape the new RuneContext change
- use decomposition commands if the old folder should become an umbrella change plus sub-changes
- rewrite Agent OS assumptions to the final RuneContext-era meaning before finalizing the migrated files
- ensure the feature plan reflects the cross-cutting decisions already captured
- validate and capture assurance
- delete the migrated legacy folder once the replacement is complete

### Step 8: Rewrite Remaining Repo References

- update repo docs and instructions that still treat `agent-os/` as canonical
- make sure all canonical references point at `runecontext/project/`, `runecontext/standards/`, `runecontext/changes/`, `runecontext/specs/`, and `runecontext/decisions/`
- where docs discuss future RuneCode behavior, ensure they describe RuneCode-owned UX over bundled RuneContext behavior rather than asking end users to use `runectx` directly

### Step 9: Remove Legacy Canonical Usage

- remove migrated `agent-os/product/`
- remove migrated `agent-os/standards/`
- remove migrated `agent-os/specs/`
- retain only intentionally frozen historical archive material until explicitly handled

### Step 10: Remove Temporary Standard Migration Aliases

- if migrated RuneContext standards still retain `aliases` entries for `agent-os/standards/...`, remove them only after all canonical references and workflow dependencies are gone
- use `runectx standard update --path standards/<path>.md --replace-aliases` for the cleanup mutation
- run `runectx standard list`, `runectx validate --json`, and `runectx status --json`
- confirm any remaining `agent-os/standards/` mentions are intentionally historical-only

## Feature-Planning Swap Guidance

When migrating old planned feature folders, apply the deeper RuneContext integration details immediately where they are relevant.

Examples of what to swap during migration:

- replace assumptions that Agent OS remains the future planning/spec/standards substrate with RuneContext as the canonical substrate
- replace assumptions that RuneCode may later integrate with RuneContext as an optional enhancement with the new assumption that context-aware RuneCode features ship with their RuneContext integration complete
- replace assumptions that RuneCode users manage RuneContext manually with the new planning assumption of bundled-by-default RuneContext integration
- replace assumptions that plain mode or optional assurance is sufficient with the new planning assumption that RuneCode-managed repos require `verified` mode
- replace assumptions that RuneContext should become RuneCode-specific with the new planning assumption that generic machine-friendly RuneContext capabilities support a deeper RuneCode-owned integration layer
- replace assumptions that end users will use raw `runectx` commands during normal RuneCode operation with the new planning assumption that RuneCode owns the user-facing command surface and invokes RuneContext internally
- replace assumptions that consumer compatibility should be enforced by RuneContext-specific knowledge of RuneCode with the new planning assumption that RuneContext may emit generic advisory compatibility warnings while RuneCode performs the hard compatibility check and fail-closed enforcement

In general:

- rewrite future-state product and integration assumptions during migration
- preserve old assumptions only as historical context when useful
- do not preserve obsolete future-state assumptions as canonical meaning

## Generic Advisory Compatibility Warnings And RuneCode Hard Enforcement

Future planning in this repository should assume the following split of responsibilities.

### Generic advisory warning support in RuneContext

RuneContext may later support generic advisory consumer-compatibility warnings during upgrade flows.

Intended behavior:

- if a project declares compatibility hints for an external consumer/tool, RuneContext may surface an advisory warning during an upgrade preview/apply flow when the target project state could fall outside that consumer's declared compatibility envelope
- those warnings remain advisory only
- those warnings should remain generic and not bake RuneCode-specific semantics into RuneContext core behavior
- those warnings should not decide whether an upgrade is allowed

### Hard enforcement in RuneCode

RuneCode remains the authoritative enforcement point for RuneCode-managed project compatibility.

Intended behavior:

- RuneCode uses its bundled RuneContext companion and compatibility metadata to determine whether a project is in a directly supported, upgrade-only, or unsupported state
- if a project is upgraded out-of-band to a newer unsupported RuneContext structure/version, RuneCode fails closed on normal operation
- in that incompatible state, RuneCode should allow only safe diagnostic/remediation flows such as doctor, status, upgrade guidance, or a read-only compatibility screen if later desired
- RuneCode does not silently downgrade, reinterpret, or partially accept unsupported newer RuneContext project states

### Migration implication

When rewriting old planned feature folders, replace any future-state text that assumes:

- end users manually drive RuneContext upgrades during normal RuneCode use
- RuneContext is the hard enforcement point for RuneCode compatibility
- RuneCode can continue normal operation on unsupported newer RuneContext project states

with the intended final-state assumptions:

- RuneCode owns normal end-user UX and wraps RuneContext operations under the hood
- generic advisory compatibility warnings may exist in RuneContext
- hard compatibility enforcement remains in RuneCode

## Assurance Expectations For The Migration

Because this repository starts in `verified` mode, the migration should explicitly capture:

- the initial verified baseline for the repo's RuneContext adoption
- imported historical evidence for legacy `agent-os/` material when needed
- change receipts for migrated planned work
- promotion receipts when changes become durable specs or decisions
- verification receipts for major migration batches

Migration assurance rules:

- do not fake native verified provenance for pre-RuneContext history
- use backfill/imported-history evidence where historical continuity matters
- ensure major batch cutovers are reviewable and verifiable

## Deletion Policy

Converted `agent-os/` content should be deleted, but only as part of the same reviewed migration step that introduces the verified replacement.

Deletion checklist for each file or folder:

1. replacement exists under `runecontext/`
2. replacement validates
3. inbound refs are updated
4. assurance capture for the migration step is complete
5. the old path is no longer canonical for any repo workflow

## Success Criteria

The migration is complete when all of the following are true:

- `runecontext.yaml` is present and this repository is operating in verified mode
- canonical repo planning content lives under `runecontext/`
- `agent-os/product/` content has been replaced by `runecontext/project/`
- all active standards live under `runecontext/standards/`
- the old standards index has been replaced by bundles and optional inventory docs
- completed feature planning has been converted into durable `runecontext/specs/*.md`
- planned feature folders have been converted into `runecontext/changes/`
- repo docs and review instructions no longer treat `agent-os/` as canonical
- migrated RuneContext standards no longer need temporary `agent-os/standards/...` aliases, unless a specific alias is intentionally retained with documented rationale
- no current planning workflow depends on `agent-os/` except any intentionally frozen archive material

## Summary Decision

The repository should not start by blindly converting every `agent-os/specs/*` folder into a RuneContext change.

The correct order is:

1. establish RuneContext root config and verified assurance
2. migrate product docs
3. migrate standards
4. classify spec folders
5. import completed items as durable specs
6. convert planned items into changes in batches
7. update references and delete legacy `agent-os/` sources

That approach preserves reviewability, keeps assurance history clean, and avoids carrying the Agent OS folder model forward as a disguised duplicate inside RuneContext.
