# Nix Flake CI Invariants

- CI sets explicit Nix cache allowlist via `NIX_CONFIG`:
  - `substituters = https://cache.nixos.org`
  - `trusted-public-keys = cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY=`
- Any non-default cache must be explicitly added with its public key pinned
- CI validates lock metadata before checks: `nix flake lock --no-update-lock-file`
- CI must not write `flake.lock`:
  - Use `--no-write-lock-file` for flake commands
  - Fail if `flake.lock` diffs before/after

```yaml
env:
  NIX_CONFIG: |
    substituters = https://cache.nixos.org
    trusted-public-keys = cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY=

steps:
  - run: nix flake lock --no-update-lock-file
  - run: nix flake check --no-write-lock-file
  - run: nix develop --no-write-lock-file -c just ci
```
