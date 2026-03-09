# References for Monorepo Scaffold + Package Boundaries (v0)

## Product Context

- **Mission:** `agent-os/product/mission.md`
- **Tech stack:** `agent-os/product/tech-stack.md`

## Repo Tooling Context (Already Implemented)

- **Dev shell:** `flake.nix`
- **Command surface:** `justfile`
- **CI entrypoint + constraints:** `.github/workflows/ci.yml`
- **High-leverage file ownership:** `.github/CODEOWNERS`

## Related Specs

- **Dev Environment + CI Bootstrap (Nix Flakes):** `agent-os/specs/2026-03-08-1128-dev-env-ci-nix-flakes/`
- **Protocol & Schema Bundle v0 (cross-boundary schemas/fixtures):** `agent-os/specs/2026-03-08-1039-protocol-schemas-v0/`

## Go TUI

- Bubble Tea (framework): https://github.com/charmbracelet/bubbletea

## Go / Node Tooling

- Go modules (module path + basics): https://go.dev/ref/mod
- `govulncheck` (optional dependency vulnerability scanning): https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck
- Node `package.json` `engines` field: https://docs.npmjs.com/cli/v10/configuring-npm/package-json#engines
- TypeScript `tsconfig.json` reference (for `rootDir`, `noEmit`, etc.): https://www.typescriptlang.org/tsconfig
- Git attributes (`.gitattributes`) reference: https://git-scm.com/docs/gitattributes

## Similar Implementations

None yet in this repo; follow conventional Go `cmd/` + `internal/` structure and keep the Node runner self-contained under `runner/`.
