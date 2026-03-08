# References for Dev Environment + CI Bootstrap (Nix Flakes)

## Reference Implementation Style

- Example `flake.nix` style (direnv + just shellHook):
  - https://raw.githubusercontent.com/ZebulonRouseFrantzich/chimera-bench/refs/heads/main/flake.nix

## Docs / Best Practices

- Nix flakes (including `nixConfig` and `--no-write-lock-file`):
  - https://nixos.org/manual/nix/stable/command-ref/new-cli/nix3-flake
- `direnv`:
  - https://direnv.net/
- `nix-direnv` (`use flake` integration):
  - https://github.com/nix-community/nix-direnv
- GitHub Actions Nix installer (example; pin versions):
  - https://github.com/DeterminateSystems/nix-installer-action
- GitHub Actions security hardening (pin third-party actions to SHAs):
  - https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions#using-third-party-actions
- GitHub Actions workflow permissions:
  - https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#permissions
- Nix store cache action:
  - https://github.com/DeterminateSystems/magic-nix-cache-action
- Optional binary cache (if CI times require it; must pin keys/substituters):
  - https://cachix.org/

## Related Specs

- `agent-os/specs/2026-03-08-1039-scaffold-ci-matrix/`
