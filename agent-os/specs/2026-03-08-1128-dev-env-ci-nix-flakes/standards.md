# Standards for Dev Environment + CI Bootstrap (Nix Flakes)

These standards apply to implementation work produced from this spec.

## Supply Chain and Execution Safety

- Treat these files as high-leverage and protect them via `CODEOWNERS` + required review: `flake.nix`, `flake.lock`, `.envrc`, `justfile`, `.github/CODEOWNERS`, `.github/workflows/*`.
- GitHub Actions workflows must declare least-privilege token permissions (default target: `permissions: contents: read`).
- All GitHub Actions `uses:` dependencies must be pinned to full commit SHAs (do not use floating tags alone).
- CI must validate `flake.lock` metadata before executing flake checks (`nix flake lock --no-update-lock-file`).
- CI must not mutate `flake.lock` (`--no-write-lock-file`) and must fail the job if `flake.lock` changes.
- CI must use an explicit Nix substituter/key allowlist (prefer setting `substituters`/`trusted-public-keys` via `NIX_CONFIG` or `--option`, not only `extra-*`). Default allowlist:
  - `https://cache.nixos.org`
  - `cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY=`
  Any non-default cache must be explicitly enumerated and its public key pinned.
- Keep `.envrc` thin (`use flake` only). Do not embed arbitrary shell logic in `.envrc`.
- Keep `shellHook` minimal: no network calls, no installers, no secret reads, and no mutation outside the repo. If it prints banners/help text, gate that output to interactive shells only.

## Reproducibility and Diagnostics

- Pin workflow tooling versions for reproducibility in native (non-Nix) jobs (for example, avoid `@latest` for `go install` commands).
- If Nix store caching actions are used, they must also be SHA-pinned and compatible with the explicit substituter/key allowlist policy.
- Log Nix + key tool versions early in CI for debuggability.
- `nix flake check` must include meaningful `checks` outputs (minimum: dev shell build + Nix formatting check).

## Portability

- `just ci` is the canonical entrypoint and must be runnable on Windows without Nix (or Windows CI must run the equivalent underlying commands explicitly and document any delta).
