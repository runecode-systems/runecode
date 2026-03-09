# Standards for Monorepo Scaffold + Package Boundaries (v0)

These standards apply to implementation work produced from this spec.

---

## product/roadmap-conventions

# Product Roadmap Conventions

Applies to: `agent-os/product/roadmap.md`

This roadmap is the canonical view of what is planned next (as specs) and what has shipped (as releases).

## Structure

- Keep a very short intro at the top (2-5 lines) explaining how to read/maintain the roadmap.
- Required sections (in this order):
  - `## Upcoming Features`
  - `## Unscheduled (Needs Specs)`
  - `## Completed Features`

## Version Grouping

- Group roadmap items under version headings one level below the section heading (H3).
- Use `### vNext (Planned)` for work that is planned but not yet assigned a concrete version.

## Spec Entry Format

- Each spec entry is a checkbox with:
  - Spec title
  - Spec folder path (in backticks)
  - A short, user-visible description (1-2 lines)

Template:

```md
- [ ] Spec Title (`agent-os/specs/YYYY-MM-DD-HHMM-spec-slug/`)
  - Short description of the user-visible outcome.
```

Rules:

- Upcoming work uses `- [ ]`.
- Completed work uses `- [x]`.
- Reference specs by title + spec folder path (do not use numeric spec IDs).
- Keep descriptions outcome-focused (what changes for the user), not implementation notes.

## Moving Items On Release

- When a version is released:
  - Mark all items in that version block as `- [x]`.
  - Move the entire version block from `## Upcoming Features` to `## Completed Features`.
  - Keep Completed ordered newest-first.

## Converting Unscheduled Items Into Specs

- If an item exists under `## Unscheduled (Needs Specs)` and a spec is created for it:
  - Replace the unscheduled item with a proper spec entry under the target version group.
  - Remove the duplicate unscheduled checkbox.

---

## CI + Portability (Repo Convention)

- `just ci` is the canonical entrypoint and must be runnable on Windows without Nix and without a bash dependency.
- `just ci` is check-only and must not modify the worktree (including lockfiles like `flake.lock`, `go.sum`, and `package-lock.json`).
- Keep `justfile` recipes simple and cross-platform (avoid unix-only tools and bashisms). Prefer language-native commands and small helper programs over shell pipelines.
- For MVP clarity, `just ci` may keep an explicit runner lint step even when `npm test` also invokes lint.

## Trust Boundary Guardrails (Scaffold Standard)

- The "untrusted scheduler" boundary is not purely documentary.
- The TS runner must have a mechanical boundary check that fails CI if runner source imports/references trusted Go areas (`cmd/`, `internal/`) or otherwise escapes `runner/` (except allowed access to `protocol/` schemas/fixtures).
- Boundary-check scan scope must cover runner JS/TS source across `runner/` (not only `runner/src/`) while excluding dependency/build directories and boundary-check tooling/test files.
- Boundary-check path handling must include Unix absolute paths and Windows drive-letter/UNC absolute paths.
- Boundary-check matching must avoid false positives for unrelated third-party package names containing substrings like `/internal/` or `/cmd/`; enforcement should be based on repo-root references and resolved path containment.
- Boundary-check violation messages should report runner-relative paths using forward slashes for deterministic behavior across operating systems.
- Boundary-check behavior must fail closed if no runner source files are discovered.
- Boundary-check implementation is a best-effort static guardrail; runtime trust enforcement remains with broker auth/schema validation, deterministic policy decisions, and isolation backends.
- Cross-boundary artifacts live in `protocol/` and must be schema-validated at consumption time by the enforcing component (broker/policy/launcher).

## Repo Hygiene (Scaffold Standard)

- `.gitignore` must be kept current as new languages/tooling are introduced (minimum: `node_modules/`, build output dirs, and compiler metadata like `*.tsbuildinfo`).
- Security-critical surfaces introduced by scaffolding must be protected by `CODEOWNERS` + required review (minimum: `/protocol/`, `/docs/trust-boundaries.md`, and the secrets/audit daemons).
- Prefer adding a minimal `.editorconfig` once the repo becomes multi-language to reduce formatting churn.

## Version Compatibility (CI Reality)

- Go and Node version expectations must be explicit in package metadata where possible:
  - Go module path is stable and does not change after initial scaffold.
  - Node runner declares supported Node versions via `package.json` `engines` and remains compatible with the CI matrix.
