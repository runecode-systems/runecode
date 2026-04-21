---
schema_version: 1
id: ci/canonical-release-artifacts
title: Canonical Release Artifacts
status: active
---

# Canonical Release Artifacts

- Unsigned release assets come only from `nix build .#release-artifacts`
- GitHub Actions may sign, attest, and publish those files, but must not rebuild or rename them
- Keep release version, tag, binaries, and targets in Nix metadata
- Release docs and maintainer steps should point to the same Nix builder
- Maintainers verify the builder locally before tagging
- If the `buildGoModule` fixed-output `vendorHash` becomes stale, refresh it explicitly in the repo rather than teaching CI to self-heal it
- The canonical repo-local refresh path for the release-artifacts package is `just refresh-release-vendor-hash` backed by `go run ./tools/releasebuilder refresh-vendor-hash`

```sh
nix build --no-link .#release-artifacts
just refresh-release-vendor-hash
```
