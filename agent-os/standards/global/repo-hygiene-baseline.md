# Repo Hygiene Baseline

- Normalize line endings for determinism: keep `.gitattributes` minimal (`* text=auto eol=lf`)
- Use `.editorconfig` as the formatting baseline (multi-language, minimal rules)
- Keep `.gitignore` current as tooling/languages are added (example: `node_modules/`, build output, `*.tsbuildinfo`)
- Keep `/.envrc` thin: `use flake` only
- Protect high-leverage surfaces via `/.github/CODEOWNERS` + required review (keep the list current as new sensitive paths are introduced)

Examples of high-leverage paths:
- `/flake.nix`, `/flake.lock`, `/.envrc`, `/justfile`, `/.github/workflows/*`, `/protocol/`, `/docs/trust-boundaries.md`, `/cmd/runecode-secretsd/`, `/cmd/runecode-auditd/`
