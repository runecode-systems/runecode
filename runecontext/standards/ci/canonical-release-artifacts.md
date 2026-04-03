---
schema_version: 1
id: ci/canonical-release-artifacts
title: Canonical Release Artifacts
status: active
aliases:
    - agent-os/standards/ci/canonical-release-artifacts
---

# Canonical Release Artifacts

- Unsigned release assets come only from `nix build .#release-artifacts`
- GitHub Actions may sign, attest, and publish those files, but must not rebuild or rename them
- Keep release version, tag, binaries, and targets in Nix metadata
- Release docs and maintainer steps should point to the same Nix builder
- Maintainers verify the builder locally before tagging

```sh
nix build --no-link .#release-artifacts
```
