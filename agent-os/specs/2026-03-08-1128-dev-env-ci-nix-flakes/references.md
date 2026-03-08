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
- Optional binary cache (if CI times require it; must pin keys/substituters):
  - https://cachix.org/

## Related Specs

- `agent-os/specs/2026-03-08-1039-scaffold-ci-matrix/`
