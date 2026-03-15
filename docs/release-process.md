# RuneCode Release Process

This repository publishes official binaries through tag-driven GitHub Releases.
Maintainers should not create releases manually through the GitHub Releases UI.
The official path is:

1. update `nix/release/metadata.nix` so the version and tag are correct
2. push a signed `v*` tag that matches that metadata exactly
3. let `.github/workflows/release.yml` build the unsigned artifacts from `flake.nix`, then sign, attest, and publish the release
4. approve the protected `release` environment before the publish job runs

## What the workflow publishes

For each pushed release tag, the workflow:

- reruns `just ci`
- builds the canonical unsigned artifacts with `nix build .#release-artifacts`
- cross-compiles the Go binaries for Linux, macOS, and Windows inside that flake-defined build
- emits the final versioned per-platform archives, `SHA256SUMS`, and `runecode_<tag>_release-manifest.json` from the Nix output tree
- generates a release SBOM
- signs each primary asset with Sigstore cosign keyless signing via GitHub OIDC
- publishes GitHub build provenance attestations
- creates or updates the corresponding GitHub Release

## One-time repository setup

Complete these steps before pushing the first release tag.

### 1. GitHub Actions workflow permissions

In the repository settings:

- go to `Settings -> Actions -> General`
- set `Workflow permissions` to `Read and write permissions`

The release workflow needs `contents: write` to create releases and `id-token: write` plus `attestations: write` to sign and attest assets.
If your organization uses GitHub Actions policy controls, also confirm artifact attestations are enabled for this repository.

### 2. Create a protected `release` environment

In the repository settings:

- go to `Settings -> Environments`
- create an environment named `release`
- add at least one required reviewer
- restrict deployment branches/tags to release tags if you use that policy surface

The workflow already targets this environment for the publish job. Required reviewers create an explicit maintainer approval checkpoint before release publication.

### 3. Protect release tags

In the repository settings, add a tag protection rule or repository ruleset for `v*`.

Recommended policy:

- only trusted maintainers can create matching tags
- force pushes and tag deletion are restricted
- signed tags are required if your GitHub plan and ruleset support it

### 4. Protect `main`

Keep `main` protected so release tags are cut from reviewed, green commits.

Recommended policy:

- require pull request review
- require the CI workflow to pass
- require the DCO check
- restrict direct pushes to maintainers

### 5. Enable maintainer tag signing

Every maintainer who cuts releases should configure Git tag signing locally and use signed annotated tags.

Example with GPG:

```sh
git config user.signingkey <your-key-id>
git config commit.gpgsign true
git config tag.gpgSign true
```

If you prefer SSH signing, configure Git's SSH signing support instead.

## Per-release procedure

### 1. Update release metadata and start from an up-to-date `main`

Set the version once in `nix/release/metadata.nix`, then commit that change before tagging.

```sh
git checkout main
git pull --ff-only origin main
just ci
nix build --no-link .#release-artifacts
```

That `nix build` step verifies the canonical release builder locally before you tag.

### 2. Create a signed annotated release tag

```sh
TAG="$(nix eval --raw .#lib.release.tag)"
git tag -s "$TAG" -m "RuneCode $TAG"
```

For prereleases, set `version = "0.1.0-alpha.1"` in `nix/release/metadata.nix`; the derived tag becomes `v0.1.0-alpha.1`, and the workflow marks any tag containing `-` as a GitHub prerelease.

The workflow fails closed if the pushed tag does not match `nix eval .#lib.release.tag`.

### 3. Push the tag

```sh
git push origin "$TAG"
```

That tag push is the release trigger. Do not create the release manually in the UI.

### 4. Approve the protected publish step

When the workflow reaches the `release` environment gate:

- review the workflow run
- confirm it is running for the expected tag and commit
- approve the environment so the publish job can sign, attest, and create the release

### 5. Verify the published release

After the workflow completes:

- open the GitHub Release page for the tag
- confirm the expected archives, `SHA256SUMS`, `runecode_<tag>_release-manifest.json`, signatures, certificates, and SBOM are present
- run the verification flow in `docs/install-verify.md` against one release asset

## Failure handling

If a release build fails before publication, fix the issue and cut a new tag.
Do not silently retag and reuse the same version.

If a published release is bad:

- publish a follow-up release with a new version
- document the superseded release clearly in the release notes
- avoid mutating release history unless there is a security incident that requires stronger remediation

## Notes for future expansion

This initial release flow ships the current Go binaries only.
When the untrusted runner has a supported SEA release artifact, extend the workflow and verification docs to include it as an additional signed and attested release asset.
