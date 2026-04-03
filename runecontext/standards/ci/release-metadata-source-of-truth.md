---
schema_version: 1
id: ci/release-metadata-source-of-truth
title: Release Metadata Source Of Truth
status: active
aliases:
    - agent-os/standards/ci/release-metadata-source-of-truth
---

# Release Metadata Source Of Truth

- Define release version, derived tag, binaries, and targets in `nix/release/metadata.nix`
- Do not independently define release version/tag data in workflows, docs, or helper scripts
- Release jobs must verify the pushed tag matches `nix eval --raw .#lib.release.tag`
- Keep release-shape changes reviewable as metadata edits first

```nix
version = "0.1.0-alpha.1";
tag = "v${base.version}";
```
