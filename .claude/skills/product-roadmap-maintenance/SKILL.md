# Skill: product-roadmap-maintenance

Keep `runecontext/project/roadmap.md` up to date as the human-facing product summary while canonical lifecycle state moves through RuneContext changes and specs.

## Standards to read first

- `runecontext/standards/product/roadmap-conventions.md`

## When to use

- After creating a new canonical RuneContext change or spec that should be reflected in the roadmap summary
- When a planned or completed item changes scope/title and the roadmap summary should be updated
- When a version is released and its block should be moved to Completed

## Inputs

- Optional: one or more canonical RuneContext change or spec paths (examples: `runecontext/changes/CHG-2026-002-33c5-git-gateway-commit-push-pr/`, `runecontext/specs/protocol-schema-bundle-v0.md`)
- Optional: a target version label (example: `v0.1.0`)

## Procedure

### A) Add (or update) change/spec entries in Upcoming

1. Resolve canonical RuneContext paths:
   - Prefer paths provided by the user.
   - If none are provided, look for recent items under `runecontext/changes/` and `runecontext/specs/` and select the most likely candidates.
   - If the likely source path is still ambiguous, ask the user for the change/spec path(s) and stop.

2. For each canonical path, extract:
   - Title: use the change/spec title from the canonical file set.
   - Short description: prefer the stated outcome/summary; otherwise write 1-2 lines describing the user-visible result.

3. Decide the target version group under `## Upcoming Features`:
   - If the user provided a version, use it.
   - Otherwise default to `### vNext (Planned)` and mention that assumption.

4. Update `runecontext/project/roadmap.md`:
   - Ensure the required sections exist: `## Upcoming Features`, `## Unscheduled (Needs Specs)`, `## Completed Features`.
   - Ensure the target version heading exists under Upcoming; create it if needed.
   - Add the entry using the standard template:

```md
- Feature Title
  - Short description of the user-visible outcome.
```

   - Avoid duplicates:
      - If the same feature already exists in the roadmap, update the title/description in place.
      - Ensure the same item is not listed in both Upcoming and Completed.

5. If an equivalent item exists under `## Unscheduled (Needs Specs)`:
   - Remove the unscheduled duplicate after adding the canonical change/spec entry.

### B) Mark a version released and move it to Completed

1. Determine the version to release:
   - If the user provides a version label, use it.
   - If not provided, ask for the version label and stop.

2. In `runecontext/project/roadmap.md`, find the version block under `## Upcoming Features`.

4. Move the entire version block (heading + items) to `## Completed Features`:
    - Keep Completed ordered newest-first.
    - Do not drop descriptions or any existing canonical RuneContext references when moving.

## Verification

- Re-read `runecontext/project/roadmap.md` and confirm:
   - Sections and headings follow the standard.
   - No duplicate feature entries appear across Upcoming/Completed.
   - The roadmap remains a summary rather than a lifecycle source of truth.

## Guardrails

- Do not invent RuneContext change IDs or spec paths.
- If version assignment is ambiguous, default to `vNext (Planned)` and state the assumption.
- Do not commit or push changes unless explicitly requested.
