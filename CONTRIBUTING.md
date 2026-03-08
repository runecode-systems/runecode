# Contributing to RuneCode

Thanks for your interest in contributing.

## License

RuneCode is licensed under Apache-2.0. By contributing, you agree that your
contributions will be licensed under the same license.

## Developer Certificate of Origin (DCO) (Required)

RuneCode uses the Developer Certificate of Origin (DCO) instead of a CLA.
Every commit in a pull request must include a `Signed-off-by:` line.

To add it when committing:

```sh
git commit -s
```

Example sign-off line:

```
Signed-off-by: Jane Smith <jane.smith@example.com>
```

The DCO text is in `DCO` and at https://developercertificate.org/.

### Fixing missing sign-offs

If you forgot to sign off:

- Last commit only:

```sh
git commit --amend -s
```

- Multiple commits on your branch (one common approach):

```sh
git rebase --signoff origin/main
```

## DCO Enforcement

We enforce DCO on pull requests using the GitHub-side DCO check (no CLA).
PRs will not be merged unless all commits are signed off.

Maintainers should:

- Install the DCO GitHub App: https://github.com/apps/dco
- Require the DCO check in branch protection rules
- Enable GitHub's "Require contributors to sign off on web-based commits"

## Dev Environment

The canonical local workflow uses Nix + `just`:

- Prerequisite: Nix `>= 2.18`
- Optional auto-entry: `direnv` + `nix-direnv`
- Canonical command surface: `just`
- CI runs the same logical checks as `just ci`

### Use the dev shell manually

```sh
nix develop
just --list
just ci
```

### Enable auto-entry with direnv

1. Install `direnv` and `nix-direnv` on your host machine.
2. Add the direnv shell hook for your shell (`bash`, `zsh`, `fish`, etc.).
3. In the repo root, run:

```sh
direnv allow
```

Entering the repository directory auto-loads the flake shell (`use flake` from `.envrc`), and leaving the directory unloads it.

### Trust model

Treat changes to `flake.nix`, `flake.lock`, and `.envrc` as high-trust changes. They control local tooling execution and are reviewed carefully.

### If you get stuck

- Stop auto-loading for this repo: `direnv deny`
- Fallback to manual shell entry: `nix develop`
- Clear cached direnv environment: remove `.direnv/` and run `direnv allow` again

## Code of Conduct

This project follows the Contributor Covenant Code of Conduct.
See `CODE_OF_CONDUCT.md`.

## Submitting a Pull Request

- Fork the repo and create a feature branch.
- Keep changes focused and well-described.
- Ensure your commits are signed off (`git commit -s`).
- Ensure tests/lint pass for the areas you changed.

If you are unsure about a design direction, open an issue first.
