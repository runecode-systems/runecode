---
applyTo: ".github/workflows/**/*.yml,justfile,flake.nix,flake.lock,.envrc,go.mod,runner/package.json"
---

Use these references for CI and tooling review comments:

- `/justfile`
- `/.github/workflows/ci.yml`
- `/docs/source-quality.md`
- `/.github/instructions/source-quality.instructions.md`
- `/CONTRIBUTING.md`

When reviewing changes in this scope, focus on:

- CI steps and local developer workflow stay aligned with `just ci` expectations.
- Runtime and tool version changes are intentional and compatible (Go, Node, Nix, just).
- Linux, macOS, and Windows matrix portability is preserved.
- Lockfile and flake metadata updates are deliberate and validated.
- New checks are deterministic and avoid hidden network or environment coupling when possible.
- Source-quality-specific policy for lint/baseline/config changes should follow `/.github/instructions/source-quality.instructions.md` so CI/tooling guidance does not drift from the dedicated source-quality review surface.

Raise issues when CI and local parity diverge in ways that hide failures until merge time.
