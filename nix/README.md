# Nix Layout

`flake.nix` is the public entrypoint for Nix interactions in this repository.
The implementation details live under `nix/` so the canonical release builder is easier to review.

- `nix/release/metadata.nix` - authoritative release metadata: version, tag, binaries, and target matrix
- `nix/packages/release-artifacts.nix` - canonical unsigned release artifact derivation
- `nix/dev-shell.nix` - development shell definition
- `nix/checks.nix` - flake checks, including release-binary metadata validation and the canonical release artifact build on Linux
- `nix/scripts/build-release-artifacts.sh` - orchestrates deterministic archive creation inside the derivation
- `tools/releasebuilder/` - Go helper that writes deterministic zip archives and release manifests for the Nix build

The expensive `release-artifacts` flake check intentionally runs on `x86_64-linux`, which matches the GitHub-hosted release runner used by `release.yml`.
